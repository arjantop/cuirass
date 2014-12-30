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
	// SemaphoreRejected event happens if there are too many concurrent requests
	// for the executed commands.
	SemaphoreRejected
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
	case SemaphoreRejected:
		s = "SEMAPHORE_REJECTED"
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
	events []ExecutionEvent) ExecutionInfo {

	return ExecutionInfo{
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

// RequestLog keeps a history of request execution information in the order that requests
// occurred.
// It is safe to access RequestLog from multiple threads simultaneously.
type RequestLog struct {
	executedRequests     []ExecutionInfo
	executedRequestsLock *sync.RWMutex
}

// newRequestLog constructs a new empty request log.
func newRequestLog() *RequestLog {
	return &RequestLog{
		executedRequests:     make([]ExecutionInfo, 0),
		executedRequestsLock: new(sync.RWMutex),
	}
}

// AddExecutionInfo add the execution of executed command to the request log.
func (l *RequestLog) AddExecutionInfo(info ExecutionInfo) {
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
	return &l.executedRequests[len(l.executedRequests)-1]
}

// String returns a nice string representation of the commands in the log.
// Used for command execution logging.
func (l *RequestLog) String() string {
	var b bytes.Buffer
	commandOrder := make([]string, 0)
	aggregatedCommands := make(map[string]*aggregatedCommand)

	l.executedRequestsLock.RLock()
	defer l.executedRequestsLock.RUnlock()
	for _, info := range l.executedRequests {
		b.Truncate(0)
		writeCommand(&b, info)
		cmd := b.String()
		if ac, ok := aggregatedCommands[cmd]; ok {
			ac.count += 1
			ac.executionTime += info.ExecutionTime()
		} else {
			commandOrder = append(commandOrder, cmd)
			aggregatedCommands[cmd] = newAggregatedCommand(info.ExecutionTime())
		}
	}

	b.Truncate(0)
	for i, cmd := range commandOrder {
		if i > 0 {
			b.WriteString(", ")
		}
		writeCommandWithTime(&b, cmd, aggregatedCommands[cmd])
	}

	return b.String()
}

type aggregatedCommand struct {
	count         int
	executionTime time.Duration
}

func newAggregatedCommand(executionTime time.Duration) *aggregatedCommand {
	return &aggregatedCommand{
		count:         1,
		executionTime: executionTime,
	}
}

// writeCommand writes a command execution info to the string buffer.
func writeCommand(b *bytes.Buffer, info ExecutionInfo) {
	b.WriteString(info.CommandName())
	b.WriteString("[")
	for i, event := range info.Events() {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(event.String())
	}
	b.WriteString("]")
}

// writeCommandWithExecution writes a command string and accumulated execution
// time with execution multiplier.
func writeCommandWithTime(b *bytes.Buffer, cmd string, ac *aggregatedCommand) {
	b.WriteString(cmd)
	b.WriteString("[")
	timeInMilliseconds := ac.executionTime.Nanoseconds() / int64(time.Millisecond)
	b.WriteString(strconv.FormatInt(timeInMilliseconds, 10))
	b.WriteString("ms]")
	if ac.count > 1 {
		b.WriteString("x")
		b.WriteString(strconv.Itoa(ac.count))
	}
}
