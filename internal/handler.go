package foodme

import (
	"fmt"

	"github.com/ryshoooo/food-me/internal/handlers"
	"github.com/sirupsen/logrus"
)

func GetHandler(conf *Configuration, logger *logrus.Logger) (IHandler, error) {
	switch conf.DestinationDatabaseType {
	case "postgres":
		return handlers.NewPostgresHandler(
				conf.DestinationHost+":"+fmt.Sprint(conf.DestinationPort),
				conf.DestinationUsername,
				conf.DestinationPassword,
				logger,
				conf.DestinationLogUpstream,
			),
			nil
	default:
		return nil, fmt.Errorf("unknown destination database type: %s", conf.DestinationDatabaseType)
	}
}
