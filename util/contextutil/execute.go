package contextutil

import "golang.org/x/net/context"

func Do(ctx context.Context, f func() error) error {
	return DoWithCancel(ctx, func() {}, f)
}

func DoWithCancel(ctx context.Context, cancel func(), f func() error) error {
	c := make(chan error, 1)
	go func() { c <- f() }()
	select {
	case <-ctx.Done():
		cancel()
		<-c
		return ctx.Err()
	case err := <-c:
		return err
	}
}
