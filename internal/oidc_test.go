package foodme

import (
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"gotest.tools/v3/assert"
)

type MockHttpClient struct {
	DoSucceed     bool
	Response      string
	FailBodyRead  bool
	StatusCode    int
	RequestBody   string
	RequestHeader http.Header
}

type MockBody struct {
	Body     string
	FailRead bool
}

func (m *MockBody) Read(p []byte) (n int, err error) {
	b := []byte(m.Body)
	copy(p, b)
	if m.FailRead {
		return len(b), fmt.Errorf("body read failure")
	} else {
		return len(b), io.EOF
	}
}

func (m *MockBody) Close() error {
	return nil
}

func (m *MockHttpClient) Do(req *http.Request) (*http.Response, error) {
	if !m.DoSucceed {
		return nil, fmt.Errorf("failed to do request")
	}

	if req.Body == nil {
		m.RequestBody = ""
	} else {
		b, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		m.RequestBody = string(b)
	}

	m.RequestHeader = req.Header
	resp := &http.Response{Body: &MockBody{Body: m.Response, FailRead: m.FailBodyRead}, StatusCode: m.StatusCode}
	return resp, nil
}

func createToken(t *testing.T, data map[string]interface{}) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims(data))
	tokenString, err := token.SignedString([]byte("secret"))
	assert.NilError(t, err)
	return tokenString
}

func TestNewOIDCClient(t *testing.T) {
	httpClient := &MockHttpClient{}
	clientId := "client-id"
	clientSecret := "client-secret"
	tokenUrl := "http://token-url"
	userInfoUrl := "http://user-info-url"
	accessToken := "access"
	refreshToken := "refresh"

	client := NewOIDCClient(httpClient, clientId, clientSecret, tokenUrl, userInfoUrl, accessToken, refreshToken)
	assert.Equal(t, client.HTTPClient, httpClient)
	assert.Equal(t, client.ClientID, clientId)
	assert.Equal(t, client.ClientSecret, clientSecret)
	assert.Equal(t, client.TokenURL, tokenUrl)
	assert.Equal(t, client.UserInfoURL, userInfoUrl)
	assert.Equal(t, client.AccessToken, accessToken)
	assert.Equal(t, client.RefreshToken, refreshToken)
}

func TestIsAccessTokenValid(t *testing.T) {
	// Test empty access token
	client := NewOIDCClient(&MockHttpClient{}, "client-id", "client-secret", "http://token-url", "http://user-info-url", "", "refresh")
	assert.Assert(t, !client.IsAccessTokenValid())

	// Fail parsing token
	client.AccessToken = "access"
	assert.Assert(t, !client.IsAccessTokenValid())

	// Parse empty token
	token := createToken(t, map[string]interface{}{})
	client.AccessToken = token
	assert.Assert(t, !client.IsAccessTokenValid())

	// Wrong AZP type
	token = createToken(t, map[string]interface{}{"azp": 2.3})
	client.AccessToken = token
	assert.Assert(t, !client.IsAccessTokenValid())

	// Wrong AZP value
	token = createToken(t, map[string]interface{}{"azp": "wrong"})
	client.AccessToken = token
	assert.Assert(t, !client.IsAccessTokenValid())

	// Missing exp claim
	token = createToken(t, map[string]interface{}{"azp": "client-id"})
	client.AccessToken = token
	assert.Assert(t, !client.IsAccessTokenValid())

	// Bad exp claim
	token = createToken(t, map[string]interface{}{"azp": "client-id", "exp": "bad"})
	client.AccessToken = token
	assert.Assert(t, !client.IsAccessTokenValid())

	// Expired token
	token = createToken(t, map[string]interface{}{"azp": "client-id", "exp": 1})
	client.AccessToken = token
	assert.Assert(t, !client.IsAccessTokenValid())

	// Valid token
	token = createToken(t, map[string]interface{}{"azp": "client-id", "exp": time.Now().Add(time.Hour).Unix()})
	client.AccessToken = token
	assert.Assert(t, client.IsAccessTokenValid())
}

func TestRefreshAccessToken(t *testing.T) {
	// Bad url
	client := NewOIDCClient(&MockHttpClient{}, "client-id", "", "blah://bad url", "http://user-info-url", "access", "refresh")
	err := client.RefreshAccessToken()
	assert.Error(t, err, "parse \"blah://bad url\": invalid character \" \" in host name")

	// Fail on do request
	client.TokenURL = "http://token-url"
	client.HTTPClient = &MockHttpClient{DoSucceed: false}
	err = client.RefreshAccessToken()
	assert.Error(t, err, "failed to do request")

	// Fail on body reading
	client.HTTPClient = &MockHttpClient{DoSucceed: true, Response: "bad response", FailBodyRead: true}
	err = client.RefreshAccessToken()
	assert.Error(t, err, "body read failure")

	// Fail on status code
	client.HTTPClient = &MockHttpClient{DoSucceed: true, Response: "bad response", StatusCode: 500}
	err = client.RefreshAccessToken()
	assert.Error(t, err, "unexpected status code from refresh token: 500. Body: bad response")

	// Fail on unmarshal
	client.HTTPClient = &MockHttpClient{DoSucceed: true, Response: "bad response", StatusCode: 200}
	err = client.RefreshAccessToken()
	assert.Error(t, err, "invalid character 'b' looking for beginning of value")

	// Test OK
	httpClient := &MockHttpClient{DoSucceed: true, Response: "{\"access_token\":\"new-access\"}", StatusCode: 200}
	client.HTTPClient = httpClient
	err = client.RefreshAccessToken()
	assert.NilError(t, err)
	assert.Equal(t, client.AccessToken, "new-access")
	assert.Equal(t, client.RefreshToken, "refresh")
	assert.Equal(t, httpClient.RequestBody, "client_id=client-id&grant_type=refresh_token&refresh_token=refresh")
	assert.DeepEqual(t, httpClient.RequestHeader, http.Header{"Content-Type": {"application/x-www-form-urlencoded"}})

	// Test OK with client secret
	client.ClientSecret = "secret"
	err = client.RefreshAccessToken()
	assert.NilError(t, err)
	assert.Equal(t, client.AccessToken, "new-access")
	assert.Equal(t, client.RefreshToken, "refresh")
	assert.Equal(t, httpClient.RequestBody, "client_id=client-id&client_secret=secret&grant_type=refresh_token&refresh_token=refresh")
	assert.DeepEqual(t, httpClient.RequestHeader, http.Header{"Content-Type": {"application/x-www-form-urlencoded"}})
}

func TestGetUserInfo(t *testing.T) {
	// Test empty access token
	client := NewOIDCClient(&MockHttpClient{}, "client-id", "client-secret", "http://token-url", "http://user-info-url", "", "refresh")
	_, err := client.GetUserInfo()
	assert.Error(t, err, "access token is required to get user info")

	// Test bad url
	client.AccessToken = "access"
	client.UserInfoURL = "blah://bad url"
	_, err = client.GetUserInfo()
	assert.Error(t, err, "parse \"blah://bad url\": invalid character \" \" in host name")

	// Test fail to do request
	client.UserInfoURL = "http://user-info-url"
	client.HTTPClient = &MockHttpClient{DoSucceed: false}
	_, err = client.GetUserInfo()
	assert.Error(t, err, "failed to do request")

	// Fail body read
	client.HTTPClient = &MockHttpClient{DoSucceed: true, Response: "bad response", FailBodyRead: true}
	_, err = client.GetUserInfo()
	assert.Error(t, err, "body read failure")

	// Test bad status code
	client.HTTPClient = &MockHttpClient{DoSucceed: true, Response: "bad response", StatusCode: 500}
	_, err = client.GetUserInfo()
	assert.Error(t, err, "unexpected status code from user info: 500. Body: bad response")

	// Fail unmarshal
	client.HTTPClient = &MockHttpClient{DoSucceed: true, Response: "bad response", StatusCode: 200}
	_, err = client.GetUserInfo()
	assert.Error(t, err, "invalid character 'b' looking for beginning of value")

	// Test OK
	httpClient := &MockHttpClient{DoSucceed: true, Response: "{\"name\":\"John\"}", StatusCode: 200}
	client.HTTPClient = httpClient
	data, err := client.GetUserInfo()
	assert.NilError(t, err)
	assert.DeepEqual(t, data, map[string]interface{}{"name": "John"})
	assert.DeepEqual(t, httpClient.RequestHeader, http.Header{"Authorization": {"Bearer access"}})
	assert.Equal(t, httpClient.RequestBody, "")
}
