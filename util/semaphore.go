package util

type Semaphore struct {
	c chan struct{}
}

func NewSemaphore(capacity int) *Semaphore {
	return &Semaphore{
		c: make(chan struct{}, capacity),
	}
}

func (s *Semaphore) TryAcquire() bool {
	select {
	case s.c <- struct{}{}:
		return true
	default:
		return false
	}
}

func (s *Semaphore) Release() {
	select {
	case <-s.c:
		return
	default:
		panic("releasing on empty semaphore")
	}
}

func (s *Semaphore) Capacity() int {
	return cap(s.c)
}
