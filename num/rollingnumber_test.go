package num_test

import (
	"testing"
	"time"

	"github.com/arjantop/cuirass/num"
	"github.com/arjantop/cuirass/util"
	"github.com/stretchr/testify/assert"
)

func TestMinimalBucketSizeIsMillisecond(t *testing.T) {
	n := num.NewRollingNumber(time.Millisecond, 10, util.NewClock())
	assert.Equal(t, time.Millisecond, n.BucketSize())
}

func newTestingRollingNumber(clock util.Clock) *num.RollingNumber {
	if clock == nil {
		clock = util.NewClock()
	}
	return num.NewRollingNumber(10*time.Millisecond, 10, clock)
}

func TestBucketSizeIsCalculated(t *testing.T) {
	n := num.NewRollingNumber(time.Minute, 30, util.NewClock())
	assert.Equal(t, 2*time.Second, n.BucketSize())
	n2 := newTestingRollingNumber(nil)
	assert.Equal(t, time.Millisecond, n2.BucketSize())
}

func TestRollingNumberIncrement(t *testing.T) {
	n := newTestingRollingNumber(nil)
	assert.Equal(t, 0, n.Sum())
	n.Increment()
	assert.Equal(t, 1, n.Sum())
	n.Increment()
	n.Increment()
	assert.Equal(t, 3, n.Sum())
	n.Increment()
	n.Increment()
	n.Increment()
	assert.Equal(t, 6, n.Sum())
}

func TestRollingNumberSleepBiggerThanWindow(t *testing.T) {
	clock := util.NewTestableClock(time.Now())
	n := newTestingRollingNumber(clock)
	n.Increment()
	clock.Add(time.Second)
	assert.Equal(t, 0, n.Sum())
}

func TestRollingNumberSumInWindow(t *testing.T) {
	clock := util.NewTestableClock(time.Now())
	n := newTestingRollingNumber(clock)
	n.Increment()
	clock.Add(time.Millisecond)
	n.Increment()
	n.Increment()
	assert.Equal(t, 3, n.Sum())
	clock.Add(9 * time.Millisecond)
	assert.Equal(t, 2, n.Sum())
	clock.Add(time.Millisecond)
	assert.Equal(t, 0, n.Sum())
}

func TestRollingNumberReset(t *testing.T) {
	clock := util.NewTestableClock(time.Now())
	n := newTestingRollingNumber(clock)
	n.Increment()
	clock.Add(time.Millisecond)
	n.Increment()
	n.Increment()
	clock.Add(time.Millisecond)
	n.Increment()
	n.Reset()
	assert.Equal(t, 0, n.Sum())
}
