package cuirass

import (
	"sync"

	"github.com/arjantop/cuirass/util"
)

type semaphore struct {
	sem      *util.Semaphore
	capacity int
}

type SemaphoreFactory struct {
	semaphores map[string]*semaphore
	lock       *sync.Mutex
}

func NewSemaphoreFactory() *SemaphoreFactory {
	return &SemaphoreFactory{
		semaphores: make(map[string]*semaphore),
		lock:       new(sync.Mutex),
	}
}

func (f *SemaphoreFactory) Get(key string, maxConcurrentRequests int) *util.Semaphore {
	f.lock.Lock()
	defer f.lock.Unlock()
	// If the capacity of the semaphore changed create the new one.
	if s, ok := f.semaphores[key]; ok && maxConcurrentRequests == s.capacity {
		return s.sem
	}
	s := util.NewSemaphore(maxConcurrentRequests)
	f.semaphores[key] = &semaphore{s, maxConcurrentRequests}
	return s
}
