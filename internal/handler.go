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

func GetHandler(conf *Configuration, logger *logrus.Logger, httpClient IHttpClient) (IHandler, error) {
	upstreamHandler := &BasicUpstreamHandler{
		Address: conf.DestinationHost + ":" + fmt.Sprint(conf.DestinationPort),
	}

	var sqlHandler ISQLHandler
	if conf.PermissionAgentEnabled {
		pAgent, err := NewPermissionAgent(conf, httpClient)
		if err != nil {
			return nil, fmt.Errorf("failed to create permission agent: %w", err)
		}
		sqlHandler, err = NewSQLHandler(conf.DestinationDatabaseType, logger, pAgent)
		if err != nil {
			return nil, fmt.Errorf("failed to create SQL handler: %w", err)
		}
	}

	switch conf.DestinationDatabaseType {
	case "postgres":
		return NewPostgresHandler(
				conf.DestinationHost+":"+fmt.Sprint(conf.DestinationPort),
				conf.DestinationUsername,
				conf.DestinationPassword,
				upstreamHandler,
				logger,
				conf.DestinationLogUpstream,
				conf.DestinationLogDownstream,
				conf.OIDCEnabled,
				httpClient,
				conf.OIDCClientID,
				conf.OIDCClientSecret,
				conf.OIDCTokenURL,
				conf.OIDCUserInfoURL,
				conf.OIDCDatabaseFallBackToBaseClient,
				conf.OIDCDatabaseClients,
				conf.OIDCPostAuthSQLTemplate,
				sqlHandler,
				conf.ServerTLSEnabled,
				conf.ServerTLSCertificateFile,
				conf.ServerTLSCertificateKeyFile,
				conf.OIDCAssumeUserSession,
				conf.OIDCAssumeUserSessionUsernameClaim,
				conf.OIDCAssumeUserSessionAllowEscape,
			),
			nil
	default:
		return nil, fmt.Errorf("unknown destination database type: %s", conf.DestinationDatabaseType)
	}
}
