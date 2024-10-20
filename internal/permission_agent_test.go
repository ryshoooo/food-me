package foodme

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestNewPermissionAgent(t *testing.T) {
	conf, err := NewConfiguration([]string{"--destination-database-type", "postgres", "--destination-host", "localhost", "--destination-port", "5432"})
	assert.NilError(t, err)
	_, err = NewPermissionAgent(conf, nil)
	assert.Error(t, err, "unknown permission agent type: ")

	conf, err = NewConfiguration([]string{"--destination-database-type", "postgres", "--destination-host", "localhost", "--destination-port", "5432", "--permission-agent-type", "opa"})
	assert.NilError(t, err)
	agent, err := NewPermissionAgent(conf, nil)
	assert.NilError(t, err)
	assert.Assert(t, agent != nil)

	conf, err = NewConfiguration([]string{"--destination-database-type", "postgres", "--destination-host", "localhost", "--destination-port", "5432", "--permission-agent-type", "http"})
	assert.NilError(t, err)
	agent, err = NewPermissionAgent(conf, nil)
	assert.NilError(t, err)
	assert.Assert(t, agent != nil)
}
