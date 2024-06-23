package foodme

import (
	"sync"
	"testing"
	"time"

	"gotest.tools/v3/assert"
)

func TestState(t *testing.T) {
	testState := &State{Connections: make(map[string]Connection), Mutex: sync.RWMutex{}}

	// Should be empty
	a, r := testState.GetTokens("test")
	assert.Equal(t, "", a)
	assert.Equal(t, "", r)

	// Add connection
	testState.AddConnection("test", "a", "r", 60)
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

	// Add connection with short lifetime
	testState.AddConnection("test", "a", "r", 0)
	a, r = testState.GetTokens("test")
	assert.Equal(t, "", a)
	assert.Equal(t, "", r)
}

func TestIsConnectionAlive(t *testing.T) {
	c := Connection{ExpiresIn: 100}
	assert.Assert(t, !c.IsAlive())
	c = Connection{ExpiresIn: time.Now().Add(time.Duration(20) * time.Second).Unix()}
	assert.Assert(t, c.IsAlive())
}
