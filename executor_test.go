package cuirass_test

import (
	"errors"
	"testing"

	"github.com/arjantop/cuirass"
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

func (c *FooCommand) Run(result interface{}) error {
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

func (c *FooCommand) Fallback(result interface{}) error {
	if c.f == "none" {
		return cuirass.FallbackNotImplemented
	} else if c.f == "error" {
		return errors.New("fallbackerr")
	}
	*result.(*string) = c.f
	return nil
}

func TestExecSuccess(t *testing.T) {
	cmd := NewFooCommand("foo", "")
	ex := cuirass.NewExecutor()
	var r string
	assert.Nil(t, ex.Exec(cmd, &r))
	assert.Equal(t, r, "foo")
}

func TestExecErrorWithFallback(t *testing.T) {
	cmd := NewFooCommand("error", "fallback")
	ex := cuirass.NewExecutor()
	var r string
	assert.Nil(t, ex.Exec(cmd, &r))
	assert.Equal(t, r, "fallback")
}

func TestExecErrorWithoutFallback(t *testing.T) {
	cmd := NewFooCommand("error", "none")
	ex := cuirass.NewExecutor()
	var r string
	assert.Equal(t, ex.Exec(cmd, &r), errors.New("foo"))
}

func TestExecErrorWithoutFallbackFailure(t *testing.T) {
	cmd := NewFooCommand("error", "error")
	ex := cuirass.NewExecutor()
	var r string
	// The original error from Run is returned if Fallback fails too.
	assert.Equal(t, ex.Exec(cmd, &r), errors.New("foo"))
}

func TestExecPanicWithFallback(t *testing.T) {
	cmd := NewFooCommand("panic", "fallback")
	ex := cuirass.NewExecutor()
	var r string
	assert.Nil(t, ex.Exec(cmd, &r))
	assert.Equal(t, r, "fallback")
}

func TestExecPanicWithoutFallback(t *testing.T) {
	cmd := NewFooCommand("panic", "none")
	ex := cuirass.NewExecutor()
	var r string
	assert.Equal(t, ex.Exec(cmd, &r), errors.New("foopanic"))
}

func TestExecIntPanicWithoutFallback(t *testing.T) {
	cmd := NewFooCommand("panicint", "none")
	ex := cuirass.NewExecutor()
	var r string
	assert.Equal(t, ex.Exec(cmd, &r), cuirass.UnknownPanic)
}
