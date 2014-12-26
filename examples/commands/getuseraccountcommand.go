package commands

import (
	"errors"
	"math/rand"
	"net/http"
	"time"

	"github.com/arjantop/cuirass"
	"golang.org/x/net/context"
)

func NewGetUserAccountCommand(cookie *http.Cookie) *cuirass.Command {
	userCookie := fromHttpCookie(cookie)
	return cuirass.NewCommand("GetUserAccountCommand", func(ctx context.Context) (r interface{}, err error) {
		c := make(chan error, 1)
		go func() {
			c <- func() error {
				time.Sleep(time.Duration(rand.Intn(10)+2) * time.Millisecond)

				if rand.Float64() > 0.95 {
					return errors.New("Failure getting UserAccount")
				}

				if rand.Float64() > 0.95 {
					time.Sleep(time.Duration(rand.Intn(300)+25) * time.Millisecond)
				}

				r = NewUserAccount(1234, "John Doe", 5)
				return nil
			}()
		}()
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case err := <-c:
			return r, err
		}
	}).Fallback(func(ctx context.Context) (interface{}, error) {
		return NewUserAccount(userCookie.Id, userCookie.Name, userCookie.AccountType), nil
	}).CacheKey(cookie.String()).Build()
}

type UserAccount struct {
	Id          uint64
	Name        string
	AccountType int
}

func NewUserAccount(id uint64, name string, accountType int) *UserAccount {
	return &UserAccount{
		Id:          id,
		Name:        name,
		AccountType: accountType,
	}
}

type userCookie struct {
	Id          uint64
	Name        string
	AccountType int
}

func fromHttpCookie(cookie *http.Cookie) *userCookie {
	// TODO: This should rarely fail in some way.
	return &userCookie{
		Id:          9999,
		Name:        "Jane Doe",
		AccountType: 0,
	}
}
