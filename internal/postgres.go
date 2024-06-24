package foodme

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"text/template"

	"github.com/blastrain/vitess-sqlparser/sqlparser"
	"github.com/sirupsen/logrus"
	"github.com/xdg-go/scram"
)

type OIDCDatabaseClientSpec struct {
	ClientID     string
	ClientSecret string
}

type PostgresHandler struct {
	// Init
	Address                          string
	Username                         string
	Password                         string
	UpstreamHandler                  IUpstreamHandler
	Logger                           *logrus.Logger
	LogUpstream                      bool
	LogDownstream                    bool
	HTTPClient                       IHttpClient
	OIDCEnabled                      bool
	OIDCClientID                     string
	OIDCClientSecret                 string
	OIDCTokenURL                     string
	OIDCUserInfoURL                  string
	OIDCDatabaseFallBackToBaseClient bool
	OIDCDatabaseClients              map[string]*OIDCDatabaseClientSpec
	OIDCPostAuthSQLTemplate          string
	AssumeUserSession                bool
	UsernameClaim                    string
	AllowSessionEscape               bool

	// Runtime
	client     net.Conn
	upstream   net.Conn
	database   string
	oidcClient *OIDCClient
	userinfo   map[string]interface{}
}

func NewPostgresHandler(
	address, username, password string,
	upstreamHandler IUpstreamHandler,
	logger *logrus.Logger,
	logUpstream, logDownstream, oidcEnabled bool,
	httpClient IHttpClient,
	oidcClientId, oidcClientSecret, oidcTokenUrl, oidcUserInfoUrl string,
	oidcBaseClientFallback bool,
	oidcDatabaseClients map[string]*OIDCDatabaseClientSpec,
	oidcPostAuthTemplate string,
	assumeUserSession bool, usernameClaim string, allowSessionEscape bool,
) *PostgresHandler {
	return &PostgresHandler{
		Address:                          address,
		Username:                         username,
		Password:                         password,
		UpstreamHandler:                  upstreamHandler,
		Logger:                           logger,
		LogUpstream:                      logUpstream,
		LogDownstream:                    logDownstream,
		HTTPClient:                       httpClient,
		OIDCEnabled:                      oidcEnabled,
		OIDCClientID:                     oidcClientId,
		OIDCClientSecret:                 oidcClientSecret,
		OIDCTokenURL:                     oidcTokenUrl,
		OIDCUserInfoURL:                  oidcUserInfoUrl,
		OIDCDatabaseFallBackToBaseClient: oidcBaseClientFallback,
		OIDCDatabaseClients:              oidcDatabaseClients,
		OIDCPostAuthSQLTemplate:          oidcPostAuthTemplate,
		AssumeUserSession:                assumeUserSession,
		UsernameClaim:                    usernameClaim,
		AllowSessionEscape:               allowSessionEscape,
	}
}

func (h *PostgresHandler) Handle(conn net.Conn) error {
	h.client = conn
	defer h.client.Close()

	destination, err := h.UpstreamHandler.Connect()
	if err != nil {
		h.Logger.Errorf("Unable to connect to destination: %v", err)
		return err
	}
	h.upstream = destination
	defer h.upstream.Close()

	// Startup
	size, err := h.startup()
	if err != nil {
		h.Logger.Errorf("Error on startup: %v", err)
		return h.sendErrorMessage("08000", err)
	}

	// Authenticate
	err = h.authenticate(size)
	if err != nil {
		h.Logger.Errorf("Error on authentication: %v", err)
		return h.sendErrorMessage("28000", err)
	}

	// Continue as proxy
	go h.proxyDownstream()
	h.proxyUpstream()

	return nil
}

func (h *PostgresHandler) startup() ([]byte, error) {
	size, err := h.read(4, "client")
	if err != nil {
		return []byte{}, err
	}

	sizeInt := calculatePacketSize(size)

	// If the size is 8, it is a startup message
	if sizeInt != 8 {
		h.Logger.Info("Startup message not found, continue without startup exchange")
		return size, nil
	}

	h.Logger.Info("Commencing startup")
	startup, err := h.read(4, "client")
	if err != nil {
		return []byte{}, err
	}
	h.Logger.Debugf("Read startup packet from client: %v", startup)
	err = h.write(append([]byte{0, 0, 0, 8}, startup...), "upstream")
	if err != nil {
		return []byte{}, err
	}
	resp, err := h.read(1, "upstream")
	if err != nil {
		return []byte{}, err
	}
	h.Logger.Debugf("Read startup response from upstream: %v", resp)
	if resp[0] != 'N' {
		return []byte{}, fmt.Errorf("unexpected response from upstream: %v", resp)
	}
	err = h.write(resp, "client")
	if err != nil {
		return []byte{}, err
	}
	h.Logger.Info("Startup successful")

	// Read the next size from the client
	size, err = h.read(4, "client")
	if err != nil {
		h.Logger.Errorf("Error reading from client: %v", err)
		return []byte{}, err
	}
	return size, nil
}

func (h *PostgresHandler) write(data []byte, name string) error {
	h.Logger.Debugf("Writing data to %s: %v", name, data)

	var n int
	var err error
	if name == "client" {
		n, err = h.client.Write(data)
	} else {
		n, err = h.upstream.Write(data)
	}

	if err != nil || n != len(data) {
		h.Logger.Errorf("Error writing to %s: %v", name, err)
		return err
	}
	return nil
}

func (h *PostgresHandler) read(size int, name string) ([]byte, error) {
	h.Logger.Debugf("Reading %v bytes from %s", size, name)
	buff := make([]byte, size)

	var n int
	var err error
	if name == "client" {
		n, err = h.client.Read(buff)
	} else {
		n, err = h.upstream.Read(buff)
	}

	if err != nil || n != size {
		if err == io.EOF {
			return nil, err
		}
		h.Logger.Errorf("Error reading from %s: %v", name, err)
		return nil, err
	}
	return buff, nil
}

func (h *PostgresHandler) sendErrorMessage(code string, err error) error {
	resp := []byte("E")
	msg := []byte("SERROR")
	msg = append(msg, 0)
	msg = append(msg, []byte("VERROR")...)
	msg = append(msg, 0)
	msg = append(msg, append([]byte("C"), []byte(code)...)...)
	msg = append(msg, 0)
	msg = append(msg, append([]byte("M"), []byte(err.Error())...)...)
	msg = append(msg, 0)
	msg = append(msg, 0)

	s := createPacketSize(len(msg) + 4)
	resp = append(resp, s...)
	resp = append(resp, msg...)
	return h.write(resp, "client")
}

func (h *PostgresHandler) authenticate(sizebuff []byte) error {
	h.Logger.Info("Commencing authentication")

	size := calculatePacketSize(sizebuff)
	auth, err := h.read(size-4, "client")
	if err != nil {
		return err
	}
	h.Logger.Debugf("Read authentication packet from client: %v (%s)", auth, auth)

	parts := bytes.Split(auth, []byte{0})
	if len(parts) < 7 {
		return fmt.Errorf("invalid authentication packet: %v", parts)
	}
	u := string(parts[3])
	uv := string(parts[4])
	d := string(parts[5])
	dv := string(parts[6])
	h.Logger.Debugf("Authentication: %v=%v %v=%v", u, uv, d, dv)

	h.database = dv
	accessToken, refreshToken := GlobalState.GetTokens(uv)
	if accessToken == "" || refreshToken == "" {
		uvs := strings.Split(uv, ";")
		if len(uvs) < 2 {
			h.Logger.Info("Username does not contain OIDC data, proxy all the requests going forward")
			return h.write(append(sizebuff, auth...), "upstream")
		}
		h.Logger.Debugf("OIDC data: %v", uvs)
		for _, ov := range uvs {
			if strings.HasPrefix(ov, "access_token=") {
				accessToken = strings.Split(ov, "=")[1]
			}
			if strings.HasPrefix(ov, "refresh_token=") {
				refreshToken = strings.Split(ov, "=")[1]
			}
		}
	}

	h.Logger.Debugf("Access token: %v", accessToken)
	h.Logger.Debugf("Refresh token: %v", refreshToken)
	if accessToken == "" || refreshToken == "" {
		h.Logger.Info("Access token or refresh token is missing, proxy all the requests going forward")
		return h.write(append(sizebuff, auth...), "upstream")
	}

	// Strange situation here, we have access and refresh tokens, but OIDC is disabled
	// Send an informative error message to the client
	if !h.OIDCEnabled {
		h.Logger.Error("OIDC is disabled, but access and refresh tokens are present")
		return fmt.Errorf("oidc as auth method is disabled, use username/password")
	}

	var clientId string
	var clientSecret string
	if cv, ok := h.OIDCDatabaseClients[h.database]; !ok {
		if h.OIDCDatabaseFallBackToBaseClient {
			clientId = h.OIDCClientID
			clientSecret = h.OIDCClientSecret
		} else {
			h.Logger.Errorf("Client ID not found for database: %v", h.database)
			return fmt.Errorf("client ID not found for database: %v", h.database)
		}
	} else {
		clientId = cv.ClientID
		clientSecret = cv.ClientSecret
	}

	h.oidcClient = NewOIDCClient(h.HTTPClient, clientId, clientSecret, h.OIDCTokenURL, h.OIDCUserInfoURL, accessToken, refreshToken)
	if !h.oidcClient.IsAccessTokenValid() {
		h.Logger.Info("Access token is invalid, refreshing the token")
		err = h.oidcClient.RefreshAccessToken()
		if err != nil {
			return err
		}
	}

	userinfo, err := h.oidcClient.GetUserInfo()
	if err != nil {
		return err
	}
	h.userinfo = userinfo
	h.Logger.Infof("User info: %v", userinfo)

	// Authenticate as the configured user
	err = h.auth()
	if err != nil {
		return err
	}

	// Auth successful, send the auth OK to the client
	err = h.write([]byte{82, 0, 0, 0, 8, 0, 0, 0, 0}, "client")
	if err != nil {
		return err
	}

	// Pipe the rest of the metadata until ready for query
	err = h.readUntilReadyForQuery("authentication", true)
	if err != nil {
		return err
	}

	// Post-authentication script
	if h.OIDCPostAuthSQLTemplate != "" {
		h.Logger.Info("Executing post-authentication script")
		err = h.executePostAuthStatement()
		if err != nil {
			return err
		}
	}

	// Assume user session
	if h.AssumeUserSession {
		h.Logger.Info("Assuming user session")
		err = h.assumeUserSession()
		if err != nil {
			return err
		}
	}

	// Send OK to client
	err = h.write([]byte{90, 0, 0, 0, 5, 73}, "client")
	if err != nil {
		return err
	}

	return nil
}

func (h *PostgresHandler) auth() error {
	h.Logger.Info("Authenticating as configured user")

	// Send initial auth request
	msg := []byte{0, 3, 0, 0}
	msg = append(msg, []byte("user")...)
	msg = append(msg, 0)
	msg = append(msg, []byte(h.Username)...)
	msg = append(msg, 0)
	msg = append(msg, []byte("database")...)
	msg = append(msg, 0)
	msg = append(msg, []byte(h.database)...)
	msg = append(msg, []byte{0, 0}...)
	size := createPacketSize(len(msg) + 4)
	msg = append(size, msg...)
	err := h.write(msg, "upstream")
	if err != nil {
		return err
	}

	// Read auth response challenge
	r, err := h.readMessage('R')
	if err != nil {
		return err
	}

	// Check trust auth method
	if checkAuthenticationSuccess(r) {
		h.Logger.Info("Trust auth method reply. Authentication successful")
		return nil
	}

	// Determine the method
	rs := bytes.Split(r, []byte{0})
	if len(rs) < 3 {
		return fmt.Errorf("unexpected response from db: %v", rs)
	}
	h.Logger.Debugf("Auth method: %v", rs)
	method := rs[3][0]
	switch int(method) {
	case 3:
		h.Logger.Info("Clear password auth method")
		return h.handleClearPasswordAuth()
	case 5:
		h.Logger.Info("MD5 password auth method")
		return h.handleMD5PasswordAuth(string(rs[3][1:]))
	case 7:
	case 8:
		h.Logger.Info("GSSAPI auth method")
		return fmt.Errorf("GSSAPI auth method not supported")
	case 10:
		h.Logger.Info("SCRAM-SHA-256 auth method")
		return h.handleSCRAMSHA256Auth()
	default:
		return fmt.Errorf("unknown auth method: %v", method)
	}

	return nil
}

func (h *PostgresHandler) readMessage(expR byte) ([]byte, error) {
	r, err := h.read(1, "upstream")
	if err != nil {
		return nil, err
	}
	if r[0] != expR {
		return nil, fmt.Errorf("unexpected response from db: %v %s", r, r)
	}
	r, err = h.read(4, "upstream")
	if err != nil {
		return nil, err
	}
	rsize := calculatePacketSize(r)
	r, err = h.read(rsize-4, "upstream")
	if err != nil {
		return nil, err
	}
	return r, nil
}

func (h *PostgresHandler) handleClearPasswordAuth() error {
	msg := []byte{'p'}
	pwd := []byte(h.Password)
	s := createPacketSize(len(pwd) + 5)
	msg = append(msg, s...)
	msg = append(msg, pwd...)
	msg = append(msg, 0)
	err := h.write(msg, "upstream")
	if err != nil {
		return err
	}

	// Read auth response
	r, err := h.readMessage('R')
	if err != nil {
		return err
	}

	if !checkAuthenticationSuccess(r) {
		return fmt.Errorf("authentication failed, response from db: %v", r)
	}

	h.Logger.Info("Clear password auth successful")
	return nil
}

func (h *PostgresHandler) handleMD5PasswordAuth(key string) error {
	msg := []byte{'p'}

	// Calculate the MD5 hash
	md5H := md5.New()
	_, err := md5H.Write([]byte(h.Password + h.Username))
	if err != nil {
		return err
	}
	pwd1 := fmt.Sprintf("%x", md5H.Sum(nil))
	md5H.Reset()
	_, err = md5H.Write([]byte(pwd1 + key))
	if err != nil {
		return err
	}
	pwd := fmt.Sprintf("md5%x", md5H.Sum(nil))

	// Send the password
	s := createPacketSize(len(pwd) + 5)
	msg = append(msg, s...)
	msg = append(msg, []byte(pwd)...)
	msg = append(msg, 0)
	err = h.write(msg, "upstream")
	if err != nil {
		return err
	}

	// Handle response
	r, err := h.readMessage('R')
	if err != nil {
		return err
	}

	if !checkAuthenticationSuccess(r) {
		return fmt.Errorf("authentication failed, response from db: %v", r)
	}
	h.Logger.Info("MD5 password auth successful")
	return nil
}

func (h *PostgresHandler) handleSCRAMSHA256Auth() error {
	msg := []byte{'p'}

	// First step
	client, err := scram.SHA256.NewClient(h.Username, h.Password, "")
	if err != nil {
		return err
	}
	conv := client.NewConversation()
	var resp string
	firstMsg, err := conv.Step(resp)
	if err != nil {
		return err
	}
	h.Logger.Debugf("First step: %v", firstMsg)

	// Send the first step to the db
	pwd := []byte("SCRAM-SHA-256")
	pwd = append(pwd, 0)
	firstStepSize := createPacketSize(len(firstMsg))
	pwd = append(pwd, firstStepSize...)
	pwd = append(pwd, []byte(firstMsg)...)
	size := createPacketSize(len(pwd) + 4)
	msg = append(msg, size...)
	msg = append(msg, pwd...)
	err = h.write(msg, "upstream")
	if err != nil {
		return err
	}

	// Get the data for second step
	r, err := h.readMessage('R')
	if err != nil {
		return err
	}
	rs := bytes.Split(r, []byte{0})
	if len(rs) < 4 {
		return fmt.Errorf("unexpected response from db: %v", rs)
	}
	if rs[3][0] != 11 {
		return fmt.Errorf("unexpected response from db: %v", rs)
	}
	resp = string(rs[3][1:])
	h.Logger.Debugf("First step response: %v", resp)

	// Second step
	secondMsg, err := conv.Step(resp)
	if err != nil {
		return err
	}
	h.Logger.Debugf("Second step: %v", secondMsg)

	// Send the second step to the db
	msg = []byte{'p'}
	size = createPacketSize(len(secondMsg) + 4)
	msg = append(msg, size...)
	msg = append(msg, []byte(secondMsg)...)
	err = h.write(msg, "upstream")
	if err != nil {
		return err
	}

	// Get the data for the third step
	r, err = h.readMessage('R')
	if err != nil {
		return err
	}
	rs = bytes.Split(r, []byte{0})
	if len(rs) < 4 {
		return fmt.Errorf("unexpected response from db: %v", rs)
	}
	if rs[3][0] != 12 {
		return fmt.Errorf("unexpected response from db: %v", rs)
	}
	resp = string(rs[3][1:])
	h.Logger.Debugf("Second step response: %v", resp)

	// Third step (validation)
	_, err = conv.Step(resp)
	if err != nil {
		return err
	}

	// Expecting success from the db
	r, err = h.readMessage('R')
	if err != nil {
		return err
	}
	if !checkAuthenticationSuccess(r) {
		return fmt.Errorf("authentication failed, response from db: %v", r)
	}

	h.Logger.Info("SCRAM-SHA-256 auth successful")
	return nil
}

func (h *PostgresHandler) readUntilReadyForQuery(process string, sendDownstream bool) error {
	for {
		op, size, data, err := h.readFullMessage("upstream")
		if err != nil {
			return err
		}
		if op[0] == 'E' {
			return fmt.Errorf("error %s: %v", process, getErrorMessage(data))
		}
		if op[0] == 'Z' {
			break
		}
		if sendDownstream {
			err = h.write(append(op, append(size, data...)...), "client")
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (h *PostgresHandler) readFullMessage(name string) ([]byte, []byte, []byte, error) {
	var doLog bool
	if name == "client" {
		doLog = h.LogDownstream
	} else {
		doLog = h.LogUpstream
	}

	operation, err := h.read(1, name)
	if err != nil {
		return nil, nil, nil, err
	}
	size, err := h.read(4, name)
	if err != nil {
		return nil, nil, nil, err
	}
	sizeInt := calculatePacketSize(size)
	data, err := h.read(sizeInt-4, name)
	if err != nil {
		return nil, nil, nil, err
	}
	if doLog {
		h.Logger.Debugf("Operation: %v (%s); Read %v bytes from %s: %s", operation, operation, sizeInt, name, data)
	}
	return operation, size, data, nil
}

func (h *PostgresHandler) executePostAuthStatement() error {
	tr, err := os.ReadFile(h.OIDCPostAuthSQLTemplate)
	if err != nil {
		return err
	}
	tmpl, err := template.New("post-auth").Parse(string(tr))
	if err != nil {
		return err
	}
	var ps bytes.Buffer
	err = tmpl.Execute(&ps, h.userinfo)
	if err != nil {
		return err
	}

	stmt := ps.String()
	h.Logger.Debugf("Post-auth statement: %s", stmt)

	err = h.executeQuery(stmt, "post-auth")
	if err != nil {
		return err
	}

	h.Logger.Info("Assumed user session")
	return nil
}

func (h *PostgresHandler) executeQuery(query, process string) error {
	// Start
	msg := []byte{'Q'}
	q := []byte("BEGIN")
	q = append(q, 0)
	size := createPacketSize(len(q) + 4)
	msg = append(msg, size...)
	msg = append(msg, q...)
	err := h.write(msg, "upstream")
	if err != nil {
		return err
	}

	err = h.readUntilReadyForQuery(process, false)
	if err != nil {
		return err
	}

	// Execute
	msg = []byte{'Q'}
	q = []byte(query)
	q = append(q, 0)
	size = createPacketSize(len(q) + 4)
	msg = append(msg, size...)
	msg = append(msg, q...)
	err = h.write(msg, "upstream")
	if err != nil {
		return err
	}
	err = h.readUntilReadyForQuery(process, false)
	if err != nil {
		return err
	}

	// Commit
	msg = []byte{'Q'}
	q = []byte("END")
	q = append(q, 0)
	size = createPacketSize(len(q) + 4)
	msg = append(msg, size...)
	msg = append(msg, q...)
	err = h.write(msg, "upstream")
	if err != nil {
		return err
	}
	err = h.readUntilReadyForQuery(process, false)
	if err != nil {
		return err
	}

	return nil
}

func (h *PostgresHandler) assumeUserSession() error {
	// Get the username
	var username string

	u, ok := h.userinfo[h.UsernameClaim]
	if !ok {
		return fmt.Errorf("username claim not found in userinfo: %v", h.UsernameClaim)
	}

	switch v := u.(type) {
	case string:
		username = u.(string)
	default:
		return fmt.Errorf("unexpected username claim type: %v with value %v", v, u)
	}

	err := h.executeQuery(fmt.Sprintf("SET SESSION AUTHORIZATION %s", username), "setting session authorization")
	if err != nil {
		return err
	}

	h.Logger.Info("Assumed user session")
	return nil
}

func (h *PostgresHandler) proxyDownstream() {
	if h.LogUpstream {
		buffer := make([]byte, 1024)
		for {
			n, err := h.upstream.Read(buffer)
			if err != nil {
				if err != io.EOF {
					h.Logger.Errorf("Error reading from upstream: %v", err)
				}
				break
			}
			h.Logger.Debugf("Read %v bytes from upstream: %v; %s", n, buffer[:n], buffer[:n])
			_, err = h.client.Write(buffer[:n])
			if err != nil {
				h.Logger.Errorf("Error writing to client: %v", err)
				break
			}
		}
	} else {
		_, err := io.Copy(h.client, h.upstream)
		if err != nil {
			if err != io.EOF {
				h.Logger.Errorf("Error copying from upstream to client: %v", err)
			}
		}
	}
}

func (h *PostgresHandler) proxyUpstream() {
	readyMessage := []byte{90, 0, 0, 0, 5, 69}

	for {
		op, size, data, err := h.readFullMessage("client")

		// Handle errors first
		if err == io.EOF {
			h.Logger.Info("Client closed connection")
			break
		}
		if err != nil {
			h.Logger.Errorf("Error reading from client: %v", err)
			break
		}
		if len(data) == 0 {
			h.Logger.Info("Client closed connection")
			break
		}

		// Check token validity
		if h.oidcClient != nil && !h.oidcClient.IsAccessTokenValid() {
			h.Logger.Debug("Access token is invalid, refreshing the token")
			err = h.oidcClient.RefreshAccessToken()
			if err != nil {
				h.Logger.Errorf("Error refreshing access token: %v", err)
				err = h.sendErrorMessage("28000", fmt.Errorf("error refreshing access token: %v", err))
				if err != nil {
					h.Logger.Errorf("Error sending error message to client: %v", err)
					break
				}
				err = h.write(readyMessage, "client")
				if err != nil {
					h.Logger.Errorf("Error writing to client: %v", err)
					break
				}
				continue
			}
		}

		if isEscapeSession(string(data[:len(data)-1])) && !h.AllowSessionEscape {
			h.Logger.Info("Session escape detected, ignoring the request")
			err = h.sendErrorMessage("28000", fmt.Errorf("session escape detected"))
			if err != nil {
				h.Logger.Errorf("Error sending error message to client: %v", err)
				break
			}
			err = h.write(readyMessage, "client")
			if err != nil {
				h.Logger.Errorf("Error writing to client: %v", err)
				break
			}
			continue
		}

		// Parse the SQL statement
		stmt, err := sqlparser.Parse(string(data[:len(data)-1]))
		if err != nil || stmt == nil {
			h.Logger.Errorf("Error parsing SQL: %v", err)
			err = h.write(append(op, append(size, data...)...), "upstream")
			if err != nil {
				h.Logger.Errorf("Error writing to upstream: %v", err)
				break
			}
			continue
		}

		h.Logger.Debugf("Parsed SQL statement: %s", sqlparser.String(stmt))
		err = h.write(append(op, append(size, data...)...), "upstream")
		if err != nil {
			h.Logger.Errorf("Error writing to upstream: %v", err)
			break
		}
	}
}
