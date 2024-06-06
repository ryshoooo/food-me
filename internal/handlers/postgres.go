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
	return h.Write(conn, resp, "client")
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

	// TODO: Verify tokens

	// TODO: Authenticate as the configured user

	// TODO: Send OK to client

	return nil
}

func calculatePacketSize(sizebuff []byte) int {
	return int(sizebuff[0])<<24 | int(sizebuff[1])<<16 | int(sizebuff[2])<<8 | int(sizebuff[3])
}
