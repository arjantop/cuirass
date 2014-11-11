package requestlog

import (
	"bytes"
	"strconv"
	"sync"
	"time"
)

// ExecutionEvent is an event that represents success or failure of different
// stages of command execution.
type ExecutionEvent byte

const (
	// Success event happens when a command was successfully executed.
	Success ExecutionEvent = iota
	// Failure event happens when a command returned and error or panicked
	// when executing.
	Failure
	// Timeout event happens when a command took too long to execute.
	Timeout
	// ShortCircuited event happens when the circuit breaker for the command
	// is closed.
	ShortCircuited
	// ResponseFromCache event happens when the response for the command came
	// from previously executed command cache.
	ResponseFromCache

	// FallbackSuccess happens when the fallback logic of a command executed
	// successfully.
	FallbackSuccess
	// FallbakcFailure event happens when the fallback logic returned and error
	// or panicked while executing.
	FallbackFailure
)

// String returns a string representation of an execution event.
func (e ExecutionEvent) String() (s string) {
	switch e {
	case Success:
		s = "SUCCESS"
	case Failure:
		s = "FAILURE"
	case Timeout:
		s = "TIMEOUT"
	case ShortCircuited:
		s = "SHORT_CIRCUITED"
	case ResponseFromCache:
		s = "RESPONSE_FROM_CACHE"
	case FallbackSuccess:
		s = "FALLBACK_SUCCESS"
	case FallbackFailure:
		s = "FALLBACK_FAILURE"
	}
	return
}

// ExecutionInfo holds the execution duration and events that happened during
// command execution.
type ExecutionInfo struct {
	commandName   string
	executionTime time.Duration
	events        []ExecutionEvent
}

// NewExecutionInfo construct a new ExecutionInfo for command name.
func NewExecutionInfo(
	commandName string,
	executionTime time.Duration,
	events []ExecutionEvent) *ExecutionInfo {

	return &ExecutionInfo{
		commandName:   commandName,
		executionTime: executionTime,
		events:        events,
	}
}

// CommandName returns a command name that this execution info belongs to.
func (e *ExecutionInfo) CommandName() string {
	return e.commandName
}

// ExecutionTime returns the duration that was spent executing the command.
func (e *ExecutionInfo) ExecutionTime() time.Duration {
	// If for some reason we have negative execution time correct it to minimum
	// so we can always assume that execution time > 0.
	if e.executionTime < 0 {
		return 0
	}
	return e.executionTime
}

// Events returns a slice of events that happend in the order that they occurred.
func (e *ExecutionInfo) Events() []ExecutionEvent {
	r := make([]ExecutionEvent, len(e.events))
	// Make a copy of the event array so users of the api canno change the underlying
	// array directly.
	copy(r, e.events)
	return r
}

// canCollapseWith returns true if the current command can be collapsed with
// the other one when displaying the log.
// Commands are collapsable if the command name and events match. Execution time
// does not matter.
func (e *ExecutionInfo) canCollapseWith(other *ExecutionInfo) bool {
	if other == nil {
		return false
	}
	// Everything except execution must be equal to allow collapsing when converting to string.
	return e.commandName == other.commandName && eventsEqual(e.events, other.events)
}

// eventsEqual returns true if both event logs are the same.
func eventsEqual(ev1 []ExecutionEvent, ev2 []ExecutionEvent) bool {
	if len(ev1) != len(ev2) {
		return false
	}
	for i, e1 := range ev1 {
		e2 := ev2[i]
		if e1 != e2 {
			return false
		}
	}
	return true
}

// RequestLog keeps a history of request execution information in the order that requests
// occurred.
// It is safe to access RequestLog from multiple threads simultaneously.
type RequestLog struct {
	executedRequests     []*ExecutionInfo
	executedRequestsLock *sync.RWMutex
}

// newRequestLog constructs a new empty request log.
func newRequestLog() *RequestLog {
	return &RequestLog{
		executedRequests:     make([]*ExecutionInfo, 0),
		executedRequestsLock: new(sync.RWMutex),
	}
}

// AddExecutionInfo add the execution of executed command to the request log.
func (l *RequestLog) AddExecutionInfo(info *ExecutionInfo) {
	l.executedRequestsLock.Lock()
	defer l.executedRequestsLock.Unlock()
	l.executedRequests = append(l.executedRequests, info)
}

// Size returns the number of request logs currently in the logger.
func (l *RequestLog) Size() int {
	l.executedRequestsLock.RLock()
	defer l.executedRequestsLock.RUnlock()
	return len(l.executedRequests)
}

// LastRequest returns the execution info of the latest request added to the log.
func (l *RequestLog) LastRequest() *ExecutionInfo {
	l.executedRequestsLock.RLock()
	defer l.executedRequestsLock.RUnlock()
	return l.executedRequests[len(l.executedRequests)-1]
}

// String returns a nice string representation of the commands in the log.
// Used for command execution logging.
func (l *RequestLog) String() string {
	var b bytes.Buffer
	var lastInfo *ExecutionInfo
	commandCount := 0
	first := true
	// Because range iterates over a copy of the slice we need no locks here.
	for _, info := range l.executedRequests {
		if !info.canCollapseWith(lastInfo) {
			if lastInfo != nil {
				if !first {
					b.WriteString(", ")
				}
				writeCommand(&b, lastInfo, commandCount)
				first = false
			}
			lastInfo = info
			commandCount = 1
		} else {
			commandCount += 1
			lastInfo.executionTime += info.executionTime
		}
	}
	if !first {
		b.WriteString(", ")
	}
	writeCommand(&b, lastInfo, commandCount)
	return b.String()
}

// writeCommand writes a command execution info to the string buffer.
func writeCommand(b *bytes.Buffer, info *ExecutionInfo, count int) {
	b.WriteString(info.CommandName())
	b.WriteString("[")
	for i, event := range info.Events() {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(event.String())
	}
	b.WriteString("]")
	b.WriteString("[")
	timeInMilliseconds := info.ExecutionTime().Nanoseconds() / int64(time.Millisecond)
	b.WriteString(strconv.FormatInt(timeInMilliseconds, 10))
	b.WriteString("ms]")
	if count > 1 {
		b.WriteString("x")
		b.WriteString(strconv.Itoa(count))
	}
}