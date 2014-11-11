package requestcache

import (
	"errors"
	"testing"
	"time"

	"github.com/arjantop/cuirass/requestlog"
	"github.com/stretchr/testify/assert"
)

func TestAddRequestSuccess(t *testing.T) {
	cache := newRequestCache()
	events := []requestlog.ExecutionEvent{requestlog.Success}
	info := requestlog.NewExecutionInfo("Cached", 10*time.Millisecond, events)
	cache.Add("Cached", "a", info, "abc", nil)

	// Request should now be retrieved from cache.
	ec := cache.Get("Cached", "a")
	r, err := ec.Response()
	assert.Nil(t, err)
	assert.Equal(t, "abc", r)
	ei := ec.ExecutionInfo()
	assert.Equal(t, 0, ei.ExecutionTime())
	assert.Equal(t, []requestlog.ExecutionEvent{
		requestlog.Success,
		requestlog.ResponseFromCache}, ei.Events())

	// Same command with different cache key should not be cached.
	assert.Nil(t, cache.Get("Cached", "b"))
}

func TestAddRequestFailure(t *testing.T) {
	cache := newRequestCache()
	cachedError := errors.New("cached")
	events := []requestlog.ExecutionEvent{requestlog.Success}
	info := requestlog.NewExecutionInfo("Cached", 0, events)
	cache.Add("Cached", "a", info, "", cachedError)

	// Request should now be retrieved from cache.
	ec := cache.Get("Cached", "a")
	r, err := ec.Response()
	assert.Equal(t, "", r)
	assert.Equal(t, cachedError, err)
}
