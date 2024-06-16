package foodme

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestCalculatePacketSize(t *testing.T) {
	assert.Equal(t, calculatePacketSize([]byte{0, 0, 0, 0}), 0)
	assert.Equal(t, calculatePacketSize([]byte{0, 0, 0, 1}), 1)
	assert.Equal(t, calculatePacketSize([]byte{0, 0, 1, 0}), 256)
	assert.Equal(t, calculatePacketSize([]byte{0, 1, 0, 0}), 65536)
	assert.Equal(t, calculatePacketSize([]byte{1, 0, 0, 0}), 16777216)
	assert.Equal(t, calculatePacketSize([]byte{1, 1, 1, 1}), 16777216+65536+256+1)
}

func TestCreatePacketSize(t *testing.T) {
	assert.DeepEqual(t, createPacketSize(0), []byte{0, 0, 0, 0})
	assert.DeepEqual(t, createPacketSize(1), []byte{0, 0, 0, 1})
	assert.DeepEqual(t, createPacketSize(256), []byte{0, 0, 1, 0})
	assert.DeepEqual(t, createPacketSize(65536), []byte{0, 1, 0, 0})
	assert.DeepEqual(t, createPacketSize(16777216), []byte{1, 0, 0, 0})
	assert.DeepEqual(t, createPacketSize(16777216+65536+256+1), []byte{1, 1, 1, 1})
}

func TestCheckAuthenticationSuccess(t *testing.T) {
	assert.Assert(t, !checkAuthenticationSuccess([]byte{}))
	assert.Assert(t, !checkAuthenticationSuccess([]byte{1, 2, 3, 4}))
	assert.Assert(t, checkAuthenticationSuccess([]byte{0, 0, 0, 0}))
}

func TestGetErrorMessage(t *testing.T) {
	assert.Equal(t, getErrorMessage([]byte{}), "unknown error")
	assert.Equal(t, getErrorMessage([]byte{0, 0, 0, 0}), "unknown error")
	assert.Equal(t, getErrorMessage([]byte("Mcustom error message")), "custom error message")
	arr := []byte{0, 1, 2, 4, 0, 0, 2, 0}
	arr = append(arr, []byte("Mcustom error message")...)
	arr = append(arr, []byte{0, 3, 4, 2, 0, 0, 1, 1, 0}...)
	assert.Equal(t, getErrorMessage(arr), "custom error message")
}
