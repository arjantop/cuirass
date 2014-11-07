package num

import "time"

var (
	DefaultWindowSize    time.Duration = 10000 * time.Millisecond
	DefaultWindowBuckets uint          = 10
)

type RollingNumber struct {
	bucketSize        time.Duration
	currentBucket     uint
	currentBucketTime time.Time
	buckets           []uint64
}

func NewRollingNumber(windowSize time.Duration, windowBuckets uint) *RollingNumber {
	return &RollingNumber{
		bucketSize:        calculateBucketSize(windowSize, windowBuckets),
		currentBucket:     0,
		currentBucketTime: time.Now(),
		buckets:           make([]uint64, windowBuckets),
	}
}

func calculateBucketSize(windowSize time.Duration, windowBuckets uint) time.Duration {
	bucketSize := windowSize / time.Duration(windowBuckets)
	if bucketSize < time.Millisecond {
		return time.Millisecond
	}
	return bucketSize
}

func (n *RollingNumber) BucketSize() time.Duration {
	return n.bucketSize
}

func (n *RollingNumber) Increment() {
	n.buckets[n.findCurrentBucket()] += 1
}

func (n *RollingNumber) Sum() uint64 {
	// We don;t need the current bucket but we must still recalculate which bucket is current
	// and reset values that are no longer valid.
	n.findCurrentBucket()
	sum := uint64(0)
	for _, v := range n.buckets {
		sum += v
	}
	return sum
}

func (n *RollingNumber) Reset() {
	n.currentBucket = 0
	n.currentBucketTime = time.Now()
	for i, _ := range n.buckets {
		n.buckets[i] = 0
	}
}

func (n *RollingNumber) findCurrentBucket() uint {
	now := time.Now()
	timeDiffFromFirstBucket := now.Sub(n.currentBucketTime)
	bucketsBehind := uint(timeDiffFromFirstBucket / n.bucketSize)
	if bucketsBehind > 0 {
		for i := uint(1); i <= bucketsBehind; i++ {
			n.buckets[(n.currentBucket+i)%uint(len(n.buckets))] = 0
		}
		n.currentBucket = (n.currentBucket + bucketsBehind) % uint(len(n.buckets))
		n.currentBucketTime = now
	}
	return n.currentBucket
}
