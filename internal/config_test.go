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
	assert.DeepEqual(t, c.OIDCDatabaseClients, map[string]*OIDCDatabaseClientSpec{})
	assert.Equal(t, c.OIDCPostAuthSQLTemplate, "")
	assert.Equal(t, c.PermissionAgentEnabled, false)
	assert.Equal(t, c.PermissionAgentType, "")
	assert.Equal(t, c.PermissionAgentOPAURL, "")
	assert.Equal(t, c.PermissionAgentOPASelectQueryTemplate, "data.{{ .TableName }}.allow == true")
	assert.Equal(t, c.PermissionAgentOPACreateQuery, "data.ddl_create.allow == true")
	assert.Equal(t, c.PermissionAgentOPAUpdateQuery, "data.ddl_update.allow == true")
	assert.Equal(t, c.PermissionAgentOPADeleteQuery, "data.ddl_delete.allow == true")
	assert.Equal(t, c.PermissionAgentOPAStringEscapeCharacter, "'")
	assert.Equal(t, c.ServerTLSEnabled, false)
	assert.Equal(t, c.ServerTLSCertificateFile, "")
	assert.Equal(t, c.ServerTLSCertificateKeyFile, "")
	assert.Equal(t, c.OIDCAssumeUserSession, false)
	assert.Equal(t, c.OIDCAssumeUserSessionUsernameClaim, "preferred_username")
	assert.Equal(t, c.OIDCAssumeUserSessionAllowEscape, false)
	assert.Equal(t, c.ServerPort, 2099)
	assert.Equal(t, c.ApiPort, 10000)
	assert.Equal(t, c.APITLSEnabled, false)
	assert.Equal(t, c.ApiUsernameLifetime, 3600)
	assert.Equal(t, c.ApiGarbageCollectionPeriod, 60)
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
		"--oidc-post-auth-sql-template", "../data/test_sql.sql",
		"--permission-agent-enabled",
		"--permission-agent-type", "opa",
		"--permission-agent-opa-url", "http://opa",
		"--permission-agent-opa-select-query-template", "data.{{ .TableName }}.blah == false",
		"--permission-agent-opa-create-query", "create query",
		"--permission-agent-opa-update-query", "update query",
		"--permission-agent-opa-delete-query", "delete query",
		"--permission-agent-opa-string-escape-character", "''",
		"--oidc-assume-user-session",
		"--oidc-assume-user-session-username-claim", "db_role",
		"--oidc-assume-user-session-allow-escape",
		"--server-tls-enabled",
		"--server-tls-certificate-file", "../data/cert.pem",
		"--server-tls-certificate-key-file", "../data/key.pem",
		"--port", "9876",
		"--api-port", "8888",
		"--api-tls-enabled",
		"--api-username-lifetime", "7200",
		"--api-garbage-collection-period", "6000",
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
	assert.DeepEqual(t, c.OIDCDatabaseClients, map[string]*OIDCDatabaseClientSpec{
		"postgres":    {ClientID: "pg-client-id", ClientSecret: "pg-secret"},
		"stuff":       {ClientID: "stuff-client-id"},
		"secretstuff": {ClientID: "secretstuff-client-id", ClientSecret: "more-secret"},
	})
	assert.Equal(t, c.OIDCPostAuthSQLTemplate, "../data/test_sql.sql")
	assert.Equal(t, c.PermissionAgentEnabled, true)
	assert.Equal(t, c.PermissionAgentType, "opa")
	assert.Equal(t, c.PermissionAgentOPAURL, "http://opa")
	assert.Equal(t, c.PermissionAgentOPASelectQueryTemplate, "data.{{ .TableName }}.blah == false")
	assert.Equal(t, c.PermissionAgentOPACreateQuery, "create query")
	assert.Equal(t, c.PermissionAgentOPAUpdateQuery, "update query")
	assert.Equal(t, c.PermissionAgentOPADeleteQuery, "delete query")
	assert.Equal(t, c.PermissionAgentOPAStringEscapeCharacter, "''")
	assert.Equal(t, c.OIDCAssumeUserSession, true)
	assert.Equal(t, c.OIDCAssumeUserSessionUsernameClaim, "db_role")
	assert.Equal(t, c.OIDCAssumeUserSessionAllowEscape, true)
	assert.Equal(t, c.ServerTLSEnabled, true)
	assert.Equal(t, c.ServerTLSCertificateFile, "../data/cert.pem")
	assert.Equal(t, c.ServerTLSCertificateKeyFile, "../data/key.pem")
	assert.Equal(t, c.ServerPort, 9876)
	assert.Equal(t, c.ApiPort, 8888)
	assert.Equal(t, c.APITLSEnabled, true)
	assert.Equal(t, c.ApiUsernameLifetime, 7200)
	assert.Equal(t, c.ApiGarbageCollectionPeriod, 6000)
}

func TestBadMapping(t *testing.T) {
	c, err := NewConfiguration([]string{
		"--destination-database-type", "postgres",
		"--destination-host", "localhost",
		"--destination-port", "5432",
		"--oidc-database-client-id", "postgres=pg-client-id=somethingelse=just-wrong",
		"--oidc-database-client-secret", "postgres=pg-client-id=somethingelse=just-wrong",
	})
	assert.DeepEqual(t, c.OIDCDatabaseClients, map[string]*OIDCDatabaseClientSpec{"postgres": {
		ClientID:     "pg-client-id=somethingelse=just-wrong",
		ClientSecret: "pg-client-id=somethingelse=just-wrong",
	}})
	assert.NilError(t, err)

	_, err = NewConfiguration([]string{
		"--destination-database-type", "postgres",
		"--destination-host", "localhost",
		"--destination-port", "5432",
		"--oidc-database-client-id", "postgres",
	})
	assert.Error(t, err, "invalid OIDC Database Client ID mapping: postgres")

	_, err = NewConfiguration([]string{
		"--destination-database-type", "postgres",
		"--destination-host", "localhost",
		"--destination-port", "5432",
		"--oidc-database-client-secret", "postgres",
	})
	assert.Error(t, err, "invalid OIDC Database Client Secret mapping: postgres")

	_, err = NewConfiguration([]string{
		"--destination-database-type", "postgres",
		"--destination-host", "localhost",
		"--destination-port", "5432",
		"--oidc-database-client-id", "postgres=pg-client-id,postgres=another-client-id",
	})
	assert.Error(t, err, "OIDC Database Client ID mapping has a duplicate database: postgres")

	_, err = NewConfiguration([]string{
		"--destination-database-type", "postgres",
		"--destination-host", "localhost",
		"--destination-port", "5432",
		"--oidc-database-client-secret", "postgres=pg-client-secret,postgres=another-client-secret",
	})
	assert.Error(t, err, "OIDC Database Client Secret mapping has a duplicate database: postgres")

	_, err = NewConfiguration([]string{
		"--destination-database-type", "postgres",
		"--destination-host", "localhost",
		"--destination-port", "5432",
		"--oidc-database-client-secret", "postgres=pg-client-secret",
	})
	assert.Error(t, err, "OIDC Database Client Secret mapping does not have a corresponding Client ID: postgres")
}

func TestMissingPostAuthTemplate(t *testing.T) {
	_, err := NewConfiguration([]string{
		"--destination-database-type", "postgres",
		"--destination-host", "localhost",
		"--destination-port", "5432",
		"--oidc-post-auth-sql-template", "missing-file.sql",
	})
	assert.Error(t, err, "OIDC Post Auth SQL template file does not exist: missing-file.sql")
}

func TestBadTLSConfiguration(t *testing.T) {
	_, err := NewConfiguration([]string{
		"--destination-database-type", "postgres",
		"--destination-host", "localhost",
		"--destination-port", "5432",
		"--server-tls-enabled",
	})
	assert.Error(t, err, "TLS certificate file is required")

	_, err = NewConfiguration([]string{
		"--destination-database-type", "postgres",
		"--destination-host", "localhost",
		"--destination-port", "5432",
		"--server-tls-enabled",
		"--server-tls-certificate-file", "missing-file.pem",
	})
	assert.Error(t, err, "TLS certificate file does not exist: missing-file.pem")

	_, err = NewConfiguration([]string{
		"--destination-database-type", "postgres",
		"--destination-host", "localhost",
		"--destination-port", "5432",
		"--server-tls-enabled",
		"--server-tls-certificate-file", "../data/cert.pem",
	})
	assert.Error(t, err, "TLS certificate key file is required")

	_, err = NewConfiguration([]string{
		"--destination-database-type", "postgres",
		"--destination-host", "localhost",
		"--destination-port", "5432",
		"--server-tls-enabled",
		"--server-tls-certificate-file", "../data/cert.pem",
		"--server-tls-certificate-key-file", "missing-key.pem",
	})
	assert.Error(t, err, "TLS certificate key file does not exist: missing-key.pem")
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
