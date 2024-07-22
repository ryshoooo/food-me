package foodme

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"gotest.tools/v3/assert"
)

type MockNetConn struct {
	FailRead    bool
	FailWrite   bool
	Responses   [][]byte
	ResponseIdx int
}

func (m *MockNetConn) Close() error {
	return nil
}

func (m *MockNetConn) LocalAddr() net.Addr {
	return nil
}

func (m *MockNetConn) RemoteAddr() net.Addr {
	return nil
}

func (m *MockNetConn) Read(buff []byte) (int, error) {
	if m.FailRead {
		return 0, fmt.Errorf("read failed")
	} else {
		if len(buff) != len(m.Responses[m.ResponseIdx]) {
			return 0, fmt.Errorf("buffer size mismatch")
		}
		copy(buff, m.Responses[m.ResponseIdx])
		m.ResponseIdx++
		return len(buff), nil
	}
}

func (m *MockNetConn) SetDeadline(time.Time) error {
	return nil
}

func (m *MockNetConn) SetReadDeadline(time.Time) error {
	return nil
}

func (m *MockNetConn) SetWriteDeadline(time.Time) error {
	return nil
}

func (m *MockNetConn) Write(buff []byte) (int, error) {
	if m.FailWrite {
		return 0, fmt.Errorf("write failed")
	} else {
		return len(buff), nil
	}
}

func TestNewPGHandler(t *testing.T) {
	logger := logrus.StandardLogger()
	handler := NewPostgresHandler("addr", "user", "pwd", nil, logger, false, false, false, nil, "clientId", "clientSecret", "token-url", "userinfo-url", false, nil, "", nil, false, "", "", false, "", false)
	assert.Assert(t, handler != nil)
}

func TestPGHandlerStartup(t *testing.T) {
	logger := logrus.StandardLogger()
	handler := NewPostgresHandler("addr", "user", "pwd", nil, logger, false, false, false, nil, "clientId", "clientSecret", "token-url", "userinfo-url", false, nil, "", nil, false, "", "", false, "", false)

	// Fail read
	handler.client = &MockNetConn{FailRead: true}
	res, err := handler.startup()
	assert.Error(t, err, "read failed")
	assert.DeepEqual(t, res, []byte{})

	// Not a startup packet
	mc := &MockNetConn{Responses: [][]byte{{0, 0, 0, 9}}}
	handler.client = mc
	res, err = handler.startup()
	assert.NilError(t, err)
	assert.DeepEqual(t, res, []byte{0, 0, 0, 9})

	// Fail mid-startup
	mc = &MockNetConn{Responses: [][]byte{{0, 0, 0, 8}, {}}}
	handler.client = mc
	res, err = handler.startup()
	assert.Error(t, err, "buffer size mismatch")
	assert.DeepEqual(t, res, []byte{})

	// Fail write to upstream
	mc = &MockNetConn{Responses: [][]byte{{0, 0, 0, 8}, {1, 2, 3, 4}}}
	mu := &MockNetConn{FailWrite: true}
	handler.client = mc
	handler.upstream = mu
	res, err = handler.startup()
	assert.Error(t, err, "write failed")
	assert.DeepEqual(t, res, []byte{})

	// Fail read from upstream
	mc = &MockNetConn{Responses: [][]byte{{0, 0, 0, 8}, {1, 2, 3, 4}}}
	mu = &MockNetConn{Responses: [][]byte{{}}}
	handler.client = mc
	handler.upstream = mu
	res, err = handler.startup()
	assert.Error(t, err, "buffer size mismatch")
	assert.DeepEqual(t, res, []byte{})

	// Bad PG response to startup
	mc = &MockNetConn{Responses: [][]byte{{0, 0, 0, 8}, {1, 2, 3, 4}}}
	mu = &MockNetConn{Responses: [][]byte{[]byte("Q")}}
	handler.client = mc
	handler.upstream = mu
	res, err = handler.startup()
	assert.Error(t, err, "unexpected response from upstream: [81]")
	assert.DeepEqual(t, res, []byte{})
}
