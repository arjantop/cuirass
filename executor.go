package cuirass

import (
	"errors"
	"sync"

	"github.com/arjantop/cuirass/circuitbreaker"
)

var UnknownPanic = errors.New("Unknown panic")

type Executor struct {
	circuitBreakers cbMap
}

func NewExecutor() *Executor {
	return &Executor{
		circuitBreakers: newCbMap(),
	}
}

func (e *Executor) Exec(cmd Command, result interface{}) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = cmd.Fallback(result)
			if err != nil {
				switch x := r.(type) {
				case error:
					err = x
				case string:
					err = errors.New(x)
				default:
					err = UnknownPanic
				}
			}
		}
	}()
	cb := e.getCircuitBreakerForCommand(cmd)
	err = cb.Do(func() error {
		return cmd.Run(result)
	})
	if err != nil {
		panic(err)
	}
	return nil
}

func (e *Executor) getCircuitBreakerForCommand(cmd Command) *circuitbreaker.CircuitBreaker {
	if cb, ok := e.circuitBreakers.get(cmd.Name()); ok {
		return cb
	} else {
		cb := circuitbreaker.New(
			circuitbreaker.DefaultErrorThreshold,
			circuitbreaker.DefaultSleepWindow,
			circuitbreaker.DefaultRequestVolumeThreshold)
		e.circuitBreakers.set(cmd.Name(), cb)
		return cb
	}
}

type cbMap struct {
	values map[string]*circuitbreaker.CircuitBreaker
	lock   *sync.RWMutex
}

func newCbMap() cbMap {
	return cbMap{
		values: make(map[string]*circuitbreaker.CircuitBreaker),
		lock:   new(sync.RWMutex),
	}
}

func (m *cbMap) get(name string) (*circuitbreaker.CircuitBreaker, bool) {
	m.lock.RLock()
	defer m.lock.RUnlock()
	cb, ok := m.values[name]
	return cb, ok
}

func (m *cbMap) set(name string, cb *circuitbreaker.CircuitBreaker) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.values[name] = cb
}
