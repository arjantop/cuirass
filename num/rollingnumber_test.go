package num_test

import (
	"testing"
	"time"

	"github.com/arjantop/cuirass/num"
	"github.com/stretchr/testify/assert"
)

func TestMinimalBucketSizeIsMillisecond(t *testing.T) {
	n := num.NewRollingNumber(time.Millisecond, 10)
	assert.Equal(t, time.Millisecond, n.BucketSize())
}

func newTestingRollingNumber() *num.RollingNumber {
	return num.NewRollingNumber(10*time.Millisecond, 10)
}

func TestBucketSizeIsCalculated(t *testing.T) {
	n := num.NewRollingNumber(time.Minute, 30)
	assert.Equal(t, 2*time.Second, n.BucketSize())
	n2 := newTestingRollingNumber()
	assert.Equal(t, time.Millisecond, n2.BucketSize())
}

func TestRollingNumberIncrement(t *testing.T) {
	n := newTestingRollingNumber()
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

func TestRollingNumberSumInWindow(t *testing.T) {
	n := newTestingRollingNumber()
	n.Increment()
	time.Sleep(time.Millisecond)
	n.Increment()
	n.Increment()
	assert.Equal(t, 3, n.Sum())
	time.Sleep(9 * time.Millisecond)
	assert.Equal(t, 2, n.Sum())
	time.Sleep(time.Millisecond)
	assert.Equal(t, 0, n.Sum())
}

func TestRollingNumberReset(t *testing.T) {
	n := newTestingRollingNumber()
	n.Increment()
	time.Sleep(time.Millisecond)
	n.Increment()
	n.Increment()
	time.Sleep(time.Millisecond)
	n.Increment()
	n.Reset()
	assert.Equal(t, 0, n.Sum())
}
