package handlers

import (
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
	_, err := net.Dial("tcp", h.Address)
	if err != nil {
		h.Logger.Errorf("Unable to connect to destination: %v", err)
		conn.Close()
		return err
	}
	return nil
}
