package foodme

import (
	"fmt"
	"net"
	"net/http"

	"github.com/sirupsen/logrus"
)

type Server struct {
	Configuration *Configuration
	Logger        *logrus.Logger
}

func NewServer(conf *Configuration, logger *logrus.Logger) *Server {
	return &Server{Configuration: conf, Logger: logger}
}

func (s *Server) Start() error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%v", s.Configuration.ServerPort))
	if err != nil {
		return err
	}
	defer listener.Close()
	s.Logger.Infof("Listening for TCP connections at :%v", s.Configuration.ServerPort)
	httpClient := &http.Client{}
	return s.Listen(listener, httpClient)
}

func (s *Server) Listen(listener net.Listener, httpClient IHttpClient) error {
	for {
		conn, err := listener.Accept()
		if err != nil {
			s.Logger.WithField("component", "server").Errorf("Error accepting connection: %v", err)
			continue
		}

		handler, err := GetHandler(s.Configuration, s.Logger, httpClient)
		if err != nil {
			s.Logger.WithField("component", "server").Errorf("Error getting handler: %v", err)
			continue
		}

		s.Logger.Infof("Accepted connection: %s", conn.RemoteAddr().String())
		go func() {
			err := handler.Handle(conn)
			if err != nil {
				s.Logger.WithField("component", "server").Errorf("Error handling connection: %v", err)
			}
		}()
	}
}
