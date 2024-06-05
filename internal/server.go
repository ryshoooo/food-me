package foodme

import (
	"fmt"
	"net"

	"github.com/sirupsen/logrus"
)

type IServer interface {
	Start() error
	Listen(listener net.Listener) error
}

type IHandler interface {
	Handle(connection net.Conn) error
}

type Server struct {
	Port    int
	Logger  *logrus.Logger
	Handler IHandler
}

func NewServer(port int, logger *logrus.Logger, handler IHandler) *Server {
	return &Server{Port: port, Logger: logger, Handler: handler}
}

func (s *Server) Start() error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%v", s.Port))
	if err != nil {
		return err
	}
	defer listener.Close()
	s.Logger.Infof("Listening for TCP connections at :%v", s.Port)
	return s.Listen(listener)
}

func (s *Server) Listen(listener net.Listener) error {
	for {
		conn, err := listener.Accept()
		if err != nil {
			s.Logger.WithField("component", "server").Errorf("Error accepting connection: %v", err)
			continue
		}

		go s.Handler.Handle(conn)
	}
}
