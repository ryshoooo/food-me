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
	ClientID     string
	ClientSecret string
	TokenURL     string
	UserInfoURL  string
	AccessToken  string
	RefreshToken string
}

func NewOIDCClient(clientId, clientSecret, tokenUrl, userInfoUrl, accessToken, refreshToken string) *OIDCClient {
	return &OIDCClient{
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

	token, _ := jwt.Parse(c.AccessToken, nil)
	if token == nil {
		return false
	}
	dt, err := token.Claims.GetExpirationTime()
	if err != nil {
		return false
	}

	return (dt.Time.Unix() - time.Now().Unix()) >= 0
}

func (c *OIDCClient) RefreshAccessToken() error {
	// Make the refresh token request
	httpClient := &http.Client{}

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

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code from refresh token: %d", resp.StatusCode)
	}

	// Parse the response
	var tokenResponse struct {
		AccessToken string `json:"access_token"`
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
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

	httpClient := &http.Client{}

	req, err := http.NewRequest(http.MethodGet, c.UserInfoURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.AccessToken))

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code from user info: %d", resp.StatusCode)
	}

	var userInfo map[string]interface{}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(b, &userInfo)
	if err != nil {
		return nil, err
	}

	return userInfo, nil
}
