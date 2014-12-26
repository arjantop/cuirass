package commands

import (
	"errors"
	"math/rand"
	"time"

	"github.com/arjantop/cuirass"
	"golang.org/x/net/context"
)

func NewGetPaymentInformationCommand(user *UserAccount) *cuirass.Command {
	return cuirass.NewCommand("GetPaymentInformationCommand", func(ctx context.Context) (r interface{}, err error) {
		c := make(chan error, 1)
		go func() {
			c <- func() error {
				time.Sleep(time.Duration(rand.Intn(10)+2) * time.Millisecond)

				if rand.Float64() > 0.9999 {
					return errors.New("Failure getting PaymentInformation")
				}

				if rand.Float64() > 0.98 {
					time.Sleep(time.Duration(rand.Intn(100)+25) * time.Millisecond)
				}

				r = NewPaymentInformation(user, "4444888833337777", 12, 15)
				return nil
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

type Expiration struct {
	Month, Year int
}

type PaymentInformation struct {
	UserAccount      *UserAccount
	CreditCardNumber string
	Expiration       Expiration
}

func NewPaymentInformation(
	userAccount *UserAccount,
	creditCardNumber string,
	expirationMonth, expirationYear int) *PaymentInformation {

	return &PaymentInformation{
		UserAccount:      userAccount,
		CreditCardNumber: creditCardNumber,
		Expiration: Expiration{
			Month: expirationMonth,
			Year:  expirationYear,
		},
	}
}
