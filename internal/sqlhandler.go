package foodme

import (
	"fmt"

	"github.com/sirupsen/logrus"
)

func NewSQLHandler(databaseType string, logger *logrus.Logger, agent IPermissionAgent) (ISQLHandler, error) {
	switch databaseType {
	case "postgres":
		return NewPostgresSQLHandler(logger, agent), nil
	}
	return nil, fmt.Errorf("unknown database type: %s", databaseType)
}
