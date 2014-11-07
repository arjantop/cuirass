package circuitbreaker

import (
	"errors"
	"sync/atomic"
	"time"

	"github.com/arjantop/cuirass/num"
)

var (
	DefaultErrorThreshold         float64       = 50.0
	DefaultSleepWindow            time.Duration = 5000 * time.Millisecond
	DefaultRequestVolumeThreshold uint64        = 20

	CircuitOpenError = errors.New("Circuit open")
)

const (
	intTrue  = 1
	intFalse = 0
)

type CircuitBreaker struct {
	errorThreshold         float64
	sleepWindow            time.Duration
	requestVolumeThreshold uint64
	circuitOpen            uint32
	lastTrialTime          int64
	errorCounter           *num.RollingNumber
	requestCounter         *num.RollingNumber
}

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

func (cb *CircuitBreaker) Do(f func() error) error {
	defer cb.updateState()
	cb.requestCounter.Increment()
	if allowed, trial := cb.isRequestAllowed(); allowed {
		err := f()
		if err != nil {
			if trial {
				err = CircuitOpenError
			}
			cb.errorCounter.Increment()
		} else if trial {
			cb.errorCounter.Reset()
			cb.requestCounter.Reset()
			atomic.StoreUint32(&cb.circuitOpen, intFalse)
		}
		return err
	} else {
		cb.errorCounter.Increment()
		return CircuitOpenError
	}
}

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

func (cb *CircuitBreaker) IsOpen() bool {
	return atomic.LoadUint32(&cb.circuitOpen) == intTrue
}
