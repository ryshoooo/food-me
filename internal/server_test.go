package foodme

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestNewServer(t *testing.T) {
	server := NewServer(nil, nil)
	assert.Assert(t, server.Configuration == nil)
	assert.Assert(t, server.Logger == nil)
}
