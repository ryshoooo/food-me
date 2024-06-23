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

func TestGetExpiredUsername(t *testing.T) {
	testState := &State{Connections: make(map[string]Connection), Mutex: sync.RWMutex{}}

	testState.AddConnection("test", "a", "r", 0)
	testState.AddConnection("test2", "a", "r", 0)
	testState.AddConnection("test3", "a", "r", 60)

	expired := testState.GetExpiredUsernames()
	assert.Equal(t, 2, len(expired))

	var containsTest, containsTest2, containsTest3 bool
	for _, username := range expired {
		if username == "test" {
			containsTest = true
		}
		if username == "test2" {
			containsTest2 = true
		}
		if username == "test3" {
			containsTest3 = true
		}
	}
	assert.Assert(t, containsTest)
	assert.Assert(t, containsTest2)
	assert.Assert(t, !containsTest3)
}
