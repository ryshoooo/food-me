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
	Writes      [][]byte
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
			return 0, fmt.Errorf("buffer size mismatch: %d != %d", len(buff), len(m.Responses[m.ResponseIdx]))
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
		m.Writes = append(m.Writes, buff)
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
	assert.Error(t, err, "buffer size mismatch: 4 != 0")
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
	assert.Error(t, err, "buffer size mismatch: 1 != 0")
	assert.DeepEqual(t, res, []byte{})

	// Bad PG response to startup
	mc = &MockNetConn{Responses: [][]byte{{0, 0, 0, 8}, {1, 2, 3, 4}}}
	mu = &MockNetConn{Responses: [][]byte{[]byte("Q")}}
	handler.client = mc
	handler.upstream = mu
	res, err = handler.startup()
	assert.Error(t, err, "unexpected response from upstream: [81]")
	assert.DeepEqual(t, res, []byte{})

	// N response - client write fail
	mc = &MockNetConn{Responses: [][]byte{{0, 0, 0, 8}, {1, 2, 3, 4}}, FailWrite: true}
	mu = &MockNetConn{Responses: [][]byte{[]byte("N")}}
	handler.client = mc
	handler.upstream = mu
	res, err = handler.startup()
	assert.Error(t, err, "write failed")
	assert.DeepEqual(t, res, []byte{})

	// N response - client read fail
	mc = &MockNetConn{Responses: [][]byte{{0, 0, 0, 8}, {1, 2, 3, 4}, {}}}
	mu = &MockNetConn{Responses: [][]byte{[]byte("N")}}
	handler.client = mc
	handler.upstream = mu
	res, err = handler.startup()
	assert.Error(t, err, "buffer size mismatch: 4 != 0")
	assert.DeepEqual(t, res, []byte{})

	// N response - OK
	mc = &MockNetConn{Responses: [][]byte{{0, 0, 0, 8}, {1, 2, 3, 4}, {0, 0, 0, 1}}}
	mu = &MockNetConn{Responses: [][]byte{[]byte("N")}}
	handler.client = mc
	handler.upstream = mu
	res, err = handler.startup()
	assert.NilError(t, err)
	assert.DeepEqual(t, res, []byte{0, 0, 0, 1})
}

func TestPGHandlerStartupTLS(t *testing.T) {
	logger := logrus.StandardLogger()
	handler := NewPostgresHandler("addr", "user", "pwd", nil, logger, false, false, false, nil, "clientId", "clientSecret", "token-url", "userinfo-url", false, nil, "", nil, false, "", "", false, "", false)

	// Client write fail
	mc := &MockNetConn{Responses: [][]byte{{0, 0, 0, 8}, {1, 2, 3, 4}}, FailWrite: true}
	mu := &MockNetConn{Responses: [][]byte{[]byte("S")}}
	handler.client = mc
	handler.upstream = mu
	res, err := handler.startup()
	assert.Error(t, err, "write failed")
	assert.DeepEqual(t, res, []byte{})

	// TLS disabled, OK
	mc = &MockNetConn{Responses: [][]byte{{0, 0, 0, 8}, {1, 2, 3, 4}, {0, 0, 0, 1}}}
	mu = &MockNetConn{Responses: [][]byte{[]byte("S")}}
	handler.client = mc
	handler.upstream = mu
	res, err = handler.startup()
	assert.NilError(t, err)
	assert.DeepEqual(t, res, []byte{0, 0, 0, 1})

	// TLS enabled, fail to load keys
	handler.TLSEnabled = true
	mc = &MockNetConn{Responses: [][]byte{{0, 0, 0, 8}, {1, 2, 3, 4}, {0, 0, 0, 1}}}
	mu = &MockNetConn{Responses: [][]byte{[]byte("S")}}
	handler.client = mc
	handler.upstream = mu
	res, err = handler.startup()
	assert.Error(t, err, "open : no such file or directory")
	assert.DeepEqual(t, res, []byte{})

	// TLS OK, fail to write to client
	handler.TLSCertificateFile = "../data/cert.pem"
	handler.TLSCertificateKeyFile = "../data/key.pem"
	mc = &MockNetConn{Responses: [][]byte{{0, 0, 0, 8}, {1, 2, 3, 4}, {0, 0, 0, 1}}, FailWrite: true}
	mu = &MockNetConn{Responses: [][]byte{[]byte("S")}}
	handler.client = mc
	handler.upstream = mu
	res, err = handler.startup()
	assert.Error(t, err, "write failed")
	assert.DeepEqual(t, res, []byte{})

	// TLS OK (expected failure as tests dont do TLS exchange)
	mc = &MockNetConn{Responses: [][]byte{{0, 0, 0, 8}, {1, 2, 3, 4}, {0}}}
	mu = &MockNetConn{Responses: [][]byte{[]byte("S")}}
	handler.client = mc
	handler.upstream = mu
	res, err = handler.startup()
	assert.Error(t, err, "buffer size mismatch: 576 != 1")
	assert.DeepEqual(t, res, []byte{})
}
