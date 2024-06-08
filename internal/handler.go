package foodme

import (
	"fmt"

	"github.com/sirupsen/logrus"
)

func GetHandler(conf *Configuration, logger *logrus.Logger) (IHandler, error) {
	switch conf.DestinationDatabaseType {
	case "postgres":
		return NewPostgresHandler(
				conf.DestinationHost+":"+fmt.Sprint(conf.DestinationPort),
				conf.DestinationUsername,
				conf.DestinationPassword,
				logger,
				conf.DestinationLogUpstream,
				conf.DestinationLogDownstream,
				conf.OIDCEnabled,
				conf.OIDCClientID,
				conf.OIDCClientSecret,
				conf.OIDCTokenURL,
				conf.OIDCUserInfoURL,
			),
			nil
	default:
		return nil, fmt.Errorf("unknown destination database type: %s", conf.DestinationDatabaseType)
	}
}
