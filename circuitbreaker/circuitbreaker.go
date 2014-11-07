package circuitbreaker

import (
	"errors"
	"sync/atomic"
	"time"

	"github.com/arjantop/cuirass/num"
)

var (
	// Default threshold in percents (%). If more than 50% of requests are failures
	// we trip the breaker.
	DefaultErrorThreshold float64 = 50.0
	// Duration that we will sleep after tripping the breaker before attempting to
	// reset the circuit.
	DefaultSleepWindow time.Duration = 5000 * time.Millisecond
	// Volume of requests that have to be made before request error percentage matters.
	// We don't want to trip the breaker a couple of first requests fail for some reason.
	DefaultRequestVolumeThreshold uint64 = 20

	// Error indicating that the circuit is open and the request was not executed
	// or an attempt to reset the circuit failed.
	CircuitOpenError = errors.New("Circuit open")
)

// Integer constants to be used as true and false constants with circuit breaker.
const (
	intTrue  = 1
	intFalse = 0
)

// CircuitBreaker is an implementation of circuit breaker pattern.
// http://martinfowler.com/bliki/CircuitBreaker.html
type CircuitBreaker struct {
	errorThreshold         float64
	sleepWindow            time.Duration
	requestVolumeThreshold uint64

	// uint32 is used instead of bool so we can use atomic operations.
	circuitOpen   uint32
	lastTrialTime int64

	errorCounter   *num.RollingNumber
	requestCounter *num.RollingNumber
}

// Constructs a new circuit breaker. The circuit is closed by default and allowed
// the initial statistical values are zero ().
func New(errorThreshold float64, sleepWindow time.Duration, requestVolumeThreshold uint64) *CircuitBreaker {
	return &CircuitBreaker{
		errorThreshold:         errorThreshold,
		sleepWindow:            sleepWindow,
		requestVolumeThreshold: requestVolumeThreshold,
		circuitOpen:            intFalse,
		lastTrialTime:          0,
		errorCounter:           num.NewRollingNumber(num.DefaultWindowSize, num.DefaultWindowBuckets),
		requestCounter:         num.NewRollingNumber(num.DefaultWindowSize, num.DefaultWindowBuckets),
	}
}

// Do executes a function in the context of this circuit breaker.
// If the circuit is open the function is not executed and an error CircuitOpenError
// is returned.
func (cb *CircuitBreaker) Do(f func() error) error {
	// Update the circuit state after every request.
	defer cb.updateState()
	cb.requestCounter.Increment()
	if allowed, trial := cb.isRequestAllowed(); allowed {
		err := f()
		if err != nil {
			if trial {
				// If the request was a trial then the real error does not matter
				// to the caller.
				err = CircuitOpenError
			}
			// If error occurs we increment the request error counter.
			cb.errorCounter.Increment()
		} else if trial {
			// If the request was a trial and it succeeded reset all the counters
			// and reset the breaker state to closed.
			cb.errorCounter.Reset()
			cb.requestCounter.Reset()
			atomic.StoreUint32(&cb.circuitOpen, intFalse)
		}
		return err
	} else {
		// Circuit is open so we fail fast and return the error.
		cb.errorCounter.Increment()
		return CircuitOpenError
	}
}

// updateState trips the breaker if the request error rate is larger than the threshold.
func (cb *CircuitBreaker) updateState() {
	if cb.IsOpen() || cb.requestCounter.Sum() < cb.requestVolumeThreshold {
		// If the circuit is open pr there were not enough requests made in the
		// configured statistical window there is nothing to do.
		return
	}
	if float64(cb.errorCounter.Sum())*100.0/float64(cb.requestCounter.Sum()) > cb.errorThreshold {
		// If the error request rate is greater that configured threshold attempt
		// to change circuit to Open.
		if atomic.CompareAndSwapUint32(&cb.circuitOpen, intFalse, intTrue) {
			// If we set the circuit state successfully update the request
			// trial time to a current time.
			atomic.StoreInt64(&cb.lastTrialTime, time.Now().UnixNano())
		}
	}
}

// isRequestAllowed returns true as first return value when a request is allowed
// to be made. Second return value indicated if the allowed request is a trial
// in half-open state with attempt to close the circuit on trial request success.
func (cb *CircuitBreaker) isRequestAllowed() (bool, bool) {
	trialCall := cb.isTrialCallAllowed()
	return !cb.IsOpen() || trialCall, trialCall
}

// istrialcallallowed returns true is the trial call is allowed to close the circuit.
// Time of last request trial is updated to the current time.
func (cb *CircuitBreaker) isTrialCallAllowed() bool {
	lastTrialTime := atomic.LoadInt64(&cb.lastTrialTime)
	timestamp := time.Now().UnixNano()
	if cb.IsOpen() && timestamp > lastTrialTime+int64(cb.sleepWindow*time.Nanosecond) {
		if atomic.CompareAndSwapInt64(&cb.lastTrialTime, lastTrialTime, timestamp) {
			return true
		}
	}
	return false
}

// IsOpen returns true if the state of circuit breaker is open or half-open
func (cb *CircuitBreaker) IsOpen() bool {
	return atomic.LoadUint32(&cb.circuitOpen) == intTrue
}
