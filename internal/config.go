package foodme

import (
	"os"

	"github.com/jessevdk/go-flags"
	"github.com/sirupsen/logrus"
)

type Configuration struct {
	// Log configuration
	LogLevel  string `long:"log-level" env:"LOG_LEVEL" default:"warn" choice:"trace" choice:"debug" choice:"info" choice:"warn" choice:"error" choice:"fatal" choice:"panic" description:"Log level"`
	LogFormat string `long:"log-format" env:"LOG_FORMAT" default:"pretty" choice:"text" choice:"json" choice:"pretty" description:"Log format"`

	// Destination
	DestinationHost          string `long:"destination-host" env:"DESTINATION_HOST" required:"true"`
	DestinationPort          int    `long:"destination-port" env:"DESTINATION_PORT" required:"true"`
	DestinationDatabaseType  string `long:"destination-database-type" env:"DESTINATION_DATABASE_TYPE" choice:"postgres" required:"true"`
	DestinationUsername      string `long:"destination-username" env:"DESTINATION_USERNAME"`
	DestinationPassword      string `long:"destination-password" env:"DESTINATION_PASSWORD"`
	DestinationLogUpstream   bool   `long:"destination-log-upstream" env:"DESTINATION_LOG_UPSTREAM"`
	DestinationLogDownstream bool   `long:"destination-log-downstream" env:"DESTINATION_LOG_DOWNSTREAM"`

	// Server
	ServerPort int `long:"port" env:"PORT" default:"2099"`
}

func NewConfiguration(args []string) (*Configuration, error) {
	c := &Configuration{LogLevel: ""}

	// Parse the command line arguments
	p := flags.NewParser(c, flags.Default)
	_, err := p.ParseArgs(args)
	if err != nil {
		return nil, err
	}

	return c, nil

}

func NewLogger(config *Configuration) *logrus.Logger {
	log := logrus.StandardLogger()
	logrus.SetOutput(os.Stdout)

	switch config.LogFormat {
	case "pretty":
		break
	case "json":
		logrus.SetFormatter(&logrus.JSONFormatter{})
	default:
		logrus.SetFormatter(&logrus.TextFormatter{
			DisableColors: true,
			FullTimestamp: true,
		})
	}

	switch config.LogLevel {
	case "trace":
		logrus.SetLevel(logrus.TraceLevel)
	case "debug":
		logrus.SetLevel(logrus.DebugLevel)
	case "info":
		logrus.SetLevel(logrus.InfoLevel)
	case "error":
		logrus.SetLevel(logrus.ErrorLevel)
	case "fatal":
		logrus.SetLevel(logrus.FatalLevel)
	case "panic":
		logrus.SetLevel(logrus.PanicLevel)
	default:
		logrus.SetLevel(logrus.WarnLevel)
	}

	return log
}
