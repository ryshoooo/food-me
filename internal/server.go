package foodme

import (
	"fmt"
	"net"

	"github.com/sirupsen/logrus"
)

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

		s.Logger.Infof("Accepted connection: %s", conn.RemoteAddr().String())
		go func() {
			err := s.Handler.Handle(conn)
			if err != nil {
				s.Logger.WithField("component", "server").Errorf("Error handling connection: %v", err)
			}
		}()
	}
}
