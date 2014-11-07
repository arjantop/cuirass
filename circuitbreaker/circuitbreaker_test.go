package circuitbreaker_test

import (
	"errors"
	"testing"
	"time"

	"github.com/arjantop/cuirass/circuitbreaker"
	"github.com/stretchr/testify/assert"
)

var testErr = errors.New("test")

func newTestingCircuitBreaker() *circuitbreaker.CircuitBreaker {
	return circuitbreaker.New(50.0, time.Millisecond, 3)
}

func TestCircuitBreakerDoClosed(t *testing.T) {
	cb := newTestingCircuitBreaker()
	assert.False(t, cb.IsOpen())
	called := false
	cb.Do(func() error {
		called = true
		return nil
	})
	assert.True(t, called)
}

func TestCircuitBreakerOpenRequestVolume(t *testing.T) {
	cb := newTestingCircuitBreaker()
	assert.False(t, cb.IsOpen())
	cb.Do(func() error { return testErr })
	cb.Do(func() error { return testErr })
	assert.False(t, cb.IsOpen())
	assert.Equal(t, testErr, cb.Do(func() error { return testErr }))
	assert.True(t, cb.IsOpen())
}

func TestCircuitBreakerOpenAfterErrorThreshold(t *testing.T) {
	cb := newTestingCircuitBreaker()
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
	cb := newTestingCircuitBreaker()
	cb.Do(func() error { return testErr })
	cb.Do(func() error { return testErr })
	cb.Do(func() error { return testErr })
	assert.True(t, cb.IsOpen())
	time.Sleep(time.Millisecond)
	// Curcuit is still closed but next request will be executed.
	assert.True(t, cb.IsOpen())
	called := false
	assert.Nil(t, cb.Do(func() error {
		called = true
		return nil
	}))
	assert.True(t, called)
	assert.False(t, cb.IsOpen())
}

func TestCircuitBreakerStaysOpenOnTrialFailure(t *testing.T) {
	cb := newTestingCircuitBreaker()
	cb.Do(func() error { return testErr })
	cb.Do(func() error { return testErr })
	cb.Do(func() error { return testErr })
	assert.True(t, cb.IsOpen())
	time.Sleep(time.Millisecond)
	called := false
	assert.Equal(t, circuitbreaker.CircuitOpenError, cb.Do(func() error {
		called = true
		return testErr
	}))
	assert.True(t, called)
	assert.True(t, cb.IsOpen())
	assert.Equal(t, circuitbreaker.CircuitOpenError, cb.Do(func() error { panic("unreachable") }))
}
