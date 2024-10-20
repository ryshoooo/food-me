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

func TestPermissionAgentDDLQuery(t *testing.T) {
	httpClient := &MockHttpClient{}
	pa := &HTTPPermissionAgent{client: httpClient}

	_, err := pa.ddlQuery("op", map[string]interface{}{"a": map[interface{}]bool{nil: false}})
	assert.Error(t, err, "failed to marshal json payload: json: unsupported type: map[interface {}]bool")

	_, err = pa.ddlQuery("op", map[string]interface{}{"a": "b"})
	assert.Error(t, err, "failed to query ddl: failed to execute request: failed to do request")

	httpClient.DoSucceed = true
	httpClient.StatusCode = http.StatusOK
	_, err = pa.ddlQuery("op", map[string]interface{}{"a": "b"})
	assert.Error(t, err, "failed to unmarshal response body: unexpected end of JSON input")

	httpClient.Response = `{"allowed": true}`
	resp, err := pa.ddlQuery("op", map[string]interface{}{"a": "b"})
	assert.NilError(t, err)
	assert.Assert(t, resp.Allowed)

	httpClient.Response = `{"allowed": false}`
	resp, err = pa.ddlQuery("op", map[string]interface{}{"a": "b"})
	assert.NilError(t, err)
	assert.Assert(t, !resp.Allowed)
}

func TestPermissionAgentCreateAllowed(t *testing.T) {
	httpClient := &MockHttpClient{}
	pa := &HTTPPermissionAgent{client: httpClient}

	err := pa.SetCreateAllowed(map[string]interface{}{"a": "b"})
	assert.Error(t, err, "failed to query ddl: failed to query ddl: failed to execute request: failed to do request")

	httpClient.DoSucceed = true
	httpClient.StatusCode = http.StatusOK
	httpClient.Response = `{"allowed": true}`
	err = pa.SetCreateAllowed(map[string]interface{}{"a": "b"})
	assert.NilError(t, err)
	assert.Assert(t, pa.CreateAllowed())

	httpClient.Response = `{"allowed": false}`
	err = pa.SetCreateAllowed(map[string]interface{}{"a": "b"})
	assert.NilError(t, err)
	assert.Assert(t, !pa.CreateAllowed())
}

func TestPermissionAgentUpdateAllowed(t *testing.T) {
	httpClient := &MockHttpClient{}
	pa := &HTTPPermissionAgent{client: httpClient}

	err := pa.SetUpdateAllowed(map[string]interface{}{"a": "b"})
	assert.Error(t, err, "failed to query ddl: failed to query ddl: failed to execute request: failed to do request")

	httpClient.DoSucceed = true
	httpClient.StatusCode = http.StatusOK
	httpClient.Response = `{"allowed": true}`
	err = pa.SetUpdateAllowed(map[string]interface{}{"a": "b"})
	assert.NilError(t, err)
	assert.Assert(t, pa.UpdateAllowed())

	httpClient.Response = `{"allowed": false}`
	err = pa.SetUpdateAllowed(map[string]interface{}{"a": "b"})
	assert.NilError(t, err)
	assert.Assert(t, !pa.UpdateAllowed())
}

func TestPermissionAgentDeleteAllowed(t *testing.T) {
	httpClient := &MockHttpClient{}
	pa := &HTTPPermissionAgent{client: httpClient}

	err := pa.SetDeleteAllowed(map[string]interface{}{"a": "b"})
	assert.Error(t, err, "failed to query ddl: failed to query ddl: failed to execute request: failed to do request")

	httpClient.DoSucceed = true
	httpClient.StatusCode = http.StatusOK
	httpClient.Response = `{"allowed": true}`
	err = pa.SetDeleteAllowed(map[string]interface{}{"a": "b"})
	assert.NilError(t, err)
	assert.Assert(t, pa.DeleteAllowed())

	httpClient.Response = `{"allowed": false}`
	err = pa.SetDeleteAllowed(map[string]interface{}{"a": "b"})
	assert.NilError(t, err)
	assert.Assert(t, !pa.DeleteAllowed())
}

func TestPermissionAgentSelectFilters(t *testing.T) {
	httpClient := &MockHttpClient{}
	pa := &HTTPPermissionAgent{client: httpClient}

	_, err := pa.SelectFilters("table", "alias", map[string]interface{}{"a": map[interface{}]bool{nil: false}})
	assert.Error(t, err, "failed to marshal json payload: json: unsupported type: map[interface {}]bool")

	_, err = pa.SelectFilters("table", "alias", map[string]interface{}{"a": "b"})
	assert.Error(t, err, "failed to query select filters: failed to execute request: failed to do request")

	httpClient.DoSucceed = true
	httpClient.StatusCode = http.StatusOK
	_, err = pa.SelectFilters("table", "alias", map[string]interface{}{"a": "b"})
	assert.Error(t, err, "failed to unmarshal response body: unexpected end of JSON input")

	httpClient.Response = `{"allowed": false}`
	_, err = pa.SelectFilters("table", "alias", map[string]interface{}{"a": "b"})
	assert.Error(t, err, "permission denied to access table table")

	httpClient.Response = `{"allowed": true}`
	resp, err := pa.SelectFilters("table", "alias", map[string]interface{}{"a": "b"})
	assert.NilError(t, err)
	assert.DeepEqual(t, resp, &SelectFilters{WhereFilters: []string{}, JoinFilters: []*JoinFilter{}})

	httpClient.Response = `{"allowed": true, "filters": {"whereFilters": ["a = 1"], "joinFilters": [{"tableName": "table", "conditions": "a = b"}]}}`
	resp, err = pa.SelectFilters("table", "alias", map[string]interface{}{"a": "b"})
	assert.NilError(t, err)
	assert.DeepEqual(t, resp, &SelectFilters{WhereFilters: []string{"a = 1"}, JoinFilters: []*JoinFilter{{TableName: "table", Conditions: "a = b"}}})
}
