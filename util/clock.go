package util

import "time"

type Clock interface {
	Now() time.Time
}

type realClock struct{}

func NewClock() Clock {
	return new(realClock)
}

func (c *realClock) Now() time.Time {
	return time.Now()
}

type TestableClock struct {
	now time.Time
}

func NewTestableClock(now time.Time) *TestableClock {
	return &TestableClock{
		now: now,
	}
}

func (c *TestableClock) Now() time.Time {
	return c.now
}

func (c *TestableClock) SetTime(t time.Time) {
	c.now = t
}

func (c *TestableClock) Add(d time.Duration) {
	c.now = c.now.Add(d)
}
