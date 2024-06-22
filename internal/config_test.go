package foodme

import (
	"testing"

	"github.com/sirupsen/logrus"
	"gotest.tools/v3/assert"
)

func TestNewConfigurationDefaults(t *testing.T) {
	// Empty should fail
	c, err := NewConfiguration([]string{})
	assert.Error(t, err, "the required flags `--destination-database-type', `--destination-host' and `--destination-port' were not specified")
	assert.Assert(t, c == nil)

	// Minimal should succeed
	c, err = NewConfiguration([]string{"--destination-database-type", "postgres", "--destination-host", "localhost", "--destination-port", "5432"})
	assert.NilError(t, err)
	assert.Equal(t, c.LogLevel, "warn")
	assert.Equal(t, c.LogFormat, "pretty")
	assert.Equal(t, c.DestinationHost, "localhost")
	assert.Equal(t, c.DestinationPort, 5432)
	assert.Equal(t, c.DestinationDatabaseType, "postgres")
	assert.Equal(t, c.DestinationUsername, "")
	assert.Equal(t, c.DestinationPassword, "")
	assert.Equal(t, c.DestinationLogUpstream, false)
	assert.Equal(t, c.DestinationLogDownstream, false)
	assert.Equal(t, c.OIDCEnabled, false)
	assert.Equal(t, c.OIDCClientID, "")
	assert.Equal(t, c.OIDCClientSecret, "")
	assert.Equal(t, c.OIDCTokenURL, "")
	assert.Equal(t, c.OIDCUserInfoURL, "")
	assert.Equal(t, c.EDatabaseClientID, "")
	assert.Equal(t, c.EDatabaseClientSecret, "")
	assert.Equal(t, c.OIDCDatabaseFallBackToBaseClient, false)
	assert.DeepEqual(t, c.OIDCDatabaseClientID, map[string]string{})
	assert.DeepEqual(t, c.OIDCDatabaseClientSecret, map[string]string{})
	assert.Equal(t, c.OIDCAssumeUserSession, false)
	assert.Equal(t, c.OIDCAssumeUserSessionUsernameClaim, "preferred_username")
	assert.Equal(t, c.OIDCAssumeUserSessionAllowEscape, false)
	assert.Equal(t, c.ServerPort, 2099)
	assert.Equal(t, c.ApiPort, 10000)
}

func TestNewConfigurationFull(t *testing.T) {
	c, err := NewConfiguration([]string{
		"--log-level", "debug",
		"--log-format", "json",
		"--destination-host", "myhost",
		"--destination-port", "7272",
		"--destination-database-type", "postgres",
		"--destination-username", "root",
		"--destination-password", "password",
		"--destination-log-upstream",
		"--destination-log-downstream",
		"--oidc-enabled",
		"--oidc-client-id", "client-id",
		"--oidc-client-secret", "client-secret",
		"--oidc-token-url", "http://token",
		"--oidc-user-info-url", "http://info",
		"--oidc-database-client-id", "postgres=pg-client-id,stuff=stuff-client-id,secretstuff=secretstuff-client-id",
		"--oidc-database-client-secret", "postgres=pg-secret,secretstuff=more-secret",
		"--oidc-database-fallback-to-base-client",
		"--oidc-assume-user-session",
		"--oidc-assume-user-session-username-claim", "db_role",
		"--oidc-assume-user-session-allow-escape",
		"--port", "9876",
		"--api-port", "8888",
	})
	assert.NilError(t, err)
	assert.Equal(t, c.LogLevel, "debug")
	assert.Equal(t, c.LogFormat, "json")
	assert.Equal(t, c.DestinationHost, "myhost")
	assert.Equal(t, c.DestinationPort, 7272)
	assert.Equal(t, c.DestinationDatabaseType, "postgres")
	assert.Equal(t, c.DestinationUsername, "root")
	assert.Equal(t, c.DestinationPassword, "password")
	assert.Equal(t, c.DestinationLogUpstream, true)
	assert.Equal(t, c.DestinationLogDownstream, true)
	assert.Equal(t, c.OIDCEnabled, true)
	assert.Equal(t, c.OIDCClientID, "client-id")
	assert.Equal(t, c.OIDCClientSecret, "client-secret")
	assert.Equal(t, c.OIDCTokenURL, "http://token")
	assert.Equal(t, c.OIDCUserInfoURL, "http://info")
	assert.Equal(t, c.EDatabaseClientID, "postgres=pg-client-id,stuff=stuff-client-id,secretstuff=secretstuff-client-id")
	assert.Equal(t, c.EDatabaseClientSecret, "postgres=pg-secret,secretstuff=more-secret")
	assert.Equal(t, c.OIDCDatabaseFallBackToBaseClient, true)
	assert.DeepEqual(t, c.OIDCDatabaseClientID, map[string]string{"postgres": "pg-client-id", "stuff": "stuff-client-id", "secretstuff": "secretstuff-client-id"})
	assert.DeepEqual(t, c.OIDCDatabaseClientSecret, map[string]string{"postgres": "pg-secret", "secretstuff": "more-secret"})
	assert.Equal(t, c.OIDCAssumeUserSession, true)
	assert.Equal(t, c.OIDCAssumeUserSessionUsernameClaim, "db_role")
	assert.Equal(t, c.OIDCAssumeUserSessionAllowEscape, true)
	assert.Equal(t, c.ServerPort, 9876)
	assert.Equal(t, c.ApiPort, 8888)
}

func TestBadMapping(t *testing.T) {
	_, err := NewConfiguration([]string{
		"--destination-database-type", "postgres",
		"--destination-host", "localhost",
		"--destination-port", "5432",
		"--oidc-database-client-id", "postgres=pg-client-id=somethingelse=just-wrong",
	})
	assert.Error(t, err, "invalid OIDC Database Client ID mapping: postgres=pg-client-id=somethingelse=just-wrong")

	_, err = NewConfiguration([]string{
		"--destination-database-type", "postgres",
		"--destination-host", "localhost",
		"--destination-port", "5432",
		"--oidc-database-client-secret", "postgres=pg-client-id=somethingelse=just-wrong",
	})
	assert.Error(t, err, "invalid OIDC Database Client Secret mapping: postgres=pg-client-id=somethingelse=just-wrong")
}

func TestNewLoggerFormatters(t *testing.T) {
	c, err := NewConfiguration([]string{"--destination-database-type", "postgres", "--destination-host", "localhost", "--destination-port", "5432"})
	assert.NilError(t, err)

	c.LogFormat = "pretty"
	logger := NewLogger(c)
	switch ft := logger.Formatter.(type) {
	case *logrus.TextFormatter:
		assert.Equal(t, ft.DisableColors, false)
		assert.Equal(t, ft.FullTimestamp, false)
	default:
		t.Fatalf("unexpected formatter type: %T", ft)
	}

	c.LogFormat = "json"
	logger = NewLogger(c)
	switch ft := logger.Formatter.(type) {
	case *logrus.JSONFormatter:
		assert.Assert(t, true)
	default:
		t.Fatalf("unexpected formatter type: %T", ft)
	}

	c.LogFormat = "unknown"
	logger = NewLogger(c)
	switch ft := logger.Formatter.(type) {
	case *logrus.TextFormatter:
		assert.Equal(t, ft.DisableColors, true)
		assert.Equal(t, ft.FullTimestamp, true)
	default:
		t.Fatalf("unexpected formatter type: %T", ft)
	}
}

func TestNewLoggerLevels(t *testing.T) {
	c, err := NewConfiguration([]string{"--destination-database-type", "postgres", "--destination-host", "localhost", "--destination-port", "5432"})
	assert.NilError(t, err)

	c.LogLevel = "trace"
	logger := NewLogger(c)
	assert.Equal(t, logger.Level, logrus.TraceLevel)

	c.LogLevel = "debug"
	logger = NewLogger(c)
	assert.Equal(t, logger.Level, logrus.DebugLevel)

	c.LogLevel = "info"
	logger = NewLogger(c)
	assert.Equal(t, logger.Level, logrus.InfoLevel)

	c.LogLevel = "error"
	logger = NewLogger(c)
	assert.Equal(t, logger.Level, logrus.ErrorLevel)

	c.LogLevel = "fatal"
	logger = NewLogger(c)
	assert.Equal(t, logger.Level, logrus.FatalLevel)

	c.LogLevel = "panic"
	logger = NewLogger(c)
	assert.Equal(t, logger.Level, logrus.PanicLevel)

	c.LogLevel = "unknown"
	logger = NewLogger(c)
	assert.Equal(t, logger.Level, logrus.WarnLevel)
}
