package circuitbreaker

import (
	"errors"
	"sync/atomic"

	"github.com/arjantop/cuirass/num"
	"github.com/arjantop/cuirass/util"
	"github.com/arjantop/vaquita"
)

var (
	// Error indicating that the circuit is open and the request was not executed
	// or an attempt to reset the circuit failed.
	CircuitOpenError = errors.New("circuit open")
)

// Integer constants to be used as true and false constants with circuit breaker.
const (
	intTrue  = 1
	intFalse = 0
)

type CircuitBreakerProperties struct {
	Enabled                  vaquita.BoolProperty
	RequestVolumeThreshold   vaquita.IntProperty
	SleepWindow              vaquita.IntProperty
	ErrorThresholdPercentage vaquita.IntProperty
	ForceOpen                vaquita.BoolProperty
	ForceClosed              vaquita.BoolProperty
}

// CircuitBreaker is an implementation of circuit breaker pattern.
// http://martinfowler.com/bliki/CircuitBreaker.html
type CircuitBreaker struct {
	props *CircuitBreakerProperties

	// uint32 is used instead of bool so we can use atomic operations.
	circuitOpen   uint32
	lastTrialTime int64

	errorCounter   *num.RollingNumber
	requestCounter *num.RollingNumber

	clock util.Clock
}

// Constructs a new circuit breaker. The circuit is closed by default and allowed
// the initial statistical values are zero ().
func New(props *CircuitBreakerProperties, clock util.Clock) *CircuitBreaker {
	return &CircuitBreaker{
		props:          props,
		circuitOpen:    intFalse,
		lastTrialTime:  0,
		errorCounter:   num.NewRollingNumber(num.DefaultWindowSize, num.DefaultWindowBuckets, clock),
		requestCounter: num.NewRollingNumber(num.DefaultWindowSize, num.DefaultWindowBuckets, clock),
		clock:          clock,
	}
}

// Do executes a function in the context of this circuit breaker.
// If the circuit is open the function is not executed and an error CircuitOpenError
// is returned.
func (cb *CircuitBreaker) Do(f func() error) error {
	if !cb.props.Enabled.Get() {
		return f()
	} else if cb.props.ForceOpen.Get() {
		return CircuitOpenError
	}
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
	if cb.IsOpen() || cb.requestCounter.Sum() < uint64(cb.props.RequestVolumeThreshold.Get()) {
		// If the circuit is open pr there were not enough requests made in the
		// configured statistical window there is nothing to do.
		return
	}
	if float64(cb.errorCounter.Sum())*100.0/float64(cb.requestCounter.Sum()) > float64(cb.props.ErrorThresholdPercentage.Get()) {
		// If the error request rate is greater that configured threshold attempt
		// to change circuit to Open.
		if atomic.CompareAndSwapUint32(&cb.circuitOpen, intFalse, intTrue) {
			// If we set the circuit state successfully update the request
			// trial time to a current time.
			atomic.StoreInt64(&cb.lastTrialTime, cb.clock.Now().UnixNano())
		}
	}
}

// isRequestAllowed returns true as first return value when a request is allowed
// to be made. Second return value indicated if the allowed request is a trial
// in half-open state with attempt to close the circuit on trial request success.
func (cb *CircuitBreaker) isRequestAllowed() (bool, bool) {
	trialCall := cb.isTrialCallAllowed()
	if cb.props.ForceClosed.Get() {
		return true, trialCall
	}
	return !cb.IsOpen() || trialCall, trialCall
}

// istrialcallallowed returns true is the trial call is allowed to close the circuit.
// Time of last request trial is updated to the current time.
func (cb *CircuitBreaker) isTrialCallAllowed() bool {
	lastTrialTime := atomic.LoadInt64(&cb.lastTrialTime)
	timestamp := cb.clock.Now().UnixNano()
	if cb.IsOpen() && timestamp > lastTrialTime+int64(cb.props.SleepWindow.Get()*1000000) {
		if atomic.CompareAndSwapInt64(&cb.lastTrialTime, lastTrialTime, timestamp) {
			return true
		}
	}
	return false
}

// IsOpen returns true if the state of circuit breaker is open or half-open
func (cb *CircuitBreaker) IsOpen() bool {
	if cb.props.ForceClosed.Get() {
		return false
	}
	return cb.props.ForceOpen.Get() || atomic.LoadUint32(&cb.circuitOpen) == intTrue
}
