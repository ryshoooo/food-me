package foodme

import (
	"fmt"
	"net"

	"github.com/sirupsen/logrus"
)

type BasicUpstreamHandler struct {
	Address string
}

func (h *BasicUpstreamHandler) Connect() (net.Conn, error) {
	return net.Dial("tcp", h.Address)
}

func GetHandler(conf *Configuration, logger *logrus.Logger) (IHandler, error) {
	upstreamHandler := &BasicUpstreamHandler{
		Address: conf.DestinationHost + ":" + fmt.Sprint(conf.DestinationPort),
	}

	switch conf.DestinationDatabaseType {
	case "postgres":
		dbc := make(map[string]*OIDCDatabaseClientSpec)
		for db, cid := range conf.OIDCDatabaseClientID {
			if spec, ok := dbc[db]; ok {
				spec.ClientID = cid
			} else {
				dbc[db] = &OIDCDatabaseClientSpec{ClientID: cid}
			}
		}
		for db, csec := range conf.OIDCDatabaseClientSecret {
			if spec, ok := dbc[db]; ok {
				spec.ClientSecret = csec
			} else {
				dbc[db] = &OIDCDatabaseClientSpec{ClientSecret: csec}
			}
		}
		return NewPostgresHandler(
				conf.DestinationHost+":"+fmt.Sprint(conf.DestinationPort),
				conf.DestinationUsername,
				conf.DestinationPassword,
				upstreamHandler,
				logger,
				conf.DestinationLogUpstream,
				conf.DestinationLogDownstream,
				conf.OIDCEnabled,
				conf.OIDCClientID,
				conf.OIDCClientSecret,
				conf.OIDCTokenURL,
				conf.OIDCUserInfoURL,
				conf.OIDCDatabaseFallBackToBaseClient,
				dbc,
				conf.OIDCAssumeUserSession,
				conf.OIDCAssumeUserSessionUsernameClaim,
				conf.OIDCAssumeUserSessionAllowEscape,
			),
			nil
	default:
		return nil, fmt.Errorf("unknown destination database type: %s", conf.DestinationDatabaseType)
	}
}
