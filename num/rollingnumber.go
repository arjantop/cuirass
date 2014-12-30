package num

import (
	"sync"
	"time"

	"github.com/arjantop/cuirass/util"
)

var (
	// Default size of statistical window over which the rolling number is defined.
	DefaultWindowSize time.Duration = 10000 * time.Millisecond
	// number of buckets in the statistical window.
	DefaultWindowBuckets int = 10
)

// RollingNumber is an implementation of a number that is defined for a specified
// window. Value changes that fall out of the sliding window are discarded.
// The implementation is thread-safe.
type RollingNumber struct {
	bucketSize        time.Duration
	currentBucket     int
	currentBucketTime time.Time
	buckets           []int64
	clock             util.Clock
	lock              *sync.RWMutex
}

// NewRollingNumber constructs a new RollingNumber with a default value of zero.
func NewRollingNumber(windowSize time.Duration, windowBuckets int, clock util.Clock) *RollingNumber {
	return &RollingNumber{
		bucketSize:        calculateBucketSize(windowSize, windowBuckets),
		currentBucket:     0,
		currentBucketTime: clock.Now(),
		buckets:           make([]int64, windowBuckets),
		clock:             clock,
		lock:              new(sync.RWMutex),
	}
}

// calculatebucketsize calculates a bucket size based on requested window size
// and number of buckets. The smallest bucket size is 1 millisecond.`
// Actual window size can be larger if the requested number of buckets does not fit in
// the window size.
// TODO: move to separate file
func calculateBucketSize(windowSize time.Duration, windowBuckets int) time.Duration {
	bucketSize := windowSize / time.Duration(windowBuckets)
	if bucketSize < time.Millisecond {
		// Calculated bucket size is smaller thatn the minimum.
		return time.Millisecond
	}
	return bucketSize
}

// BucketSize returns the calculated bucket size based on requested sliding window parameters.
func (n *RollingNumber) BucketSize() time.Duration {
	n.lock.RLock()
	size := n.bucketSize
	n.lock.RUnlock()
	return size
}

// Increment increments the number by one.
func (n *RollingNumber) Increment() {
	n.lock.Lock()
	n.buckets[n.findCurrentBucket()] += 1
	n.lock.Unlock()
}

// Sum sums all the bucket values of the sliding window and returns the result.
func (n *RollingNumber) Sum() int64 {
	n.lock.Lock()
	// We don't need the current bucket but we must still recalculate which bucket is current
	// and reset values that are no longer valid.
	n.findCurrentBucket()
	sum := int64(0)
	for _, v := range n.buckets {
		sum += v
	}
	n.lock.Unlock()
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
func (n *RollingNumber) findCurrentBucket() int {
	now := n.clock.Now()
	timeDiffFromFirstBucket := now.Sub(n.currentBucketTime)
	bucketsBehind := int(timeDiffFromFirstBucket / n.bucketSize)
	if bucketsBehind > 0 {
		// We are not in the current bucket so we must reset the values of
		// buckets that fell out of the sliding window.
		numBuckets := len(n.buckets)
		for i := 1; i <= bucketsBehind; i++ {
			n.buckets[(n.currentBucket+i)%numBuckets] = 0
		}
		n.currentBucket = (n.currentBucket + bucketsBehind) % numBuckets
		n.currentBucketTime = now
	}
	return n.currentBucket
}
