package util_test

import (
	"testing"

	"github.com/arjantop/cuirass/util"
	"github.com/stretchr/testify/assert"
)

func newTestingSemaphore(capacity int) *util.Semaphore {
	return util.NewSemaphore(capacity)
}

func TestSemaphoreAcquireAndRelease(t *testing.T) {
	s := newTestingSemaphore(10)
	assert.True(t, s.TryAcquire())
	assert.True(t, s.TryAcquire())
	s.Release()
	s.Release()
}

func TestSemaphoreAcquireOverCapacity(t *testing.T) {
	s := newTestingSemaphore(1)
	assert.True(t, s.TryAcquire())
	assert.False(t, s.TryAcquire())
	s.Release()
	assert.True(t, s.TryAcquire())
}

func TestSemaphoreReleaseEmpty(t *testing.T) {
	s := newTestingSemaphore(1)
	assert.Panics(t, func() {
		s.Release()
	})
}
