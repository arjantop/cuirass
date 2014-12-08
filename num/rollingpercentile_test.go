package num_test

import (
	"testing"
	"time"

	"github.com/arjantop/cuirass/num"
	"github.com/arjantop/cuirass/util"
	"github.com/stretchr/testify/assert"
)

func newTestingRollingPercentile(clock util.Clock) *num.RollingPercentile {
	if clock == nil {
		clock = util.NewClock()
	}
	return num.NewRollingPercentile(time.Millisecond, 3, clock)
}

func TestRollingPercentileBucketSize(t *testing.T) {
	n := num.NewRollingPercentile(time.Minute, 30, util.NewClock())
	assert.Equal(t, 2*time.Second, n.BucketSize())
	n2 := newTestingRollingPercentile(nil)
	assert.Equal(t, time.Millisecond, n2.BucketSize())
}

func TestRollingPercentileBasic(t *testing.T) {
	rp := newTestingRollingPercentile(nil)
	addAll(rp, 3, 10, 7, 5)

	// 0th
	assert.Equal(t, 3, rp.Get(0))
	assert.Equal(t, 3, rp.Get(-10))

	// 100th
	assert.Equal(t, 10, rp.Get(100))
	assert.Equal(t, 10, rp.Get(120))
}

func TestRollingPercentileAll(t *testing.T) {
	rp := newTestingRollingPercentile(nil)
	addAll(rp, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10)

	assert.Equal(t, 1, rp.Get(0))
	assert.Equal(t, 2, rp.Get(10))
	assert.Equal(t, 3, rp.Get(20))
	assert.Equal(t, 4, rp.Get(30))
	assert.Equal(t, 5, rp.Get(40))
	assert.Equal(t, 6, rp.Get(50))
	assert.Equal(t, 7, rp.Get(60))
	assert.Equal(t, 8, rp.Get(70))
	assert.Equal(t, 9, rp.Get(80))
	assert.Equal(t, 10, rp.Get(90))
	assert.Equal(t, 10, rp.Get(100))
}

func TestRollingPercentileInWindow(t *testing.T) {
	clock := util.NewTestableClock(time.Now())
	rp := newTestingRollingPercentile(clock)

	addAll(rp, 1)
	assert.Equal(t, 1, rp.Get(100))
	clock.Add(time.Millisecond)
	addAll(rp, 3)
	assert.Equal(t, 3, rp.Get(100))
	clock.Add(time.Millisecond)
	addAll(rp, 2)
	assert.Equal(t, 3, rp.Get(100))
	clock.Add(time.Millisecond)
	assert.Equal(t, 3, rp.Get(100), "The value 3 is still in the window")
	clock.Add(time.Millisecond)
	assert.Equal(t, 2, rp.Get(100))
	clock.Add(time.Millisecond)
	assert.Equal(t, 0, rp.Get(100))
}

func addAll(rp *num.RollingPercentile, vs ...int) {
	for _, v := range vs {
		rp.Add(v)
	}
}
