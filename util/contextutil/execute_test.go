package contextutil_test

import (
	"errors"
	"testing"
	"time"

	"github.com/arjantop/cuirass/util/contextutil"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
)

func TestDoErrorReturned(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	err := contextutil.Do(ctx, func() error {
		return errors.New("foo")
	})
	assert.Equal(t, errors.New("foo"), err)
}

func TestDoWithCancelErrorReturned(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	err := contextutil.DoWithCancel(ctx, func() {}, func() error {
		return errors.New("foo")
	})
	assert.Equal(t, errors.New("foo"), err)
}

func TestDoWithCancelTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Nanosecond)
	defer cancel()
	var called bool
	err := contextutil.DoWithCancel(ctx, func() { called = true }, func() error {
		time.Sleep(2 * time.Nanosecond)
		return nil
	})
	assert.Equal(t, context.DeadlineExceeded, err)
}
