package cuirass_test

import (
	"errors"
	"testing"
	"time"

	"code.google.com/p/go.net/context"

	"github.com/arjantop/cuirass"
	"github.com/arjantop/cuirass/circuitbreaker"
	"github.com/stretchr/testify/assert"
)

type FooCommand struct {
	s, f string
}

func NewFooCommand(s, f string) *FooCommand {
	return &FooCommand{s, f}
}

func (c *FooCommand) Name() string {
	return "FooCommand"
}

func (c *FooCommand) Run(ctx context.Context, result interface{}) error {
	if c.s == "error" {
		return errors.New("foo")
	} else if c.s == "panic" {
		panic("foopanic")
	} else if c.s == "panicint" {
		panic(1)
	}
	*result.(*string) = c.s
	return nil
}

func (c *FooCommand) Fallback(ctx context.Context, result interface{}) error {
	if c.f == "none" {
		return cuirass.FallbackNotImplemented
	} else if c.f == "error" {
		return errors.New("fallbackerr")
	} else if c.f == "panic" {
		panic("fallpanic")
	}
	*result.(*string) = c.f
	return nil
}

func newTestingExecutor() *cuirass.Executor {
	return cuirass.NewExecutor(100 * time.Millisecond)
}

func TestExecSuccess(t *testing.T) {
	ctx := context.Background()
	cmd := NewFooCommand("foo", "")
	ex := newTestingExecutor()
	var r string
	assert.Nil(t, ex.Exec(ctx, cmd, &r))
	assert.Equal(t, r, "foo")
}

func TestExecErrorWithFallback(t *testing.T) {
	ctx := context.Background()
	cmd := NewFooCommand("error", "fallback")
	ex := newTestingExecutor()
	var r string
	assert.Nil(t, ex.Exec(ctx, cmd, &r))
	assert.Equal(t, r, "fallback")
}

func TestExecErrorWithFallbackPanic(t *testing.T) {
	ctx := context.Background()
	cmd := NewFooCommand("error", "panic")
	ex := newTestingExecutor()
	var r string
	assert.Equal(t, errors.New("fallpanic"), ex.Exec(ctx, cmd, &r))
}

func TestExecErrorWithoutFallback(t *testing.T) {
	ctx := context.Background()
	cmd := NewFooCommand("error", "none")
	ex := newTestingExecutor()
	var r string
	assert.Equal(t, ex.Exec(ctx, cmd, &r), errors.New("foo"))
}

func TestExecErrorWithoutFallbackFailure(t *testing.T) {
	ctx := context.Background()
	cmd := NewFooCommand("error", "error")
	ex := newTestingExecutor()
	var r string
	// The original error from Run is returned if Fallback fails too.
	assert.Equal(t, ex.Exec(ctx, cmd, &r), errors.New("foo"))
}

func TestExecPanicWithFallback(t *testing.T) {
	ctx := context.Background()
	cmd := NewFooCommand("panic", "fallback")
	ex := newTestingExecutor()
	var r string
	assert.Nil(t, ex.Exec(ctx, cmd, &r))
	assert.Equal(t, r, "fallback")
}

func TestExecPanicWithoutFallback(t *testing.T) {
	ctx := context.Background()
	cmd := NewFooCommand("panic", "none")
	ex := newTestingExecutor()
	var r string
	assert.Equal(t, ex.Exec(ctx, cmd, &r), errors.New("foopanic"))
}

func TestExecIntPanicWithoutFallback(t *testing.T) {
	ctx := context.Background()
	cmd := NewFooCommand("panicint", "none")
	ex := newTestingExecutor()
	var r string
	assert.Equal(t, ex.Exec(ctx, cmd, &r), cuirass.UnknownPanic)
}

func TestExecFailuresTripCircuitBreaker(t *testing.T) {
	ctx := context.Background()
	cmd := NewFooCommand("error", "none")
	ex := newTestingExecutor()
	var r string
	for i := 0; i < int(circuitbreaker.DefaultRequestVolumeThreshold); i++ {
		assert.Equal(t, ex.Exec(ctx, cmd, &r), errors.New("foo"))
	}
	assert.Equal(t, ex.Exec(ctx, cmd, &r), circuitbreaker.CircuitOpenError)
}

type TimeoutCommand struct{}

func NewTimeoutCommand() *TimeoutCommand {
	return &TimeoutCommand{}
}

func (c *TimeoutCommand) Name() string {
	return "TimeoutCommand"
}

func (c *TimeoutCommand) Run(ctx context.Context, result interface{}) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(time.Second):
		return nil
	}
}

func (c *TimeoutCommand) Fallback(ctx context.Context, result interface{}) error {
	return cuirass.FallbackNotImplemented
}

func TestExecTimesOut(t *testing.T) {
	ctx := context.Background()
	cmd := NewTimeoutCommand()
	ex := cuirass.NewExecutor(time.Millisecond)
	var r string
	assert.Equal(t, ex.Exec(ctx, cmd, &r), context.DeadlineExceeded)
}
