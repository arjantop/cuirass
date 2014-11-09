package requestlog

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAddRequestToLog(t *testing.T) {
	logger := newRequestLog()
	assert.Equal(t, 0, logger.Size())
	logger.AddRequest("Foo")
	assert.Equal(t, 1, logger.Size())
	logger.AddRequest("Foo")
	assert.Equal(t, 2, logger.Size())
	logger.AddRequest("Bar")
	logger.AddRequest("Baz")
	assert.Equal(t, 4, logger.Size())
}

func TestAddRequestString(t *testing.T) {
	logger := newRequestLog()
	commands := []string{"Foo", "Bar", "Baz"}
	logger.AddRequest(commands[0])
	assert.Equal(t, "Foo", logger.String())
	logger.AddRequest(commands[1])
	logger.AddRequest(commands[2])
	assert.Equal(t, "Foo, Bar, Baz", logger.String())
}

func TestAddRequestStringCommandCollapsing(t *testing.T) {
	logger := newRequestLog()
	commands := []string{"Foo", "Bar", "Baz"}
	logger.AddRequest(commands[0])
	logger.AddRequest(commands[0])
	assert.Equal(t, "Foox2", logger.String())
	logger.AddRequest(commands[2])
	assert.Equal(t, "Foox2, Baz", logger.String())
	logger.AddRequest(commands[0])
	logger.AddRequest(commands[0])
	logger.AddRequest(commands[0])
	assert.Equal(t, "Foox2, Baz, Foox3", logger.String())
}
