package num

import (
	"sort"
	"sync"
	"time"

	"github.com/arjantop/cuirass/util"
)

type RollingPercentile struct {
	bucketSize        time.Duration
	currentBucket     uint
	currentBucketTime time.Time
	buckets           []percentileBucket
	clock             util.Clock
	lock              *sync.RWMutex
}

func NewRollingPercentile(windowSize time.Duration, windowBuckets uint, clock util.Clock) *RollingPercentile {
	return &RollingPercentile{
		bucketSize:        calculateBucketSize(windowSize, windowBuckets),
		currentBucket:     0,
		currentBucketTime: clock.Now(),
		buckets:           make([]percentileBucket, windowBuckets),
		clock:             clock,
		lock:              new(sync.RWMutex),
	}
}

func (p *RollingPercentile) BucketSize() time.Duration {
	p.lock.RLock()
	defer p.lock.RUnlock()
	return p.bucketSize
}

func (p *RollingPercentile) Add(v int) {
	p.findCurrentBucket().add(v)
}

// TODO: try to extract common functionality with RollingNumber.
func (p *RollingPercentile) findCurrentBucket() *percentileBucket {
	p.lock.Lock()
	defer p.lock.Unlock()

	now := p.clock.Now()
	timeDiffFromFirstBucket := now.Sub(p.currentBucketTime)
	bucketsBehind := uint(timeDiffFromFirstBucket / p.bucketSize)
	if bucketsBehind > 0 {
		numBuckets := uint(len(p.buckets))
		for i := uint(1); i <= bucketsBehind%numBuckets; i++ {
			p.buckets[(p.currentBucket+i)%numBuckets].reset()
		}
		p.currentBucket = (p.currentBucket + bucketsBehind) % numBuckets
		p.currentBucketTime = now
	}
	return &p.buckets[p.currentBucket]
}

func (p *RollingPercentile) Get(percentile float64) int {
	// Update the current bucket.
	p.findCurrentBucket()
	p.lock.Lock()
	defer p.lock.Unlock()
	values := make([]int, 0)
	for _, b := range p.buckets {
		// TODO: use copy
		for _, v := range b.values() {
			values = append(values, v)
		}
	}
	sort.Sort(sort.IntSlice(values))
	return calculatePercentile(percentile, values)
}

// TODO: cache the calculation.
func (p *RollingPercentile) Mean() int {
	// Update the current bucket.
	p.findCurrentBucket()
	p.lock.Lock()
	defer p.lock.Unlock()
	count := int64(0)
	sum := int64(0)
	for _, b := range p.buckets {
		for _, v := range b.values() {
			count += 1
			sum += int64(v)
		}
	}
	if count == 0 {
		return 0
	}
	return int(sum / count)
}

// TODO: Use the method with interpolation
func calculatePercentile(p float64, values []int) int {
	if len(values) == 0 {
		return 0
	}
	if p <= 0 {
		return values[0]
	} else if p >= 100 {
		return values[len(values)-1]
	}
	percentileIndex := p / 100.0 * float64(len(values))
	index := round(percentileIndex)
	if index > len(values)-1 {
		index = len(values) - 1
	}
	return values[index]
}

func round(v float64) int {
	if v < 0.0 {
		v -= 0.5
	} else {
		v += 0.5
	}
	return int(v)
}

type percentileBucket struct {
	vals []int
}

func (b *percentileBucket) add(v int) {
	b.vals = append(b.vals, v)
}

func (b *percentileBucket) values() []int {
	return b.vals
}

func (b *percentileBucket) reset() {
	b.vals = make([]int, 0)
}
