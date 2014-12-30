package num

import (
	"math"
	"sort"
	"sync"
	"time"

	"github.com/arjantop/cuirass/util"
)

type RollingPercentile struct {
	bucketSize        time.Duration
	maxBucketItems    int
	currentBucket     uint
	currentBucketTime time.Time
	buckets           []percentileBucket
	clock             util.Clock
	lock              *sync.RWMutex
}

func NewRollingPercentile(windowSize time.Duration, windowBuckets, maxBucketItems int, clock util.Clock) *RollingPercentile {
	p := &RollingPercentile{
		bucketSize:        calculateBucketSize(windowSize, windowBuckets),
		maxBucketItems:    maxBucketItems,
		currentBucket:     0,
		currentBucketTime: clock.Now(),
		buckets:           make([]percentileBucket, windowBuckets),
		clock:             clock,
		lock:              new(sync.RWMutex),
	}
	for i, _ := range p.buckets {
		p.buckets[i].reset(maxBucketItems)
	}
	return p
}

func (p *RollingPercentile) BucketSize() time.Duration {
	p.lock.RLock()
	size := p.bucketSize
	p.lock.RUnlock()
	return size
}

func (p *RollingPercentile) Add(v int) {
	p.findCurrentBucket().add(v)
}

func (p *RollingPercentile) findCurrentBucket() *percentileBucket {
	p.lock.Lock()

	now := p.clock.Now()
	timeDiffFromFirstBucket := now.Sub(p.currentBucketTime)
	bucketsBehind := uint(timeDiffFromFirstBucket / p.bucketSize)
	if bucketsBehind > 0 {
		numBuckets := uint(len(p.buckets))
		for i := uint(1); i <= bucketsBehind%numBuckets; i++ {
			p.buckets[(p.currentBucket+i)%numBuckets].reset(p.maxBucketItems)
		}
		p.currentBucket = (p.currentBucket + bucketsBehind) % numBuckets
		p.currentBucketTime = now
	}
	bucket := &p.buckets[p.currentBucket]
	p.lock.Unlock()
	return bucket
}

func (p *RollingPercentile) Get(percentile float64) int {
	// Update the current bucket.
	p.findCurrentBucket()
	p.lock.Lock()
	values := make([]int, 0)
	for _, b := range p.buckets {
		// TODO: use copy
		for _, v := range b.values() {
			values = append(values, v)
		}
	}
	sort.Sort(sort.IntSlice(values))
	p.lock.Unlock()
	return calculatePercentile(percentile, values)
}

// TODO: cache the calculation.
func (p *RollingPercentile) Mean() int {
	// Update the current bucket.
	p.findCurrentBucket()
	p.lock.Lock()
	count := int64(0)
	sum := int64(0)
	for _, b := range p.buckets {
		for _, v := range b.values() {
			count += 1
			sum += int64(v)
		}
	}
	p.lock.Unlock()
	if count == 0 {
		return 0
	}
	return int(sum / count)
}

func calculatePercentile(p float64, values []int) int {
	if len(values) == 0 {
		return 0
	}
	if p <= 0 {
		return values[0]
	} else if p >= 100 {
		return values[len(values)-1]
	}
	rank := p / 100.0 * float64(len(values))
	lowIndex := int(math.Floor(rank))
	highIndex := int(math.Ceil(rank))
	if highIndex >= len(values) {
		return values[len(values)-1]
	} else if lowIndex == highIndex {
		return values[lowIndex]
	} else {
		return values[lowIndex] + int((rank-float64(lowIndex))*float64(values[highIndex]-values[lowIndex]))
	}
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
	currentPosition int
	vals            []int
}

func (b *percentileBucket) add(v int) {
	if len(b.vals) < cap(b.vals) {
		b.vals = append(b.vals, v)
	} else {
		b.vals[b.currentPosition] = v
	}
	b.currentPosition = (b.currentPosition + 1) % cap(b.vals)
}

func (b *percentileBucket) values() []int {
	return b.vals
}

func (b *percentileBucket) reset(size int) {
	if b.vals == nil || len(b.vals) != size {
		b.vals = make([]int, 0, size)
	} else {
		b.vals = b.vals[:0]
	}
	b.currentPosition = 0
}
