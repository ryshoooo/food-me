package handlers

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"io"
	"net"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/xdg-go/scram"
)

type PostgresHandler struct {
	Address       string
	Username      string
	Password      string
	Logger        *logrus.Logger
	LogUpstream   bool
	LogDownstream bool

	Database string
}

func NewPostgresHandler(address, username, password string, logger *logrus.Logger, logUpstream, logDownstream bool) *PostgresHandler {
	return &PostgresHandler{
		Address:       address,
		Username:      username,
		Password:      password,
		Logger:        logger,
		LogUpstream:   logUpstream,
		LogDownstream: logDownstream,
	}
}

func (h *PostgresHandler) Handle(conn net.Conn) error {
	defer conn.Close()

	destination, err := net.Dial("tcp", h.Address)
	if err != nil {
		h.Logger.Errorf("Unable to connect to destination: %v", err)
		return err
	}
	defer destination.Close()

	err = h.Startup(conn, destination)
	if err != nil {
		h.Logger.Errorf("Error on startup: %v", err)
		return err
	}

	err = h.Authenticate(conn, destination)
	if err != nil {
		h.Logger.Errorf("Error on authentication: %v", err)
		return err
	}
	go h.PipeForever(destination, conn, "postgres")
	h.PipeClientNicely(conn, destination)

	return nil
}

func (h *PostgresHandler) PipeForever(upstream, client net.Conn, upstreamName string) {
	if h.LogUpstream {
		buffer := make([]byte, 1024)
		for {
			n, err := upstream.Read(buffer)
			if err != nil {
				if err != io.EOF {
					h.Logger.Errorf("Error reading from upstream: %v", err)
				}
				break
			}
			h.Logger.Debugf("Read %v bytes from %s: %v; %s", n, upstreamName, buffer[:n], buffer[:n])
			client.Write(buffer[:n])
		}
	} else {
		io.Copy(client, upstream)
	}
}

func (h *PostgresHandler) PipeClientNicely(client, dest net.Conn) {
	for {
		op, size, data, err := h.ReadClientMessage(client)
		if err == io.EOF {
			h.Logger.Info("Client closed connection")
			break
		}
		if err != nil {
			h.Logger.Errorf("Error reading from client: %v", err)
			break
		}
		// TODO: Possible data manipulation :)
		dest.Write(append(op, append(size, data...)...))
	}
}

func (h *PostgresHandler) Startup(conn, dest net.Conn) error {
	h.Logger.Info("Commencing startup")
	startup, err := h.Read(conn, 8, "client")
	if err != nil {
		return err
	}
	h.Logger.Debugf("Read startup packet from client: %v", startup)
	err = h.Write(dest, startup, "db")
	if err != nil {
		return err
	}
	resp, err := h.Read(dest, 1, "db")
	if err != nil {
		return err
	}
	h.Logger.Debugf("Read startup response from db: %v", resp)
	if resp[0] != 'N' {
		return fmt.Errorf("unexpected response from db: %v", resp)
	}
	err = h.Write(conn, resp, "client")
	if err != nil {
		return err
	}
	h.Logger.Info("Startup successful")
	return nil
}

func (h *PostgresHandler) Write(dest net.Conn, data []byte, name string) error {
	h.Logger.Debugf("Writing data to %s: %v", name, data)
	n, err := dest.Write(data)
	if err != nil || n != len(data) {
		h.Logger.Errorf("Error writing to %s: %v", name, err)
		return err
	}
	return nil
}

func (h *PostgresHandler) Read(conn net.Conn, size int, name string) ([]byte, error) {
	h.Logger.Debugf("Reading %v bytes from %s", size, name)
	buff := make([]byte, size)
	n, err := conn.Read(buff)
	if err != nil || n != size {
		if err == io.EOF {
			return nil, err
		}
		h.Logger.Errorf("Error reading from %s: %v", name, err)
		return nil, err
	}
	return buff, nil
}

func (h *PostgresHandler) Authenticate(conn, dest net.Conn) error {
	h.Logger.Info("Commencing authentication")
	sizebuff, err := h.Read(conn, 4, "client")
	if err != nil {
		return err
	}

	size := calculatePacketSize(sizebuff)
	auth, err := h.Read(conn, size-4, "client")
	if err != nil {
		return err
	}
	h.Logger.Debugf("Read authentication packet from client: %v (%s)", auth, auth)

	parts := bytes.Split(auth, []byte{0})
	if len(parts) < 7 {
		return fmt.Errorf("invalid authentication packet: %v", parts)
	}
	u := string(parts[3])
	uv := string(parts[4])
	d := string(parts[5])
	dv := string(parts[6])
	h.Logger.Debugf("Authentication: %v=%v %v=%v", u, uv, d, dv)

	uvs := strings.Split(uv, ";")
	if len(uvs) < 2 {
		h.Logger.Info("Username does not contain OIDC data, proxy all the requests going forward")
		h.Write(dest, sizebuff, "db")
		h.Write(dest, auth, "db")
		return nil
	}

	h.Logger.Debugf("OIDC data: %v", uvs)
	var accessToken string
	var refreshToken string
	for _, ov := range uvs {
		if strings.HasPrefix(ov, "access_token=") {
			accessToken = strings.Split(ov, "=")[1]
		}
		if strings.HasPrefix(ov, "refresh_token=") {
			refreshToken = strings.Split(ov, "=")[1]
		}
	}

	h.Logger.Debugf("Access token: %v", accessToken)
	h.Logger.Debugf("Refresh token: %v", refreshToken)
	if accessToken == "" || refreshToken == "" {
		h.Logger.Info("Access token or refresh token is missing, proxy all the requests going forward")
		h.Write(dest, sizebuff, "db")
		h.Write(dest, auth, "db")
		return nil
	}

	h.Database = dv

	// TODO: Verify tokens

	// Authenticate as the configured user
	err = h.auth(dest)
	if err != nil {
		return err
	}

	// Send OK to client
	err = h.Write(conn, []byte{82, 0, 0, 0, 8, 0, 0, 0, 0}, "client")
	if err != nil {
		return err
	}

	return nil
}

func (h *PostgresHandler) ReadMessage(conn net.Conn, expR byte) ([]byte, error) {
	r, err := h.Read(conn, 1, "db")
	if err != nil {
		return nil, err
	}
	if r[0] != expR {
		return nil, fmt.Errorf("unexpected response from db: %v %s", r, r)
	}
	r, err = h.Read(conn, 4, "db")
	if err != nil {
		return nil, err
	}
	rsize := calculatePacketSize(r)
	r, err = h.Read(conn, rsize-4, "db")
	if err != nil {
		return nil, err
	}
	return r, nil
}

func (h *PostgresHandler) ReadClientMessage(conn net.Conn) ([]byte, []byte, []byte, error) {
	operation, err := h.Read(conn, 1, "client")
	if err != nil {
		return nil, nil, nil, err
	}
	size, err := h.Read(conn, 4, "client")
	if err != nil {
		return nil, nil, nil, err
	}
	sizeInt := calculatePacketSize(size)
	data, err := h.Read(conn, sizeInt-4, "client")
	if err != nil {
		return nil, nil, nil, err
	}
	if h.LogDownstream {
		h.Logger.Debugf("Operation: %v (%s); Read %v bytes from client: %s", operation, operation, sizeInt, data)
	}
	return operation, size, data, nil
}

func (h *PostgresHandler) auth(dest net.Conn) error {
	h.Logger.Info("Authenticating as configured user")

	// Send initial auth request
	msg := []byte{0, 3, 0, 0}
	msg = append(msg, []byte("user")...)
	msg = append(msg, 0)
	msg = append(msg, []byte(h.Username)...)
	msg = append(msg, 0)
	msg = append(msg, []byte("database")...)
	msg = append(msg, 0)
	msg = append(msg, []byte(h.Database)...)
	msg = append(msg, []byte{0, 0}...)
	size := createPacketSize(len(msg) + 4)
	msg = append(size, msg...)
	err := h.Write(dest, msg, "db")
	if err != nil {
		return err
	}

	// Read auth response challenge
	r, err := h.ReadMessage(dest, 'R')
	if err != nil {
		return err
	}

	// Check trust auth method
	if checkAuthenticationSuccess(r) {
		h.Logger.Info("Trust auth method reply. Authentication successful")
		return nil
	}

	// Determine the method
	rs := bytes.Split(r, []byte{0})
	if len(rs) < 3 {
		return fmt.Errorf("unexpected response from db: %v", rs)
	}
	h.Logger.Debugf("Auth method: %v", rs)
	method := rs[3][0]
	switch int(method) {
	case 3:
		h.Logger.Info("Clear password auth method")
		return h.handleClearPasswordAuth(dest)
	case 5:
		h.Logger.Info("MD5 password auth method")
		return h.handleMD5PasswordAuth(dest, string(rs[3][1:]))
	case 7:
	case 8:
		h.Logger.Info("GSSAPI auth method")
		return fmt.Errorf("GSSAPI auth method not supported")
	case 10:
		h.Logger.Info("SCRAM-SHA-256 auth method")
		return h.handleSCRAMSHA256Auth(dest)
	default:
		return fmt.Errorf("unknown auth method: %v", method)
	}

	return nil
}

func (h *PostgresHandler) handleClearPasswordAuth(dest net.Conn) error {
	msg := []byte{'p'}
	pwd := []byte(h.Password)
	s := createPacketSize(len(pwd) + 5)
	msg = append(msg, s...)
	msg = append(msg, pwd...)
	msg = append(msg, 0)
	h.Write(dest, msg, "db")

	// Read auth response
	r, err := h.ReadMessage(dest, 'R')
	if err != nil {
		return err
	}

	if !checkAuthenticationSuccess(r) {
		return fmt.Errorf("authentication failed, response from db: %v", r)
	}

	h.Logger.Info("Clear password auth successful")
	return nil
}

func (h *PostgresHandler) handleMD5PasswordAuth(dest net.Conn, key string) error {
	msg := []byte{'p'}

	// Calculate the MD5 hash
	md5H := md5.New()
	_, err := md5H.Write([]byte(h.Password + h.Username))
	if err != nil {
		return err
	}
	pwd1 := fmt.Sprintf("%x", md5H.Sum(nil))
	md5H.Reset()
	_, err = md5H.Write([]byte(pwd1 + key))
	if err != nil {
		return err
	}
	pwd := fmt.Sprintf("md5%x", md5H.Sum(nil))

	// Send the password
	s := createPacketSize(len(pwd) + 5)
	msg = append(msg, s...)
	msg = append(msg, []byte(pwd)...)
	msg = append(msg, 0)
	h.Write(dest, msg, "db")

	// Handle response
	r, err := h.ReadMessage(dest, 'R')
	if err != nil {
		return err
	}

	if !checkAuthenticationSuccess(r) {
		return fmt.Errorf("authentication failed, response from db: %v", r)
	}
	h.Logger.Info("MD5 password auth successful")
	return nil
}

func (h *PostgresHandler) handleSCRAMSHA256Auth(dest net.Conn) error {
	msg := []byte{'p'}

	// First step
	client, err := scram.SHA256.NewClient(h.Username, h.Password, "")
	if err != nil {
		return err
	}
	conv := client.NewConversation()
	var resp string
	firstMsg, err := conv.Step(resp)
	if err != nil {
		return err
	}
	h.Logger.Debugf("First step: %v", firstMsg)

	// Send the first step to the db
	pwd := []byte("SCRAM-SHA-256")
	pwd = append(pwd, 0)
	firstStepSize := createPacketSize(len(firstMsg))
	pwd = append(pwd, firstStepSize...)
	pwd = append(pwd, []byte(firstMsg)...)
	size := createPacketSize(len(pwd) + 4)
	msg = append(msg, size...)
	msg = append(msg, pwd...)
	h.Write(dest, msg, "db")

	// Get the data for second step
	r, err := h.ReadMessage(dest, 'R')
	if err != nil {
		return err
	}
	rs := bytes.Split(r, []byte{0})
	if len(rs) < 4 {
		return fmt.Errorf("unexpected response from db: %v", rs)
	}
	if rs[3][0] != 11 {
		return fmt.Errorf("unexpected response from db: %v", rs)
	}
	resp = string(rs[3][1:])
	h.Logger.Debugf("First step response: %v", resp)

	// Second step
	secondMsg, err := conv.Step(resp)
	if err != nil {
		return err
	}
	h.Logger.Debugf("Second step: %v", secondMsg)

	// Send the second step to the db
	msg = []byte{'p'}
	size = createPacketSize(len(secondMsg) + 4)
	msg = append(msg, size...)
	msg = append(msg, []byte(secondMsg)...)
	h.Write(dest, msg, "db")

	// Get the data for the third step
	r, err = h.ReadMessage(dest, 'R')
	if err != nil {
		return err
	}
	rs = bytes.Split(r, []byte{0})
	if len(rs) < 4 {
		return fmt.Errorf("unexpected response from db: %v", rs)
	}
	if rs[3][0] != 12 {
		return fmt.Errorf("unexpected response from db: %v", rs)
	}
	resp = string(rs[3][1:])
	h.Logger.Debugf("Second step response: %v", resp)

	// Third step (validation)
	_, err = conv.Step(resp)
	if err != nil {
		return err
	}

	// Expecting success from the db
	r, err = h.ReadMessage(dest, 'R')
	if err != nil {
		return err
	}
	if !checkAuthenticationSuccess(r) {
		return fmt.Errorf("authentication failed, response from db: %v", r)
	}

	h.Logger.Info("SCRAM-SHA-256 auth successful")
	return nil
}

func calculatePacketSize(sizebuff []byte) int {
	return int(sizebuff[0])<<24 | int(sizebuff[1])<<16 | int(sizebuff[2])<<8 | int(sizebuff[3])
}

func createPacketSize(size int) []byte {
	return []byte{byte(size >> 24), byte(size >> 16), byte(size >> 8), byte(size)}
}

func checkAuthenticationSuccess(r []byte) bool {
	if len(r) != 4 {
		return false
	}
	if r[0] != 0 || r[1] != 0 || r[2] != 0 || r[3] != 0 {
		return false
	}
	return true
}
