package api

import (
	"encoding/json"
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
	Body string
}

func (m *MockBody) Read(p []byte) (n int, err error) {
	b := []byte(m.Body)
	copy(p, b)
	return len(b), nil
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

func TestHandleErrorResponse(t *testing.T) {
	w := MockResponseWriter{buffer: &MockBuffer{buffer: []byte{}}, headers: &MockHeaders{headers: []int{}}}
	HandleErrorResponse(w, 500, "message")
	assert.DeepEqual(t, w.headers.headers, []int{500})
	assert.DeepEqual(t, w.buffer.buffer, []byte("{\"detail\":\"message\"}\n"))
}

func TestCreateNewConnectionFail(t *testing.T) {
	log := logrus.StandardLogger()
	handler := CreateNewConnection(log)
	w := MockResponseWriter{buffer: &MockBuffer{buffer: []byte{}}, headers: &MockHeaders{headers: []int{}}}
	body := &MockBody{Body: "bad json"}
	r := &http.Request{Body: body}
	handler(w, r)
	assert.DeepEqual(t, w.headers.headers, []int{400})
	assert.DeepEqual(t, w.buffer.buffer, []byte("{\"detail\":\"Failed to parse request\"}\n"))
}

func TestCreateNewConnectionOK(t *testing.T) {
	log := logrus.StandardLogger()
	handler := CreateNewConnection(log)
	w := MockResponseWriter{buffer: &MockBuffer{buffer: []byte{}}, headers: &MockHeaders{headers: []int{}}}
	body := &MockBody{Body: "{\"access_token\":\"a\",\"refresh_token\":\"r\"}"}
	r := &http.Request{Body: body}
	handler(w, r)
	assert.DeepEqual(t, w.headers.headers, []int{200})
	data := &NewConnectionResponse{}
	json.Unmarshal(w.buffer.buffer, data)
	assert.Assert(t, data.Username != "")
	at, rt := foodme.GlobalState.GetTokens(data.Username)
	assert.Equal(t, at, "a")
	assert.Equal(t, rt, "r")
	foodme.GlobalState.DeleteConnection(data.Username)
}
