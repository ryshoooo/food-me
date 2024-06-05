package handlers

import (
	"io"
	"net"

	"github.com/sirupsen/logrus"
)

type PostgresHandler struct {
	Address  string
	Username string
	Password string
	Logger   *logrus.Logger
}

func NewPostgresHandler(address, username, password string, logger *logrus.Logger) *PostgresHandler {
	return &PostgresHandler{
		Address:  address,
		Username: username,
		Password: password,
		Logger:   logger,
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
	go io.Copy(conn, destination)

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

	return nil
}

func (h *PostgresHandler) Startup(conn, dest net.Conn) error {
	startup, err := h.Read(conn, 8)
	if err != nil {
		h.Logger.Errorf("Error reading startup packet from source: %v", err)
		return err
	}
	h.Logger.Debugf("Read startup packet from source: %v", startup)
	return h.Write(dest, startup)
}

func (h *PostgresHandler) Write(dest net.Conn, data []byte) error {
	h.Logger.Debugf("Writing data to destination: %v", data)
	n, err := dest.Write(data)
	if err != nil || n != len(data) {
		h.Logger.Errorf("Error writing to destination: %v", err)
		return err
	}
	return nil
}

func (h *PostgresHandler) Read(conn net.Conn, size int) ([]byte, error) {
	h.Logger.Debugf("Reading %v bytes from source", size)
	buff := make([]byte, size)
	n, err := conn.Read(buff)
	if err != nil || n != size {
		h.Logger.Errorf("Error reading from source: %v", err)
		return nil, err
	}
	return buff, nil
}

func (h *PostgresHandler) Authenticate(conn, dest net.Conn) error {
	h.Logger.Info("Commencing authentication")
	sizebuff, err := h.Read(conn, 4)
	if err != nil {
		h.Logger.Errorf("Error reading authentication size from source: %v", err)
		return err
	}

	h.Logger.Debugf("Sizebuff %v", sizebuff)
	size := calculatePacketSize(sizebuff)
	h.Logger.Debugf("Computed size %v", size)

	auth, err := h.Read(conn, size-4)
	if err != nil {
		h.Logger.Errorf("Error reading authentication packet from source: %v", err)
		return err
	}
	h.Logger.Debugf("Read authentication packet from source: %v (%s)", auth, auth)

	return nil
}

func calculatePacketSize(sizebuff []byte) int {
	return int(sizebuff[0])<<24 | int(sizebuff[1])<<16 | int(sizebuff[2])<<8 | int(sizebuff[3])
}
