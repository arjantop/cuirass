package requestcache_test

import (
	"testing"

	"github.com/arjantop/cuirass/requestcache"
	"github.com/arjantop/cuirass/requestlog"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
)

func TestFromContext(t *testing.T) {
	ctx := requestcache.WithRequestCache(context.Background())
	cache := requestcache.FromContext(ctx)

	info := requestlog.NewExecutionInfo("Cmd", 0, []requestlog.ExecutionEvent{})
	cache.Add("Cmd", "b", info, "", nil)

	// The same instance of cache should be returned.
	cache2 := requestcache.FromContext(ctx)
	assert.NotNil(t, cache2.Get("Cmd", "b"))
}
