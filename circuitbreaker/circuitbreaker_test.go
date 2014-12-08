package circuitbreaker_test

import (
	"errors"
	"testing"
	"time"

	"github.com/arjantop/cuirass/circuitbreaker"
	"github.com/arjantop/cuirass/util"
	"github.com/arjantop/vaquita"
	"github.com/stretchr/testify/assert"
)

var testErr = errors.New("test")

func newTestingCircuitBreaker(cfg vaquita.DynamicConfig, clock util.Clock) *circuitbreaker.CircuitBreaker {
	if cfg == nil {
		cfg = vaquita.NewEmptyMapConfig()
	}
	if clock == nil {
		clock = util.NewClock()
	}
	f := vaquita.NewPropertyFactory(cfg)
	return circuitbreaker.New(&circuitbreaker.CircuitBreakerProperties{
		f.GetBoolProperty("enabled", true),
		f.GetIntProperty("requestThreshold", 3),
		f.GetIntProperty("sleepWindow", 500),
		f.GetIntProperty("errorThreshold", 50),
		f.GetBoolProperty("forceOpen", false),
		f.GetBoolProperty("forceClosed", false),
	}, clock)
}

func TestCircuitBreakerDoClosed(t *testing.T) {
	cb := newTestingCircuitBreaker(nil, nil)
	assert.False(t, cb.IsOpen())
	called := false
	cb.Do(func() error {
		called = true
		return nil
	})
	assert.True(t, called)
}

func TestCircuitBreakerOpenRequestVolume(t *testing.T) {
	cb := newTestingCircuitBreaker(nil, nil)
	assert.False(t, cb.IsOpen())
	cb.Do(func() error { return testErr })
	cb.Do(func() error { return testErr })
	assert.False(t, cb.IsOpen())
	assert.Equal(t, testErr, cb.Do(func() error { return testErr }))
	assert.True(t, cb.IsOpen())
}

func TestCircuitBreakerOpenAfterErrorThreshold(t *testing.T) {
	cb := newTestingCircuitBreaker(nil, nil)
	assert.False(t, cb.IsOpen())
	cb.Do(func() error { return nil })
	cb.Do(func() error { return testErr })
	cb.Do(func() error { return nil })
	assert.False(t, cb.IsOpen())
	cb.Do(func() error { return testErr })
	assert.False(t, cb.IsOpen())
	cb.Do(func() error { return testErr })
	assert.True(t, cb.IsOpen())
	assert.Equal(t, circuitbreaker.CircuitOpenError, cb.Do(func() error { return testErr }))
}

func TestCircuitBreakerClosesOnTrialSuccess(t *testing.T) {
	cfg := vaquita.NewEmptyMapConfig()
	cfg.SetProperty("requestThreshold", "1")
	clock := util.NewTestableClock(time.Now())
	cb := newTestingCircuitBreaker(cfg, clock)

	// Trip the breaker.
	cb.Do(func() error { return testErr })
	assert.True(t, cb.IsOpen())
	clock.Add(499 * time.Millisecond)
	assert.True(t, cb.IsOpen(), "Still open untill sleep window time passes")
	called := false
	assert.Equal(t, circuitbreaker.CircuitOpenError, cb.Do(func() error {
		called = true
		return nil
	}))

	clock.Add(2 * time.Millisecond)
	// Curcuit is still closed but next request will be executed.
	assert.True(t, cb.IsOpen())
	called = false
	assert.Nil(t, cb.Do(func() error {
		called = true
		return nil
	}))
	assert.True(t, called)
	assert.False(t, cb.IsOpen())
}

func TestCircuitBreakerStaysOpenOnTrialFailure(t *testing.T) {
	cfg := vaquita.NewEmptyMapConfig()
	cfg.SetProperty("requestThreshold", "1")
	clock := util.NewTestableClock(time.Now())
	cb := newTestingCircuitBreaker(cfg, clock)

	// Trip the breaker.
	cb.Do(func() error { return testErr })
	assert.True(t, cb.IsOpen())
	clock.Add(501 * time.Millisecond)
	called := false
	assert.Equal(t, circuitbreaker.CircuitOpenError, cb.Do(func() error {
		called = true
		return testErr
	}))
	assert.True(t, called)
	assert.True(t, cb.IsOpen())
	assert.Equal(t, circuitbreaker.CircuitOpenError, cb.Do(func() error { panic("unreachable") }))
}

func TestCircuitBreakerPropertyDisabled(t *testing.T) {
	cfg := vaquita.NewEmptyMapConfig()
	cfg.SetProperty("enabled", "false")
	cb := newTestingCircuitBreaker(cfg, nil)
	cb.Do(func() error { return testErr })
	cb.Do(func() error { return testErr })
	cb.Do(func() error { return testErr })
	assert.False(t, cb.IsOpen(), "If disabled circuit breaker is never opened")
}

func TestCircuitBreakerPropertyForceClosed(t *testing.T) {
	cfg := vaquita.NewEmptyMapConfig()
	cfg.SetProperty("forceClosed", "true")
	cb := newTestingCircuitBreaker(cfg, nil)
	cb.Do(func() error { return testErr })
	cb.Do(func() error { return testErr })
	cb.Do(func() error { return testErr })
	var called bool
	assert.Nil(t, cb.Do(func() error {
		called = true
		return nil
	}))
	assert.True(t, called)
	assert.False(t, cb.IsOpen(), "If force closed all requests are always allowed")
}

func TestCircuitBreakerPropertyForceOpen(t *testing.T) {
	cfg := vaquita.NewEmptyMapConfig()
	cfg.SetProperty("forceOpen", "true")
	cb := newTestingCircuitBreaker(cfg, nil)
	var called bool
	assert.Equal(t, circuitbreaker.CircuitOpenError, cb.Do(func() error {
		called = true
		return nil
	}))
	assert.False(t, called, "No requests should be executed")
}
