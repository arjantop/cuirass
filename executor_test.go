package cuirass_test

import (
	"errors"
	"testing"
	"time"

	"github.com/arjantop/cuirass"
	"github.com/arjantop/cuirass/circuitbreaker"
	"github.com/arjantop/cuirass/metrics"
	"github.com/arjantop/cuirass/requestcache"
	"github.com/arjantop/cuirass/requestlog"
	"github.com/arjantop/cuirass/util"
	"github.com/arjantop/vaquita"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
)

func NewFooCommand(s, f string) *cuirass.Command {
	b := cuirass.NewCommand("FooCommand", func(ctx context.Context) (interface{}, error) {
		if s == "error" {
			return nil, errors.New("foo")
		} else if s == "panic" {
			panic("foopanic")
		} else if s == "panicint" {
			panic(1)
		}
		return s, nil
	}).Fallback(func(ctx context.Context) (interface{}, error) {
		if f == "none" {
			return nil, cuirass.FallbackNotImplemented
		} else if f == "error" {
			return nil, errors.New("fallbackerr")
		} else if f == "panic" {
			panic("fallpanic")
		}
		return f, nil
	})
	return b.Build()
}

func newTestingExecutor(cfg vaquita.DynamicConfig) *cuirass.CommandExecutor {
	if cfg == nil {
		cfg = vaquita.NewEmptyMapConfig()
	}
	cfg.SetProperty("cuirass.command.default.execution.isolation.thread.timeoutInMilliseconds", "100")
	return cuirass.NewExecutor(cfg)
}

func TestExecSuccessNoLogging(t *testing.T) {
	ctx := context.Background()
	cmd := NewFooCommand("foo", "")
	ex := newTestingExecutor(nil)
	_, err := ex.Exec(ctx, cmd)
	assert.Nil(t, err)
}

func TestExecSuccess(t *testing.T) {
	ctx := requestlog.WithRequestLog(context.Background())
	cmd := NewFooCommand("foo", "")
	ex := newTestingExecutor(nil)
	r, err := ex.Exec(ctx, cmd)
	assert.Nil(t, err)
	assert.Equal(t, "foo", r)
	request := requestlog.FromContext(ctx).LastRequest()
	assert.Equal(t, "FooCommand", request.CommandName())
	assert.Equal(t, []requestlog.ExecutionEvent{requestlog.Success}, request.Events())

}

func TestExecErrorWithFallback(t *testing.T) {
	ctx := requestlog.WithRequestLog(context.Background())
	cmd := NewFooCommand("error", "fallback")
	ex := newTestingExecutor(nil)
	r, err := ex.Exec(ctx, cmd)
	assert.Nil(t, err)
	assert.Equal(t, "fallback", r)

	request := requestlog.FromContext(ctx).LastRequest()
	assert.Equal(t, "FooCommand", request.CommandName())
	assert.Equal(t,
		[]requestlog.ExecutionEvent{requestlog.Failure, requestlog.FallbackSuccess},
		request.Events())
}

func TestExecErrorWithFallbackPanic(t *testing.T) {
	ctx := requestlog.WithRequestLog(context.Background())
	cmd := NewFooCommand("error", "panic")
	ex := newTestingExecutor(nil)
	_, err := ex.Exec(ctx, cmd)
	assert.Equal(t, errors.New("fallpanic"), err)

	request := requestlog.FromContext(ctx).LastRequest()
	assert.Equal(t, "FooCommand", request.CommandName())
	assert.Equal(t,
		[]requestlog.ExecutionEvent{requestlog.Failure, requestlog.FallbackFailure},
		request.Events())
}

func TestExecErrorWithoutFallback(t *testing.T) {
	ctx := requestlog.WithRequestLog(context.Background())
	cmd := NewFooCommand("error", "none")
	ex := newTestingExecutor(nil)
	_, err := ex.Exec(ctx, cmd)
	assert.Equal(t, errors.New("foo"), err)

	request := requestlog.FromContext(ctx).LastRequest()
	assert.Equal(t, "FooCommand", request.CommandName())
	assert.Equal(t,
		[]requestlog.ExecutionEvent{requestlog.Failure},
		request.Events())
}

func TestExecErrorWithoutFallbackFailure(t *testing.T) {
	ctx := requestlog.WithRequestLog(context.Background())
	cmd := NewFooCommand("error", "error")
	ex := newTestingExecutor(nil)
	_, err := ex.Exec(ctx, cmd)
	// The original error from Run is returned if Fallback fails too.
	assert.Equal(t, errors.New("foo"), err)

	request := requestlog.FromContext(ctx).LastRequest()
	assert.Equal(t, "FooCommand", request.CommandName())
	assert.Equal(t,
		[]requestlog.ExecutionEvent{requestlog.Failure, requestlog.FallbackFailure},
		request.Events())
}

func TestExecPanicWithFallback(t *testing.T) {
	ctx := requestlog.WithRequestLog(context.Background())
	cmd := NewFooCommand("panic", "fallback")
	ex := newTestingExecutor(nil)
	r, err := ex.Exec(ctx, cmd)
	assert.Nil(t, err)
	assert.Equal(t, "fallback", r)

	request := requestlog.FromContext(ctx).LastRequest()
	assert.Equal(t, "FooCommand", request.CommandName())
	assert.Equal(t,
		[]requestlog.ExecutionEvent{requestlog.Failure, requestlog.FallbackSuccess},
		request.Events())
}

func TestExecFallbackDisabled(t *testing.T) {
	ctx := requestlog.WithRequestLog(context.Background())
	cmd := NewFooCommand("error", "fallback")

	cfg := vaquita.NewEmptyMapConfig()
	cfg.SetProperty("cuirass.command.default.fallback.enabled", "false")
	ex := newTestingExecutor(cfg)
	_, err := ex.Exec(ctx, cmd)
	assert.Equal(t, errors.New("foo"), err)

	request := requestlog.FromContext(ctx).LastRequest()
	assert.Equal(t, "FooCommand", request.CommandName())
	assert.Equal(t,
		[]requestlog.ExecutionEvent{requestlog.Failure},
		request.Events())
}

func TestExecPanicWithoutFallback(t *testing.T) {
	ctx := requestlog.WithRequestLog(context.Background())
	cmd := NewFooCommand("panic", "none")
	ex := newTestingExecutor(nil)
	_, err := ex.Exec(ctx, cmd)
	assert.Equal(t, errors.New("foopanic"), err)

	request := requestlog.FromContext(ctx).LastRequest()
	assert.Equal(t, "FooCommand", request.CommandName())
	assert.Equal(t,
		[]requestlog.ExecutionEvent{requestlog.Failure},
		request.Events())
}

func TestExecIntPanicWithoutFallback(t *testing.T) {
	ctx := requestlog.WithRequestLog(context.Background())
	cmd := NewFooCommand("panicint", "none")
	ex := newTestingExecutor(nil)
	_, err := ex.Exec(ctx, cmd)
	assert.Equal(t, cuirass.UnknownPanic, err)

	request := requestlog.FromContext(ctx).LastRequest()
	assert.Equal(t, "FooCommand", request.CommandName())
	assert.Equal(t,
		[]requestlog.ExecutionEvent{requestlog.Failure},
		request.Events())
}

func TestExecFailuresTripCircuitBreaker(t *testing.T) {
	ctx := requestlog.WithRequestLog(context.Background())
	cmd := NewFooCommand("error", "none")
	cfg := vaquita.NewEmptyMapConfig()
	clock := util.NewTestableClock(time.Now())
	ex := cuirass.NewExecutorWithClock(cfg, clock)
	assert.False(t, ex.IsCircuitBreakerOpen("FooCommand"))
	for i := 0; i < 20; i++ {
		_, err := ex.Exec(ctx, cmd)
		assert.Equal(t, errors.New("foo"), err)
	}
	clock.Add(metrics.HealthSnapshotIntervalDefault + 1)
	_, err := ex.Exec(ctx, cmd)
	assert.Equal(t, circuitbreaker.CircuitOpenError, err)
	assert.True(t, ex.IsCircuitBreakerOpen("FooCommand"))

	request := requestlog.FromContext(ctx).LastRequest()
	assert.Equal(t, "FooCommand", request.CommandName())
	assert.Equal(t,
		[]requestlog.ExecutionEvent{requestlog.ShortCircuited},
		request.Events())
}

func TestExecRequestLogging(t *testing.T) {
	ctx := requestlog.WithRequestLog(context.Background())
	cmd := NewFooCommand("foo", "")
	ex := newTestingExecutor(nil)

	ex.Exec(ctx, cmd)
	log := requestlog.FromContext(ctx)
	assert.Equal(t, 1, log.Size())

	cmd2 := NewFooCommand("panic", "none")
	ex.Exec(ctx, cmd2)
	assert.Equal(t, 2, log.Size())

	cmd3 := NewFooCommand("error", "panic")
	ex.Exec(ctx, cmd3)
	assert.Equal(t, 3, log.Size())
}

func TestExecRequestLogDisabled(t *testing.T) {
	ctx := requestlog.WithRequestLog(context.Background())
	cmd := NewFooCommand("foo", "")
	cfg := vaquita.NewEmptyMapConfig()
	cfg.SetProperty("cuirass.command.default.requestLog.enabled", "false")
	ex := newTestingExecutor(cfg)

	ex.Exec(ctx, cmd)
	log := requestlog.FromContext(ctx)
	assert.Equal(t, 0, log.Size())
}

func NewTimeoutCommand(c <-chan time.Time, group string) *cuirass.Command {
	if c == nil {
		c = time.After(time.Second)
	}
	return cuirass.NewCommand("TimeoutCommand", func(ctx context.Context) (interface{}, error) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-c:
			return 0, nil
		}
	}).Group(group).Build()
}

func TestExecTimesOut(t *testing.T) {
	ctx := requestlog.WithRequestLog(context.Background())
	cmd := NewTimeoutCommand(nil, "Group")
	cfg := vaquita.NewEmptyMapConfig()
	cfg.SetProperty("cuirass.command.default.execution.isolation.thread.timeoutInMilliseconds", "1")
	ex := cuirass.NewExecutor(cfg)
	_, err := ex.Exec(ctx, cmd)
	assert.Equal(t, context.DeadlineExceeded, err)

	request := requestlog.FromContext(ctx).LastRequest()
	assert.Equal(t, "TimeoutCommand", request.CommandName())
	assert.Equal(t,
		[]requestlog.ExecutionEvent{requestlog.Timeout},
		request.Events())
}

func TestExecSemaphoreRejected(t *testing.T) {
	ctx := requestlog.WithRequestLog(context.Background())

	cfg := vaquita.NewEmptyMapConfig()
	cfg.SetProperty("cuirass.command.default.execution.isolation.semaphore.maxConcurrentRequests", "1")
	ex := cuirass.NewExecutor(cfg)

	c1 := make(chan time.Time)
	cmd1 := NewTimeoutCommand(c1, "FooCommand")
	go ex.Exec(ctx, cmd1)
	time.Sleep(time.Millisecond)

	cmd2 := NewFooCommand("foo", "none")
	_, err := ex.Exec(ctx, cmd2)
	assert.Equal(t, cuirass.SemaphoreRejected, err)

	request := requestlog.FromContext(ctx).LastRequest()
	assert.Equal(t, "FooCommand", request.CommandName())
	assert.Equal(t,
		[]requestlog.ExecutionEvent{requestlog.SemaphoreRejected},
		request.Events())

	c1 <- time.Now()
	time.Sleep(time.Millisecond)

	cmd3 := NewFooCommand("foo", "none")
	r, err := ex.Exec(ctx, cmd3)
	assert.NoError(t, err)
	assert.Equal(t, "foo", r)
}

func NewCachableCommand(s, f, key string) *cuirass.Command {
	return cuirass.NewCommand("Cachable", func(ctx context.Context) (interface{}, error) {
		if s == "error" {
			return nil, errors.New("foo")
		}
		return s, nil
	}).Fallback(func(ctx context.Context) (interface{}, error) {
		if f == "none" {
			return nil, cuirass.FallbackNotImplemented
		}
		return f, nil
	}).CacheKey(key).Build()
}

func TestExecSuccessCacheableCommandIsCached(t *testing.T) {
	ctx := requestlog.WithRequestLog(requestcache.WithRequestCache(context.Background()))
	ex := newTestingExecutor(nil)

	cmd := NewCachableCommand("foo", "", "a")
	r, err := ex.Exec(ctx, cmd)
	assert.Nil(t, err)
	assert.Equal(t, "foo", r)

	// This execution should return the value "foo" from cache instead of returning
	// the value "bar" that woul be returned in the case of evaluation the command.
	assertCommandFromCache(t, ex, ctx)

	// Multiple responses from cache should always respond with the same result.
	assertCommandFromCache(t, ex, ctx)

	// This command has a different cache key so it should be evaluated.
	cmd2 := NewCachableCommand("baz", "", "b")
	r, err = ex.Exec(ctx, cmd2)
	assert.Nil(t, err)
	assert.Equal(t, "baz", r)
}

func assertCommandFromCache(t *testing.T, ex *cuirass.CommandExecutor, ctx context.Context) {
	cmd := NewCachableCommand("bar", "", "a")
	r, err := ex.Exec(ctx, cmd)
	assert.Nil(t, err)
	assert.Equal(t, "foo", r)

	request := requestlog.FromContext(ctx).LastRequest()
	assert.Equal(t, "Cachable", request.CommandName())
	assert.Equal(t, 0, request.ExecutionTime())
	assert.Equal(t,
		[]requestlog.ExecutionEvent{requestlog.Success, requestlog.ResponseFromCache},
		request.Events())
}

func TestExecFallbackCacheableCommandIsCached(t *testing.T) {
	ctx := requestcache.WithRequestCache(context.Background())
	ex := newTestingExecutor(nil)

	cmd := NewCachableCommand("error", "foo", "a")
	r, err := ex.Exec(ctx, cmd)
	assert.Nil(t, err)
	assert.Equal(t, "foo", r)

	// The fallback value of previous command execution should be returned.
	cmd2 := NewCachableCommand("bar", "", "a")
	r, err = ex.Exec(ctx, cmd2)
	assert.Nil(t, err)
	assert.Equal(t, "foo", r)
}

func TestExecContextLocalCaching(t *testing.T) {
	ctx1 := requestcache.WithRequestCache(context.Background())
	ex := newTestingExecutor(nil)

	cmd := NewCachableCommand("foo", "", "a")
	_, err := ex.Exec(ctx1, cmd)
	assert.Nil(t, err)

	ctx2 := requestcache.WithRequestCache(context.Background())

	// Caching is context local.
	cmd2 := NewCachableCommand("bar", "", "a")
	r, err := ex.Exec(ctx2, cmd2)
	assert.Nil(t, err)
	assert.Equal(t, "bar", r)
}

func TestExecCachingOnlyInCacheContext(t *testing.T) {
	ctx := context.Background()
	ex := newTestingExecutor(nil)

	cmd := NewCachableCommand("foo", "", "a")
	_, err := ex.Exec(ctx, cmd)
	assert.Nil(t, err)

	cmd2 := NewCachableCommand("bar", "", "a")
	r, err := ex.Exec(ctx, cmd2)
	assert.Nil(t, err)
	assert.Equal(t, "bar", r)
}

func TestExecFailureCacheableCommandIsCached(t *testing.T) {
	ctx := requestcache.WithRequestCache(context.Background())
	ex := newTestingExecutor(nil)

	cmd := NewCachableCommand("error", "none", "a")
	_, err := ex.Exec(ctx, cmd)
	assert.Equal(t, errors.New("foo"), err)

	// The error of previous command execution should be returned.
	cmd2 := NewCachableCommand("bar", "", "a")
	_, err = ex.Exec(ctx, cmd2)
	assert.Equal(t, errors.New("foo"), err)
}

func TestExecRequestCacheDisabled(t *testing.T) {
	ctx := requestcache.WithRequestCache(context.Background())
	cfg := vaquita.NewEmptyMapConfig()
	cfg.SetProperty("cuirass.command.default.requestCache.enabled", "false")
	ex := newTestingExecutor(cfg)

	cmd := NewCachableCommand("error", "none", "a")
	_, err := ex.Exec(ctx, cmd)
	assert.Equal(t, errors.New("foo"), err)

	cmd2 := NewCachableCommand("bar", "", "a")
	r, err := ex.Exec(ctx, cmd2)
	assert.Nil(t, err)
	assert.Equal(t, "bar", r)
}

func TestExecNonCacheableCommandNotCached(t *testing.T) {
	ctx := requestcache.WithRequestCache(context.Background())
	ex := newTestingExecutor(nil)

	cmd := NewFooCommand("foo", "")
	r, err := ex.Exec(ctx, cmd)
	assert.Nil(t, err)
	assert.Equal(t, "foo", r)

	cmd2 := NewFooCommand("bar", "")
	r, err = ex.Exec(ctx, cmd2)
	assert.Nil(t, err)
	assert.Equal(t, "bar", r)
}

func TestExecutorMetricsAreUpdated(t *testing.T) {
	ctx := requestcache.WithRequestCache(context.Background())
	ex := newTestingExecutor(nil)
	m := ex.Metrics().ForCommand("FooCommand")

	cmd := NewFooCommand("foo", "")
	ex.Exec(ctx, cmd)
	assert.Equal(t, 1, m.TotalRequests())
	assert.Equal(t, 0, m.ErrorCount())
	assert.Equal(t, 1, m.RollingSum(requestlog.Success))

	cmd2 := NewFooCommand("panic", "response")
	ex.Exec(ctx, cmd2)
	assert.Equal(t, 2, m.TotalRequests())
	assert.Equal(t, 1, m.ErrorCount())
	assert.Equal(t, 1, m.RollingSum(requestlog.Failure))
	assert.Equal(t, 1, m.RollingSum(requestlog.FallbackSuccess))

	cmd3 := NewFooCommand("error", "panic")
	ex.Exec(ctx, cmd3)
	assert.Equal(t, 3, m.TotalRequests())
	assert.Equal(t, 2, m.ErrorCount())
	assert.Equal(t, 2, m.RollingSum(requestlog.Failure))
	assert.Equal(t, 1, m.RollingSum(requestlog.FallbackFailure))

	m2 := ex.Metrics().ForCommand("Cachable")
	cmd4 := NewCachableCommand("foo", "", "a")
	ex.Exec(ctx, cmd4)
	ex.Exec(ctx, cmd4)

	assert.Equal(t, 1, m2.RollingSum(requestlog.ResponseFromCache))
	assert.True(t, m2.ExecutionTimePercentile(100) > 0)
	assert.True(t, m2.ExecutionTimePercentile(50) > 0)
	assert.True(t, m2.ExecutionTimePercentile(0) > 0)
}

func BenchmarkExecutorCommandExecution(b *testing.B) {
	ctx := context.Background()
	cmd := NewFooCommand("foo", "")
	cfg := vaquita.NewEmptyMapConfig()
	ex := cuirass.NewExecutor(cfg)
	for i := 0; i < b.N; i++ {
		ex.Exec(ctx, cmd)
	}
}
