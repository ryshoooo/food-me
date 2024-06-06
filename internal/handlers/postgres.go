package handlers

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"strings"

	"github.com/sirupsen/logrus"
)

type PostgresHandler struct {
	Address     string
	Username    string
	Password    string
	Logger      *logrus.Logger
	LogUpstream bool

	Database string
}

func NewPostgresHandler(address, username, password string, logger *logrus.Logger, logUpstream bool) *PostgresHandler {
	return &PostgresHandler{
		Address:     address,
		Username:    username,
		Password:    password,
		Logger:      logger,
		LogUpstream: logUpstream,
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
	// h.PipeForever(conn, destination, "client")

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

	h.Logger.Debugf("Sizebuff %v", sizebuff)
	size := calculatePacketSize(sizebuff)
	h.Logger.Debugf("Computed size %v", size)

	auth, err := h.Read(conn, size-4, "client")
	if err != nil {
		return err
	}
	h.Logger.Debugf("Read authentication packet from client: %v (%s)", auth, auth)

	parts := bytes.Split(auth, []byte{0})
	h.Logger.Debugf("Split authentication packet: %v", parts)
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

	// TODO: Authenticate as the configured user
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
	r, err := h.Read(dest, 1, "db")
	if err != nil {
		return err
	}
	if r[0] != 'R' {
		return fmt.Errorf("unexpected response from db: %v %s", r, r)
	}
	r, err = h.Read(dest, 4, "db")
	if err != nil {
		return err
	}
	rsize := calculatePacketSize(r)
	r, err = h.Read(dest, rsize-4, "db")
	if err != nil {
		return err
	}

	// Check trust auth method
	if rsize == 8 && r[0] == 0 && r[1] == 0 && r[2] == 0 && r[3] == 0 {
		h.Logger.Info("Trust auth method reply. Authentication successful")
		return nil
	}

	// Determine the method
	rs := bytes.Split(r, []byte{0})
	if len(rs) < 3 {
		return fmt.Errorf("unexpected response from db: %v", rs)
	}
	method := rs[3][0]
	switch int(method) {
	case 3:
		h.Logger.Info("Clear password auth method")
		return h.handleClearPasswordAuth(dest)
	case 5:
		h.Logger.Info("MD5 password auth method")
		return nil
	case 7:
	case 8:
		h.Logger.Info("GSSAPI auth method")
		return fmt.Errorf("GSSAPI auth method not supported")
	case 10:
		h.Logger.Info("SCRAM-SHA-256 auth method")
		return nil
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
	r, err := h.Read(dest, 1, "db")
	if err != nil {
		return err
	}
	if r[0] != 'R' {
		return fmt.Errorf("unexpected response from db: %v %s", r, r)
	}
	r, err = h.Read(dest, 4, "db")
	if err != nil {
		return err
	}
	rsize := calculatePacketSize(r)
	r, err = h.Read(dest, rsize-4, "db")
	if err != nil {
		return err
	}

	if len(r) != 4 {
		return fmt.Errorf("unexpected response from db: %v", r)
	}
	if r[0] != 0 || r[1] != 0 || r[2] != 0 || r[3] != 0 {
		return fmt.Errorf("unexpected response from db: %v", r)
	}

	h.Logger.Info("Clear password auth successful")
	return nil
}

func calculatePacketSize(sizebuff []byte) int {
	return int(sizebuff[0])<<24 | int(sizebuff[1])<<16 | int(sizebuff[2])<<8 | int(sizebuff[3])
}

func createPacketSize(size int) []byte {
	return []byte{byte(size >> 24), byte(size >> 16), byte(size >> 8), byte(size)}
}
