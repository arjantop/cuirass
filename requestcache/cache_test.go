package requestcache

import (
	"errors"
	"testing"
	"time"

	"code.google.com/p/go.net/context"
	"github.com/arjantop/cuirass"
	"github.com/arjantop/cuirass/requestlog"
	"github.com/stretchr/testify/assert"
)

func NewNormalCommand() *cuirass.Command {
	return cuirass.NewCommand("Normal", func(ctx context.Context) (interface{}, error) {
		return "normal", nil
	}).Build()
}

func NewCachedCommand(cacheKey string) *cuirass.Command {
	return cuirass.NewCommand("Cached", func(ctx context.Context) (interface{}, error) {
		return "executed", nil
	}).CacheKey(cacheKey).Build()
}

func TestAddRequestSuccess(t *testing.T) {
	cache := newRequestCache()
	events := []requestlog.ExecutionEvent{requestlog.Success}
	info := requestlog.NewExecutionInfo("Cached", 10*time.Millisecond, events)
	assert.True(t, cache.Add(NewCachedCommand("a"), info, "abc", nil))

	// Request should now be retrieved from cache.
	ec := cache.Get(NewCachedCommand("a"))
	r, err := ec.Response()
	assert.Nil(t, err)
	assert.Equal(t, "abc", r)
	ei := ec.ExecutionInfo()
	assert.Equal(t, 0, ei.ExecutionTime())
	assert.Equal(t, []requestlog.ExecutionEvent{
		requestlog.Success,
		requestlog.ResponseFromCache}, ei.Events())

	// Same command with different cache key should not be cached.
	assert.Nil(t, cache.Get(NewCachedCommand("b")))
}

func TestAddRequestFailure(t *testing.T) {
	cache := newRequestCache()
	cachedError := errors.New("cached")
	events := []requestlog.ExecutionEvent{requestlog.Success}
	info := requestlog.NewExecutionInfo("Cached", 0, events)
	assert.True(t, cache.Add(NewCachedCommand("a"), info, "", cachedError))

	// Request should now be retrieved from cache.
	ec := cache.Get(NewCachedCommand("a"))
	r, err := ec.Response()
	assert.Equal(t, "", r)
	assert.Equal(t, cachedError, err)
}

func TestAddRequestNonCachable(t *testing.T) {
	cache := newRequestCache()
	info := requestlog.NewExecutionInfo("Normal", 10*time.Millisecond, nil)
	assert.False(t, cache.Add(NewNormalCommand(), info, "abc", nil))

	assert.Nil(t, cache.Get(NewNormalCommand()))
}
