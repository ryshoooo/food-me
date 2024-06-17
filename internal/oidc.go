package foodme

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type OIDCClient struct {
	HTTPClient   IHttpClient
	ClientID     string
	ClientSecret string
	TokenURL     string
	UserInfoURL  string
	AccessToken  string
	RefreshToken string
}

func NewOIDCClient(httpClient IHttpClient, clientId, clientSecret, tokenUrl, userInfoUrl, accessToken, refreshToken string) *OIDCClient {
	return &OIDCClient{
		HTTPClient:   httpClient,
		ClientID:     clientId,
		ClientSecret: clientSecret,
		TokenURL:     tokenUrl,
		UserInfoURL:  userInfoUrl,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}
}

func (c *OIDCClient) IsAccessTokenValid() bool {
	if c.AccessToken == "" {
		return false
	}

	// Parse the token
	claims := jwt.MapClaims{}
	token, _ := jwt.ParseWithClaims(c.AccessToken, claims, nil)
	if token == nil {
		return false
	}

	// Verify azp claim
	azp, ok := claims["azp"]
	if !ok {
		return false
	}
	switch azpType := azp.(type) {
	case string:
		if azpType != c.ClientID {
			return false
		}
	default:
		return false
	}

	// Verify exp claim
	dt, err := token.Claims.GetExpirationTime()
	if err != nil || dt == nil {
		return false
	}

	return (dt.Time.Unix() - time.Now().Unix()) >= 0
}

func (c *OIDCClient) RefreshAccessToken() error {
	data := url.Values{}
	data.Set("client_id", c.ClientID)
	if c.ClientSecret != "" {
		data.Set("client_secret", c.ClientSecret)
	}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", c.RefreshToken)
	req, err := http.NewRequest(http.MethodPost, c.TokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code from refresh token: %d. Body: %s", resp.StatusCode, string(b))
	}

	// Parse the response
	var tokenResponse struct {
		AccessToken string `json:"access_token"`
	}

	err = json.Unmarshal(b, &tokenResponse)
	if err != nil {
		return err
	}

	c.AccessToken = tokenResponse.AccessToken
	return nil
}

func (c *OIDCClient) GetUserInfo() (map[string]interface{}, error) {
	if c.AccessToken == "" {
		return nil, fmt.Errorf("access token is required to get user info")
	}

	req, err := http.NewRequest(http.MethodGet, c.UserInfoURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.AccessToken))

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code from user info: %d. Body: %s", resp.StatusCode, string(b))
	}

	var userInfo map[string]interface{}
	err = json.Unmarshal(b, &userInfo)
	if err != nil {
		return nil, err
	}
	return userInfo, nil
}
