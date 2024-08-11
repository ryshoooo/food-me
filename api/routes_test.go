package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"

	foodme "github.com/ryshoooo/food-me/internal"
	"github.com/sirupsen/logrus"
	"gotest.tools/v3/assert"
)

type MockBuffer struct {
	buffer []byte
}
type MockHeaders struct {
	headers []int
}

type MockResponseWriter struct {
	buffer  *MockBuffer
	headers *MockHeaders
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

func (m MockResponseWriter) Header() http.Header {
	return http.Header{}
}

func (m MockResponseWriter) Write(data []byte) (int, error) {
	m.buffer.buffer = append(m.buffer.buffer, data...)
	return len(data), nil
}

func (m MockResponseWriter) WriteHeader(statusCode int) {
	m.headers.headers = append(m.headers.headers, statusCode)
}

type MockHttpClient struct {
	DoSucceed     bool
	Response      []string
	FailBodyRead  bool
	StatusCode    int
	RequestBody   string
	RequestHeader http.Header

	responseIdx int
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
	resp := &http.Response{Body: &MockBody{Body: m.Response[m.responseIdx], FailRead: m.FailBodyRead}, StatusCode: m.StatusCode}
	m.responseIdx++
	return resp, nil
}

func TestHandleErrorResponse(t *testing.T) {
	w := MockResponseWriter{buffer: &MockBuffer{buffer: []byte{}}, headers: &MockHeaders{headers: []int{}}}
	HandleErrorResponse(w, 500, "message")
	assert.DeepEqual(t, w.headers.headers, []int{500})
	assert.DeepEqual(t, w.buffer.buffer, []byte("{\"detail\":\"message\"}\n"))
}

func TestCreateNewConnectionFail(t *testing.T) {
	log := logrus.StandardLogger()
	handler := CreateNewConnection(log, 60)
	w := MockResponseWriter{buffer: &MockBuffer{buffer: []byte{}}, headers: &MockHeaders{headers: []int{}}}
	body := &MockBody{Body: "bad json"}
	r := &http.Request{Body: body}
	handler(w, r)
	assert.DeepEqual(t, w.headers.headers, []int{400})
	assert.DeepEqual(t, w.buffer.buffer, []byte("{\"detail\":\"Failed to parse request\"}\n"))
}

func TestCreateNewConnectionOK(t *testing.T) {
	log := logrus.StandardLogger()
	handler := CreateNewConnection(log, 60)
	w := MockResponseWriter{buffer: &MockBuffer{buffer: []byte{}}, headers: &MockHeaders{headers: []int{}}}
	body := &MockBody{Body: "{\"access_token\":\"a\",\"refresh_token\":\"r\"}"}
	r := &http.Request{Body: body}
	handler(w, r)
	assert.DeepEqual(t, w.headers.headers, []int{200})
	data := &NewConnectionResponse{}
	err := json.Unmarshal(w.buffer.buffer, data)
	assert.NilError(t, err)
	assert.Assert(t, data.Username != "")
	at, rt := foodme.GlobalState.GetTokens(data.Username)
	assert.Equal(t, at, "a")
	assert.Equal(t, rt, "r")
	foodme.GlobalState.DeleteConnection(data.Username)
}

func TestApplyPermissionAgent(t *testing.T) {
	log := logrus.StandardLogger()
	conf, err := foodme.NewConfiguration([]string{"--destination-database-type", "postgres", "--destination-host", "localhost", "--destination-port", "5432"})
	assert.NilError(t, err)
	handler := ApplyPermissionAgent(log, conf, nil)

	// Bad input data
	w := MockResponseWriter{buffer: &MockBuffer{buffer: []byte{}}, headers: &MockHeaders{headers: []int{}}}
	r := &http.Request{Body: &MockBody{Body: "bad body"}}

	handler(w, r)
	assert.DeepEqual(t, w.headers.headers, []int{400})
	assert.DeepEqual(t, w.buffer.buffer, []byte("{\"detail\":\"Failed to parse request: invalid character 'b' looking for beginning of value\"}\n"))

	// Missing username
	w = MockResponseWriter{buffer: &MockBuffer{buffer: []byte{}}, headers: &MockHeaders{headers: []int{}}}
	r = &http.Request{Body: &MockBody{Body: "{}"}}
	handler(w, r)
	assert.DeepEqual(t, w.headers.headers, []int{400})
	assert.DeepEqual(t, w.buffer.buffer, []byte("{\"detail\":\"No username provided\"}\n"))

	// Missing SQL
	w = MockResponseWriter{buffer: &MockBuffer{buffer: []byte{}}, headers: &MockHeaders{headers: []int{}}}
	r = &http.Request{Body: &MockBody{Body: "{\"username\":\"test\"}"}}
	handler(w, r)
	assert.DeepEqual(t, w.headers.headers, []int{400})
	assert.DeepEqual(t, w.buffer.buffer, []byte("{\"detail\":\"No SQL provided\"}\n"))

	// OIDC disabled
	w = MockResponseWriter{buffer: &MockBuffer{buffer: []byte{}}, headers: &MockHeaders{headers: []int{}}}
	r = &http.Request{Body: &MockBody{Body: "{\"username\":\"test\", \"sql\":\"select * from pets\"}"}}
	handler(w, r)
	assert.DeepEqual(t, w.headers.headers, []int{424})
	assert.DeepEqual(t, w.buffer.buffer, []byte("{\"detail\":\"OIDC is disabled\"}\n"))

	// Permission agent disabled
	conf.OIDCEnabled = true
	w = MockResponseWriter{buffer: &MockBuffer{buffer: []byte{}}, headers: &MockHeaders{headers: []int{}}}
	r = &http.Request{Body: &MockBody{Body: "{\"username\":\"test\", \"sql\":\"select * from pets\"}"}}
	handler(w, r)
	assert.DeepEqual(t, w.headers.headers, []int{424})
	assert.DeepEqual(t, w.buffer.buffer, []byte("{\"detail\":\"Permission agent is disabled\"}\n"))

	// Missing tokens
	conf.PermissionAgentEnabled = true
	w = MockResponseWriter{buffer: &MockBuffer{buffer: []byte{}}, headers: &MockHeaders{headers: []int{}}}
	r = &http.Request{Body: &MockBody{Body: "{\"username\":\"test\", \"sql\":\"select * from pets\"}"}}
	handler(w, r)
	assert.DeepEqual(t, w.headers.headers, []int{404})
	assert.DeepEqual(t, w.buffer.buffer, []byte("{\"detail\":\"No tokens found for user test\"}\n"))

	// Missing client
	foodme.GlobalState.AddConnection("test", "a", "r", 60)
	w = MockResponseWriter{buffer: &MockBuffer{buffer: []byte{}}, headers: &MockHeaders{headers: []int{}}}
	r = &http.Request{Body: &MockBody{Body: "{\"username\":\"test\", \"sql\":\"select * from pets\"}"}}
	handler(w, r)
	assert.DeepEqual(t, w.headers.headers, []int{404})
	assert.DeepEqual(t, w.buffer.buffer, []byte("{\"detail\":\"No client found for database \"}\n"))

	// Bad tokens
	conf.OIDCDatabaseFallBackToBaseClient = true
	mockHttpClient := &MockHttpClient{DoSucceed: true, Response: []string{"bad response"}, StatusCode: 200}
	handler = ApplyPermissionAgent(log, conf, mockHttpClient)
	w = MockResponseWriter{buffer: &MockBuffer{buffer: []byte{}}, headers: &MockHeaders{headers: []int{}}}
	r = &http.Request{Body: &MockBody{Body: "{\"username\":\"test\", \"sql\":\"select * from pets\"}"}}
	handler(w, r)
	assert.DeepEqual(t, w.headers.headers, []int{401})
	assert.DeepEqual(t, w.buffer.buffer, []byte("{\"detail\":\"Failed to refresh access token: invalid character 'b' looking for beginning of value\"}\n"))

	// Bad user info
	mockHttpClient = &MockHttpClient{DoSucceed: true, Response: []string{"{\"access_token\":\"access\"}", "bad response"}, StatusCode: 200}
	handler = ApplyPermissionAgent(log, conf, mockHttpClient)
	w = MockResponseWriter{buffer: &MockBuffer{buffer: []byte{}}, headers: &MockHeaders{headers: []int{}}}
	r = &http.Request{Body: &MockBody{Body: "{\"username\":\"test\", \"sql\":\"select * from pets\"}"}}
	handler(w, r)
	assert.DeepEqual(t, w.headers.headers, []int{401})
	assert.DeepEqual(t, w.buffer.buffer, []byte("{\"detail\":\"Failed to get user info: invalid character 'b' looking for beginning of value\"}\n"))

	// Missing permission agent
	mockHttpClient = &MockHttpClient{DoSucceed: true, Response: []string{"{\"access_token\":\"access\"}", "{\"preferred_username\":\"test_user\"}"}, StatusCode: 200}
	handler = ApplyPermissionAgent(log, conf, mockHttpClient)
	w = MockResponseWriter{buffer: &MockBuffer{buffer: []byte{}}, headers: &MockHeaders{headers: []int{}}}
	r = &http.Request{Body: &MockBody{Body: "{\"username\":\"test\", \"sql\":\"select * from pets\"}"}}
	handler(w, r)
	assert.DeepEqual(t, w.headers.headers, []int{500})
	assert.DeepEqual(t, w.buffer.buffer, []byte("{\"detail\":\"Failed to create permission agent\"}\n"))

	// Missing SQL handler
	conf.PermissionAgentEnabled = true
	conf.PermissionAgentType = "opa"
	conf.DestinationDatabaseType = "bad"
	mockHttpClient = &MockHttpClient{DoSucceed: true, Response: []string{"{\"access_token\":\"access\"}", "{\"preferred_username\":\"test_user\"}"}, StatusCode: 200}
	handler = ApplyPermissionAgent(log, conf, mockHttpClient)
	w = MockResponseWriter{buffer: &MockBuffer{buffer: []byte{}}, headers: &MockHeaders{headers: []int{}}}
	r = &http.Request{Body: &MockBody{Body: "{\"username\":\"test\", \"sql\":\"select * from pets\"}"}}
	handler(w, r)
	assert.DeepEqual(t, w.headers.headers, []int{500})
	assert.DeepEqual(t, w.buffer.buffer, []byte("{\"detail\":\"Failed to create SQL handler: unknown database type: bad\"}\n"))

	// Fail to handle set DDL
	conf.DestinationDatabaseType = "postgres"
	mockHttpClient = &MockHttpClient{
		DoSucceed: true,
		Response: []string{
			"{\"access_token\":\"access\"}",
			"{\"preferred_username\":\"test_user\"}",
			"bad",
		},
		StatusCode: 200,
	}
	handler = ApplyPermissionAgent(log, conf, mockHttpClient)
	w = MockResponseWriter{buffer: &MockBuffer{buffer: []byte{}}, headers: &MockHeaders{headers: []int{}}}
	r = &http.Request{Body: &MockBody{Body: "{\"username\":\"test\", \"sql\":\"select * from pets\"}"}}
	handler(w, r)
	assert.DeepEqual(t, w.headers.headers, []int{500})
	assert.DeepEqual(t, w.buffer.buffer, []byte("{\"detail\":\"Failed to set DDL: failed to unmarshal response body: invalid character 'b' looking for beginning of value\"}\n"))

	// OK
	mockHttpClient = &MockHttpClient{
		DoSucceed: true,
		Response: []string{
			"{\"access_token\":\"access\"}",
			"{\"preferred_username\":\"test_user\"}",
			"{}",
			"{}",
			"{}",
			"{\"result\":{\"queries\":[[{\"terms\":[{\"type\":\"number\",\"value\":23},{\"type\":\"ref\",\"value\":[{\"type\":\"var\",\"value\":\"gte\"}]},{\"type\":\"ref\",\"value\":[{\"type\":\"var\",\"value\":\"data\"},{\"type\":\"string\",\"value\":\"tables\"},{\"type\":\"string\",\"value\":\"pets\"},{\"type\":\"string\",\"value\":\"owners\"}]}]}]]}}",
		},
		StatusCode: 200,
	}
	handler = ApplyPermissionAgent(log, conf, mockHttpClient)
	w = MockResponseWriter{buffer: &MockBuffer{buffer: []byte{}}, headers: &MockHeaders{headers: []int{}}}
	r = &http.Request{Body: &MockBody{Body: "{\"username\":\"test\", \"sql\":\"select * from pets\"}"}}
	handler(w, r)
	assert.DeepEqual(t, w.headers.headers, []int{200})
	respData := map[string]interface{}{}
	err = json.Unmarshal(w.buffer.buffer, &respData)
	assert.NilError(t, err)
	assert.DeepEqual(t, respData, map[string]interface{}{"sql": "select * from pets", "new_sql": "SELECT * FROM pets WHERE ((pets.owners >= 23))"})

	// OK with alias
	mockHttpClient = &MockHttpClient{
		DoSucceed: true,
		Response: []string{
			"{\"access_token\":\"access\"}",
			"{\"preferred_username\":\"test_user\"}",
			"{}",
			"{}",
			"{}",
			"{\"result\":{\"queries\":[[{\"terms\":[{\"type\":\"number\",\"value\":23},{\"type\":\"ref\",\"value\":[{\"type\":\"var\",\"value\":\"gte\"}]},{\"type\":\"ref\",\"value\":[{\"type\":\"var\",\"value\":\"data\"},{\"type\":\"string\",\"value\":\"tables\"},{\"type\":\"string\",\"value\":\"pets\"},{\"type\":\"string\",\"value\":\"owners\"}]}]}]]}}",
		},
		StatusCode: 200,
	}
	handler = ApplyPermissionAgent(log, conf, mockHttpClient)
	w = MockResponseWriter{buffer: &MockBuffer{buffer: []byte{}}, headers: &MockHeaders{headers: []int{}}}
	r = &http.Request{Body: &MockBody{Body: "{\"username\":\"test\", \"sql\":\"select * from pets p\"}"}}
	handler(w, r)
	assert.DeepEqual(t, w.headers.headers, []int{200})
	respData = map[string]interface{}{}
	err = json.Unmarshal(w.buffer.buffer, &respData)
	assert.NilError(t, err)
	assert.DeepEqual(t, respData, map[string]interface{}{"sql": "select * from pets p", "new_sql": "SELECT * FROM pets AS p WHERE ((p.owners >= 23))"})
}
