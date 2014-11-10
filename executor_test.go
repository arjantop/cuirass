package cuirass_test

import (
	"errors"
	"testing"
	"time"

	"code.google.com/p/go.net/context"

	"github.com/arjantop/cuirass"
	"github.com/arjantop/cuirass/circuitbreaker"
	"github.com/arjantop/cuirass/requestcache"
	"github.com/arjantop/cuirass/requestlog"
	"github.com/stretchr/testify/assert"
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

func newTestingExecutor() *cuirass.Executor {
	return cuirass.NewExecutor(100 * time.Millisecond)
}

func TestExecSuccessNoLogging(t *testing.T) {
	ctx := context.Background()
	cmd := NewFooCommand("foo", "")
	ex := newTestingExecutor()
	_, err := ex.Exec(ctx, cmd)
	assert.Nil(t, err)
}

func TestExecSuccess(t *testing.T) {
	ctx := requestlog.WithRequestLog(context.Background())
	cmd := NewFooCommand("foo", "")
	ex := newTestingExecutor()
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
	ex := newTestingExecutor()
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
	ex := newTestingExecutor()
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
	ex := newTestingExecutor()
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
	ex := newTestingExecutor()
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
	ex := newTestingExecutor()
	r, err := ex.Exec(ctx, cmd)
	assert.Nil(t, err)
	assert.Equal(t, "fallback", r)

	request := requestlog.FromContext(ctx).LastRequest()
	assert.Equal(t, "FooCommand", request.CommandName())
	assert.Equal(t,
		[]requestlog.ExecutionEvent{requestlog.Failure, requestlog.FallbackSuccess},
		request.Events())
}

func TestExecPanicWithoutFallback(t *testing.T) {
	ctx := requestlog.WithRequestLog(context.Background())
	cmd := NewFooCommand("panic", "none")
	ex := newTestingExecutor()
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
	ex := newTestingExecutor()
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
	ex := newTestingExecutor()
	for i := 0; i < int(circuitbreaker.DefaultRequestVolumeThreshold); i++ {
		_, err := ex.Exec(ctx, cmd)
		assert.Equal(t, errors.New("foo"), err)
	}
	_, err := ex.Exec(ctx, cmd)
	assert.Equal(t, circuitbreaker.CircuitOpenError, err)

	request := requestlog.FromContext(ctx).LastRequest()
	assert.Equal(t, "FooCommand", request.CommandName())
	assert.Equal(t,
		[]requestlog.ExecutionEvent{requestlog.ShortCircuited},
		request.Events())
}

func TestExecRequestLogging(t *testing.T) {
	ctx := requestlog.WithRequestLog(context.Background())
	cmd := NewFooCommand("foo", "")
	ex := newTestingExecutor()

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

func NewTimeoutCommand() *cuirass.Command {
	return cuirass.NewCommand("TimeoutCommand", func(ctx context.Context) (interface{}, error) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Second):
			return 0, nil
		}
	}).Build()
}

func TestExecTimesOut(t *testing.T) {
	ctx := requestlog.WithRequestLog(context.Background())
	cmd := NewTimeoutCommand()
	ex := cuirass.NewExecutor(time.Millisecond)
	_, err := ex.Exec(ctx, cmd)
	assert.Equal(t, context.DeadlineExceeded, err)

	request := requestlog.FromContext(ctx).LastRequest()
	assert.Equal(t, "TimeoutCommand", request.CommandName())
	assert.Equal(t,
		[]requestlog.ExecutionEvent{requestlog.Timeout},
		request.Events())
}

func NewCachableCommand(s, f, key string) *cuirass.Command {
	return cuirass.NewCommand("FooCommand", func(ctx context.Context) (interface{}, error) {
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
	ctx := requestcache.WithRequestCache(context.Background())
	ex := newTestingExecutor()

	cmd := NewCachableCommand("foo", "", "a")
	r, err := ex.Exec(ctx, cmd)
	assert.Nil(t, err)
	assert.Equal(t, "foo", r)

	// This execution should return the value "foo" from cache instead of returning
	// the value "bar" that woul be returned in the case of evaluation the command.
	cmd2 := NewCachableCommand("bar", "", "a")
	r, err = ex.Exec(ctx, cmd2)
	assert.Nil(t, err)
	assert.Equal(t, "foo", r)

	// This command has a different cache key so it should be evaluated.
	cmd3 := NewCachableCommand("baz", "", "b")
	r, err = ex.Exec(ctx, cmd3)
	assert.Nil(t, err)
	assert.Equal(t, "baz", r)
}

func TestExecFallbackCacheableCommandIsCached(t *testing.T) {
	ctx := requestcache.WithRequestCache(context.Background())
	ex := newTestingExecutor()

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
	ex := newTestingExecutor()

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
	ex := newTestingExecutor()

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
	ex := newTestingExecutor()

	cmd := NewCachableCommand("error", "none", "a")
	_, err := ex.Exec(ctx, cmd)
	assert.Equal(t, errors.New("foo"), err)

	// The error of previous command execution should be returned.
	cmd2 := NewCachableCommand("bar", "", "a")
	_, err = ex.Exec(ctx, cmd2)
	assert.Equal(t, errors.New("foo"), err)
}

func TestExecNonCacheableCommandNotCached(t *testing.T) {
	ctx := requestcache.WithRequestCache(context.Background())
	ex := newTestingExecutor()

	cmd := NewFooCommand("foo", "")
	r, err := ex.Exec(ctx, cmd)
	assert.Nil(t, err)
	assert.Equal(t, "foo", r)

	cmd2 := NewFooCommand("bar", "")
	r, err = ex.Exec(ctx, cmd2)
	assert.Nil(t, err)
	assert.Equal(t, "bar", r)
}
