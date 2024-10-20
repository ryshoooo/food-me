package foodme

import (
	"net/http"
	"testing"

	"gotest.tools/v3/assert"
)

func TestPermissionAgentQuery(t *testing.T) {
	httpClient := &MockHttpClient{}
	pa := &HTTPPermissionAgent{client: httpClient}

	_, err := pa.query("not a good method", "bad endpoint", []byte("test"))
	assert.Error(t, err, "failed to create http request: net/http: invalid method \"not a good method\"")

	_, err = pa.query(http.MethodPost, "http://localhost:8080", []byte("test"))
	assert.Error(t, err, "failed to execute request: failed to do request")

	httpClient.DoSucceed = true
	httpClient.StatusCode = http.StatusNotFound
	_, err = pa.query(http.MethodPost, "http://localhost:8080", []byte("test"))
	assert.Error(t, err, "unexpected status code: 404")

	httpClient.StatusCode = http.StatusOK
	httpClient.FailBodyRead = true
	httpClient.Response = "hello"
	_, err = pa.query(http.MethodPost, "http://localhost:8080", []byte("test"))
	assert.Error(t, err, "failed to read response body: body read failure")

	httpClient.FailBodyRead = false
	resp, err := pa.query(http.MethodPost, "http://localhost:8080", []byte("test"))
	assert.NilError(t, err)
	assert.Equal(t, string(resp), "hello")
}
