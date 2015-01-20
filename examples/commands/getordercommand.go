package commands

import (
	"errors"
	"math/rand"
	"net/http"
	"time"

	"github.com/arjantop/cuirass"
	"golang.org/x/net/context"
)

func NewGetOrderCommand(ex cuirass.Executor, orderId int) *cuirass.Command {
	return cuirass.NewCommand("GetOrderCommand", func(ctx context.Context) (r interface{}, err error) {
		c := make(chan error, 1)
		go func() {
			c <- func() error {
				time.Sleep(time.Duration(rand.Intn(200)+50) * time.Millisecond)

				if rand.Float64() > 0.9999 {
					return errors.New("Failure getting Order")
				}

				if rand.Float64() > 0.95 {
					time.Sleep(time.Duration(rand.Intn(300)+25) * time.Millisecond)
				}

				r, err = NewOrder(ex, ctx, orderId)
				return err
			}()
		}()
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case err := <-c:
			return r, err
		}
	}).Build()
}

type Order struct {
	OrderId int
	User    *UserAccount
}

func NewOrder(ex cuirass.Executor, ctx context.Context, orderId int) (*Order, error) {
	user, err := ex.Exec(ctx, NewGetUserAccountCommand(&http.Cookie{
		Name:  "name",
		Value: "value",
	}))
	if err != nil {
		return nil, err
	}
	return &Order{
		OrderId: orderId,
		User:    user.(*UserAccount),
	}, nil
}
