package requestlog

import (
	"bytes"
	"strconv"
	"sync"
)

type RequestLog struct {
	executedCommands     []string
	executedCommandsLock *sync.RWMutex
}

func newRequestLog() *RequestLog {
	return &RequestLog{
		executedCommands:     make([]string, 0),
		executedCommandsLock: new(sync.RWMutex),
	}
}

func (l *RequestLog) AddRequest(commandName string) {
	l.executedCommandsLock.Lock()
	defer l.executedCommandsLock.Unlock()
	l.executedCommands = append(l.executedCommands, commandName)
}

func (l *RequestLog) Size() int {
	l.executedCommandsLock.RLock()
	defer l.executedCommandsLock.RUnlock()
	return len(l.executedCommands)
}

func (l *RequestLog) String() string {
	var b bytes.Buffer
	lastCommand := ""
	commandCount := 0
	first := true
	// Because range iterates over a copy of the slice we need no locks here.
	for _, commandName := range l.executedCommands {
		if lastCommand != commandName {
			if lastCommand != "" {
				if !first {
					b.WriteString(", ")
				}
				writeCommand(&b, lastCommand, commandCount)
				first = false
			}
			lastCommand = commandName
			commandCount = 1
		} else {
			commandCount += 1
		}
	}
	if !first {
		b.WriteString(", ")
	}
	writeCommand(&b, lastCommand, commandCount)
	return b.String()
}

func writeCommand(b *bytes.Buffer, name string, count int) {
	b.WriteString(name)
	if count > 1 {
		b.WriteString("x")
		b.WriteString(strconv.Itoa(count))
	}
}
