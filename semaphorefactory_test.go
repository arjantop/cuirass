package cuirass_test

import (
	"testing"

	"github.com/arjantop/cuirass"
	"github.com/stretchr/testify/assert"
)

func newTestringSemaphoreFactory() *cuirass.SemaphoreFactory {
	return cuirass.NewSemaphoreFactory()
}

func TestSemaphoreFactoryGetSameInstance(t *testing.T) {
	sf := newTestringSemaphoreFactory()
	s1 := sf.Get("s1", 1)
	assert.True(t, s1.TryAcquire())
	s2 := sf.Get("s1", 1)
	assert.False(t, s2.TryAcquire())
}

func TestSemaphoreFactoryGetSemaphoreUniqueKeys(t *testing.T) {
	sf := newTestringSemaphoreFactory()
	s1 := sf.Get("s1", 1)
	assert.True(t, s1.TryAcquire())
	s2 := sf.Get("s2", 1)
	assert.True(t, s2.TryAcquire())
}

func TestSemaphoreFactoryGetChangedCapacity(t *testing.T) {
	sf := newTestringSemaphoreFactory()
	s1 := sf.Get("s1", 1)
	assert.True(t, s1.TryAcquire())
	s2 := sf.Get("s1", 2)
	assert.True(t, s2.TryAcquire(), "Acquired resources are reset to zero")
	assert.True(t, s2.TryAcquire())
}
