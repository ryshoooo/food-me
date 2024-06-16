package foodme

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestNewServer(t *testing.T) {
	server := NewServer(10000, nil, nil)
	assert.Equal(t, server.Port, 10000)
	assert.Assert(t, server.Logger == nil)
	assert.Assert(t, server.Handler == nil)
}
