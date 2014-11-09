package requestlog

import (
	"bytes"
	"strconv"
	"sync"
	"time"
)

type ExecutionEvent byte

const (
	Success ExecutionEvent = iota
	Failure
	Timeout
	ShortCircuited
	ResponseFromCache

	FallbackSuccess
	FallbackFailure
)

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

type ExecutionInfo struct {
	commandName   string
	executionTime time.Duration
	events        []ExecutionEvent
}

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

func (e *ExecutionInfo) CommandName() string {
	return e.commandName
}

func (e *ExecutionInfo) ExecutionTime() time.Duration {
	// If for some reason we have negative execution time correct it to minimum
	// so we can always assume that execution time > 0.
	if e.executionTime < 0 {
		return 0
	}
	return e.executionTime
}

func (e *ExecutionInfo) Events() []ExecutionEvent {
	r := make([]ExecutionEvent, len(e.events))
	// Make a copy of the event array so users of the api canno change the underlying
	// array directly.
	copy(r, e.events)
	return r
}

func (e *ExecutionInfo) canCollapseWith(other *ExecutionInfo) bool {
	if other == nil {
		return false
	}
	// Everything except execution must be equal to allow collapsing when converting to string.
	return e.commandName == other.commandName && eventsEqual(e.events, other.events)
}

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

type RequestLog struct {
	executedRequests     []*ExecutionInfo
	executedRequestsLock *sync.RWMutex
}

func newRequestLog() *RequestLog {
	return &RequestLog{
		executedRequests:     make([]*ExecutionInfo, 0),
		executedRequestsLock: new(sync.RWMutex),
	}
}

func (l *RequestLog) AddRequest(info *ExecutionInfo) {
	l.executedRequestsLock.Lock()
	defer l.executedRequestsLock.Unlock()
	l.executedRequests = append(l.executedRequests, info)
}

func (l *RequestLog) Size() int {
	l.executedRequestsLock.RLock()
	defer l.executedRequestsLock.RUnlock()
	return len(l.executedRequests)
}

func (l *RequestLog) LastRequest() *ExecutionInfo {
	l.executedRequestsLock.RLock()
	defer l.executedRequestsLock.RUnlock()
	return l.executedRequests[len(l.executedRequests)-1]
}

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
