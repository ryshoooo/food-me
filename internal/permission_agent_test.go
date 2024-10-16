package foodme

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestNewPermissionAgent(t *testing.T) {
	conf, err := NewConfiguration([]string{"--destination-database-type", "postgres", "--destination-host", "localhost", "--destination-port", "5432"})
	assert.NilError(t, err)
	agent := NewPermissionAgent(conf, nil)
	assert.Assert(t, agent == nil)

	conf, err = NewConfiguration([]string{"--destination-database-type", "postgres", "--destination-host", "localhost", "--destination-port", "5432", "--permission-agent-type", "opa"})
	assert.NilError(t, err)
	agent = NewPermissionAgent(conf, nil)
	assert.Assert(t, agent != nil)

	conf, err = NewConfiguration([]string{"--destination-database-type", "postgres", "--destination-host", "localhost", "--destination-port", "5432", "--permission-agent-type", "http"})
	assert.NilError(t, err)
	agent = NewPermissionAgent(conf, nil)
	assert.Assert(t, agent != nil)
}
