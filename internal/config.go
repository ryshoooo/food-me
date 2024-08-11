package foodme

import (
	"fmt"
	"os"
	"strings"

	"github.com/jessevdk/go-flags"
	"github.com/sirupsen/logrus"
)

type OIDCDatabaseClientSpec struct {
	ClientID     string
	ClientSecret string
}

type Configuration struct {
	// Log configuration
	LogLevel  string `long:"log-level" env:"LOG_LEVEL" default:"warn" choice:"trace" choice:"debug" choice:"info" choice:"warn" choice:"error" choice:"fatal" choice:"panic" description:"Log level"`
	LogFormat string `long:"log-format" env:"LOG_FORMAT" default:"pretty" choice:"text" choice:"json" choice:"pretty" description:"Log format"`

	// Destination
	DestinationHost          string `long:"destination-host" env:"DESTINATION_HOST" required:"true" description:"Database host"`
	DestinationPort          int    `long:"destination-port" env:"DESTINATION_PORT" required:"true" description:"Database port"`
	DestinationDatabaseType  string `long:"destination-database-type" env:"DESTINATION_DATABASE_TYPE" choice:"postgres" required:"true" description:"Database type"`
	DestinationUsername      string `long:"destination-username" env:"DESTINATION_USERNAME" description:"Database root username"`
	DestinationPassword      string `long:"destination-password" env:"DESTINATION_PASSWORD" description:"Database root password"`
	DestinationLogUpstream   bool   `long:"destination-log-upstream" env:"DESTINATION_LOG_UPSTREAM" description:"Log packets from the destination database"`
	DestinationLogDownstream bool   `long:"destination-log-downstream" env:"DESTINATION_LOG_DOWNSTREAM" description:"Log packets from the source client"`

	// OIDC
	OIDCEnabled      bool   `long:"oidc-enabled" env:"OIDC_ENABLED" description:"Enable OIDC authentication"`
	OIDCClientID     string `long:"oidc-client-id" env:"OIDC_CLIENT_ID" description:"Global OIDC Client ID"`
	OIDCClientSecret string `long:"oidc-client-secret" env:"OIDC_CLIENT_SECRET" description:"Global OIDC Client Secret"`
	OIDCTokenURL     string `long:"oidc-token-url" env:"OIDC_TOKEN_URL" description:"OIDC Token URL"`
	OIDCUserInfoURL  string `long:"oidc-user-info-url" env:"OIDC_USER_INFO_URL" description:"OIDC User Info URL"`

	// OIDC-Database
	EDatabaseClientID                  string `long:"oidc-database-client-id" env:"OIDC_DATABASE_CLIENT_ID" description:"OIDC Database Client ID mapping"`
	EDatabaseClientSecret              string `long:"oidc-database-client-secret" env:"OIDC_DATABASE_CLIENT_SECRET" description:"OIDC Database Client Secret mapping"`
	OIDCDatabaseClients                map[string]*OIDCDatabaseClientSpec
	OIDCDatabaseFallBackToBaseClient   bool   `long:"oidc-database-fallback-to-base-client" env:"OIDC_DATABASE_FALLBACK_TO_BASE_CLIENT" description:"Fall back to the base client if the client ID is not found"`
	OIDCAssumeUserSession              bool   `long:"oidc-assume-user-session" env:"OIDC_ASSUME_USER_SESSION" description:"Assume the user role upon successful authentication"`
	OIDCAssumeUserSessionUsernameClaim string `long:"oidc-assume-user-session-username-claim" env:"OIDC_ASSUME_USER_SESSION_USERNAME_CLAIM" default:"preferred_username" description:"Username claim of the UserInfo response to use as the username for the connection session"`
	OIDCAssumeUserSessionAllowEscape   bool   `long:"oidc-assume-user-session-allow-escape" env:"OIDC_ASSUME_USER_SESSION_ALLOW_ESCAPE" description:"Allow the user to escape the assumed session"`
	OIDCPostAuthSQLTemplate            string `long:"oidc-post-auth-sql-template" env:"OIDC_POST_AUTH_SQL_TEMPLATE" description:"SQL template file to execute after a successful OIDC authentication"`

	// Permission Agents
	PermissionAgentEnabled bool   `long:"permission-agent-enabled" env:"PERMISSION_AGENT_ENABLED" description:"Enable permission agent for handling SQL queries"`
	PermissionAgentType    string `long:"permission-agent-type" env:"PERMISSION_AGENT_TYPE" choice:"opa" description:"Permission agent type"`

	// OPA Permission Agent Configuration
	PermissionAgentOPAURL                   string `long:"permission-agent-opa-url" env:"PERMISSION_AGENT_OPA_URL" description:"URL endpoint for OPA server"`
	PermissionAgentOPASelectQueryTemplate   string `long:"permission-agent-opa-select-query-template" env:"PERMISSION_AGENT_OPA_SELECT_QUERY_TEMPLATE" description:"Golang template for OPA SELECT query formulation" default:"data.{{ .TableName }}.allow == true"`
	PermissionAgentOPACreateQuery           string `long:"permission-agent-opa-create-query" env:"PERMISSION_AGENT_OPA_CREATE_QUERY" description:"OPA query for CREATE operations" default:"data.ddl_create.allow == true"`
	PermissionAgentOPAUpdateQuery           string `long:"permission-agent-opa-update-query" env:"PERMISSION_AGENT_OPA_UPDATE_QUERY" description:"OPA query for UPDATE operations" default:"data.ddl_update.allow == true"`
	PermissionAgentOPADeleteQuery           string `long:"permission-agent-opa-delete-query" env:"PERMISSION_AGENT_OPA_DELETE_QUERY" description:"OPA query for DELETE operations" default:"data.ddl_delete.allow == true"`
	PermissionAgentOPAStringEscapeCharacter string `long:"permission-agent-opa-string-escape-character" env:"PERMISSION_AGENT_OPA_STRING_ESCAPE_CHARACTER" description:"Wrap the resulting OPA string fields with this characters" default:"'"`

	// TLS
	ServerTLSEnabled            bool   `long:"server-tls-enabled" env:"SERVER_TLS_ENABLED" description:"Enable TLS for the server"`
	ServerTLSCertificateFile    string `long:"server-tls-certificate-file" env:"SERVER_TLS_CERTIFICATE_FILE" description:"TLS certificate file"`
	ServerTLSCertificateKeyFile string `long:"server-tls-certificate-key-file" env:"SERVER_TLS_CERTIFICATE_KEY_FILE" description:"TLS certificate key file"`

	// Server
	ServerPort int `long:"port" env:"PORT" default:"2099" description:"Server proxy port"`

	// API
	ApiPort                    int  `long:"api-port" env:"API_PORT" default:"10000" description:"API port"`
	APITLSEnabled              bool `long:"api-tls-enabled" env:"API_TLS_ENABLED" description:"Enable TLS for the API"`
	ApiUsernameLifetime        int  `long:"api-username-lifetime" env:"API_USERNAME_LIFETIME" default:"3600" description:"Username lifetime in seconds"`
	ApiGarbageCollectionPeriod int  `long:"api-garbage-collection-period" env:"API_GARBAGE_COLLECTION_PERIOD" default:"60" description:"Garbage collection period in seconds"`
}

func NewConfiguration(args []string) (*Configuration, error) {
	c := &Configuration{OIDCDatabaseClients: make(map[string]*OIDCDatabaseClientSpec)}

	// Parse the command line arguments
	p := flags.NewParser(c, flags.Default)
	_, err := p.ParseArgs(args)
	if err != nil {
		return nil, err
	}

	// parse database client id and secret
	var clientids = make(map[string]string)
	for _, key := range strings.Split(c.EDatabaseClientID, ",") {
		if key == "" {
			continue
		}
		kv := strings.Split(key, "=")
		if len(kv) < 2 {
			return nil, fmt.Errorf("invalid OIDC Database Client ID mapping: %s", key)
		}
		if _, ok := clientids[kv[0]]; ok {
			return nil, fmt.Errorf("OIDC Database Client ID mapping has a duplicate database: %s", kv[0])
		}
		clientids[kv[0]] = strings.Join(kv[1:], "=")
	}

	var clientsecrets = make(map[string]string)
	for _, key := range strings.Split(c.EDatabaseClientSecret, ",") {
		if key == "" {
			continue
		}
		kv := strings.Split(key, "=")
		if len(kv) < 2 {
			return nil, fmt.Errorf("invalid OIDC Database Client Secret mapping: %s", key)
		}
		if _, ok := clientsecrets[kv[0]]; ok {
			return nil, fmt.Errorf("OIDC Database Client Secret mapping has a duplicate database: %s", kv[0])
		}
		clientsecrets[kv[0]] = strings.Join(kv[1:], "=")
	}

	for db, cid := range clientids {
		c.OIDCDatabaseClients[db] = &OIDCDatabaseClientSpec{ClientID: cid}
	}
	for db, csec := range clientsecrets {
		if spec, ok := c.OIDCDatabaseClients[db]; ok {
			spec.ClientSecret = csec
		} else {
			return nil, fmt.Errorf("OIDC Database Client Secret mapping does not have a corresponding Client ID: %s", db)
		}
	}

	// Check whether the template file exists
	if c.OIDCPostAuthSQLTemplate != "" {
		if _, err := os.Stat(c.OIDCPostAuthSQLTemplate); os.IsNotExist(err) {
			return nil, fmt.Errorf("OIDC Post Auth SQL template file does not exist: %s", c.OIDCPostAuthSQLTemplate)
		}
	}

	// Check TLS files
	if c.ServerTLSEnabled || c.APITLSEnabled {
		if c.ServerTLSCertificateFile == "" {
			return nil, fmt.Errorf("TLS certificate file is required")
		}
		if _, err := os.Stat(c.ServerTLSCertificateFile); os.IsNotExist(err) {
			return nil, fmt.Errorf("TLS certificate file does not exist: %s", c.ServerTLSCertificateFile)
		}
		if c.ServerTLSCertificateKeyFile == "" {
			return nil, fmt.Errorf("TLS certificate key file is required")
		}
		if _, err := os.Stat(c.ServerTLSCertificateKeyFile); os.IsNotExist(err) {
			return nil, fmt.Errorf("TLS certificate key file does not exist: %s", c.ServerTLSCertificateKeyFile)
		}
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
