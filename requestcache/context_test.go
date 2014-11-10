package requestcache_test

import (
	"testing"

	"code.google.com/p/go.net/context"
	"github.com/arjantop/cuirass/requestcache"
	"github.com/arjantop/cuirass/requestlog"
	"github.com/stretchr/testify/assert"
)

func TestFromContext(t *testing.T) {
	ctx := requestcache.WithRequestCache(context.Background())
	cache := requestcache.FromContext(ctx)

	info := requestlog.NewExecutionInfo("Cached", 0, []requestlog.ExecutionEvent{})
	assert.True(t, cache.Add(requestcache.NewCachedCommand("b"), info, "", nil))

	// The same instance of cache should be returned.
	cache2 := requestcache.FromContext(ctx)
	assert.NotNil(t, cache2.Get(requestcache.NewCachedCommand("b")))
}
