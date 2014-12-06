package num

import (
	"time"

	"github.com/arjantop/cuirass/util"
)

var (
	// Default size of statistical window over which the rolling number is defined.
	DefaultWindowSize time.Duration = 10000 * time.Millisecond
	// number of buckets in the statistical window.
	DefaultWindowBuckets uint = 10
)

// RollingNumber is an implementation of a number that is defined for a specified
// window. Value changes that fall out of the sliding window are discarded.
type RollingNumber struct {
	bucketSize        time.Duration
	currentBucket     uint
	currentBucketTime time.Time
	buckets           []uint64
	clock             util.Clock
}

// NewRollingNumber constructs a new RollingNumber with a default value of zero.
func NewRollingNumber(windowSize time.Duration, windowBuckets uint, clock util.Clock) *RollingNumber {
	return &RollingNumber{
		bucketSize:        calculateBucketSize(windowSize, windowBuckets),
		currentBucket:     0,
		currentBucketTime: clock.Now(),
		buckets:           make([]uint64, windowBuckets),
		clock:             clock,
	}
}

// calculatebucketsize calculates a bucket size based on requested window size
// and number of buckets. The smallest bucket size is 1 millisecond.`
// Actual window size can be larger if the requested number of buckets does not fit in
// the window size.
func calculateBucketSize(windowSize time.Duration, windowBuckets uint) time.Duration {
	bucketSize := windowSize / time.Duration(windowBuckets)
	if bucketSize < time.Millisecond {
		// Calculated bucket size is smaller thatn the minimum.
		return time.Millisecond
	}
	return bucketSize
}

// BucketSize returns the calculated bucket size based on requested sliding window parameters.
func (n *RollingNumber) BucketSize() time.Duration {
	return n.bucketSize
}

// Increment increments the number by one.
func (n *RollingNumber) Increment() {
	n.buckets[n.findCurrentBucket()] += 1
}

// Sum sums all the bucket values of the sliding window and returns the result.
func (n *RollingNumber) Sum() uint64 {
	// We don't need the current bucket but we must still recalculate which bucket is current
	// and reset values that are no longer valid.
	n.findCurrentBucket()
	sum := uint64(0)
	for _, v := range n.buckets {
		sum += v
	}
	return sum
}

// Reset resets a number to a default value.
func (n *RollingNumber) Reset() {
	n.currentBucket = 0
	n.currentBucketTime = n.clock.Now()
	for i, _ := range n.buckets {
		n.buckets[i] = 0
	}
}

// findCurrentBucket returns an index of the current bucket and updates the buckets
// that should be reset for reuse based on the time elapsed since last access.
func (n *RollingNumber) findCurrentBucket() uint {
	now := n.clock.Now()
	timeDiffFromFirstBucket := now.Sub(n.currentBucketTime)
	bucketsBehind := uint(timeDiffFromFirstBucket / n.bucketSize)
	if bucketsBehind > 0 {
		// We are not in the current bucket so we must reset the values of
		// buckets that fell out of the sliding window.
		numBuckets := uint(len(n.buckets))
		for i := uint(1); i <= bucketsBehind%numBuckets; i++ {
			n.buckets[(n.currentBucket+i)%numBuckets] = 0
		}
		n.currentBucket = (n.currentBucket + bucketsBehind) % numBuckets
		n.currentBucketTime = now
	}
	return n.currentBucket
}
