package foodme

import (
	"sync"
	"testing"

	"gotest.tools/v3/assert"
)

func TestState(t *testing.T) {
	testState := &State{Connections: make(map[string]Connection), Mutex: sync.RWMutex{}}

	// Should be empty
	a, r := testState.GetTokens("test")
	assert.Equal(t, "", a)
	assert.Equal(t, "", r)

	// Add connection
	testState.AddConnection("test", "a", "r")
	a, r = testState.GetTokens("test")
	assert.Equal(t, "a", a)
	assert.Equal(t, "r", r)

	// Remove connection
	testState.DeleteConnection("test")
	a, r = testState.GetTokens("test")
	assert.Equal(t, "", a)
	assert.Equal(t, "", r)

	// Remove non-existing connection
	testState.DeleteConnection("test")
	a, r = testState.GetTokens("test")
	assert.Equal(t, "", a)
	assert.Equal(t, "", r)
}
