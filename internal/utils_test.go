package foodme

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestContains(t *testing.T) {
	assert.Assert(t, contains([]int{1, 2, 3}, 1))
	assert.Assert(t, !contains([]string{"1", "2", "3"}, "4"))
}
