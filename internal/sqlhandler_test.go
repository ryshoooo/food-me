package foodme

import (
	"testing"

	"github.com/sirupsen/logrus"
	"gotest.tools/v3/assert"
)

func TestNewPGSQLHandler(t *testing.T) {
	log := logrus.StandardLogger()
	h, err := NewSQLHandler("postgres", log, &DummyAgent{})
	assert.NilError(t, err)
	assert.Assert(t, h != nil)
	switch h.(type) {
	case *PostgresSQLHandler:
		assert.Assert(t, true)
	default:
		assert.Assert(t, false)
	}
}

func TestNewSQLHandlerFail(t *testing.T) {
	log := logrus.StandardLogger()
	_, err := NewSQLHandler("blah", log, &DummyAgent{})
	assert.Error(t, err, "unknown database type: blah")
}
