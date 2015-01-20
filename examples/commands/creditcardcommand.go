package commands

import (
	"math/big"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/arjantop/cuirass"
	"golang.org/x/net/context"
)

func NewCreditCardCommand(
	ex cuirass.Executor, gateway *AuthorizeNetGateway,
	order *Order,
	payment *PaymentInformation,
	amount *big.Rat) *cuirass.Command {

	return cuirass.NewCommand("CreditCardCommand", func(ctx context.Context) (r interface{}, err error) {
		c := make(chan error, 1)
		go func() {
			c <- func() error {
				user, err := ex.Exec(ctx, NewGetUserAccountCommand(&http.Cookie{
					Name:  "name",
					Value: "value",
				}))
				if err != nil {
					return err
				}
				if user.(*UserAccount).AccountType == 1 {
					// do something
				} else {
					// do something else
				}

				authResult := gateway.Submit(
					payment.CreditCardNumber,
					strconv.Itoa(payment.Expiration.Month),
					strconv.Itoa(payment.Expiration.Year),
					AuthCapture, amount, order)

				if authResult.IsApproved() {
					r = NewSuccessResponse(authResult.Target().TransactionId(), authResult.Target().AuthorizationCode())
				} else if authResult.IsDeclined() {
					r = NewFailedResponse(strconv.Itoa(authResult.ReasonResponseCode()) + ":" + authResult.ResponseText())
				}

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

type CreditCardAuthorizationResult struct {
	Success                          bool
	AuthorizationCode, TransactionId string
	ErrorMessage                     string
}

func NewSuccessResponse(transactionId, authorizationCode string) CreditCardAuthorizationResult {
	return CreditCardAuthorizationResult{
		Success:           true,
		AuthorizationCode: authorizationCode,
		TransactionId:     transactionId,
	}
}

func NewFailedResponse(message string) CreditCardAuthorizationResult {
	return CreditCardAuthorizationResult{
		Success:      false,
		ErrorMessage: message,
	}
}

type TransactionType int

const AuthCapture TransactionType = 0

type AuthorizeNetGateway struct {
}

func (g *AuthorizeNetGateway) Submit(
	creditCardNumber, expirationMonth, expirationYear string,
	authCapture TransactionType,
	amount *big.Rat,
	order *Order) Result {

	time.Sleep(time.Duration(rand.Intn(700)+800) * time.Millisecond)

	if rand.Float64() > 0.99 {
		time.Sleep(time.Duration(8000) * time.Millisecond)
	}
	if rand.Float64() < 0.8 {
		return NewResult(true)
	}
	return NewResult(false)
}

type Result struct {
	approved bool
}

func NewResult(approved bool) Result {
	return Result{approved}
}

func (r *Result) IsApproved() bool {
	return r.approved
}

func (r *Result) IsDeclined() bool {
	return !r.approved
}

func (r *Result) ResponseText() string {
	return ""
}

func (r *Result) Target() *Target {
	return NewTarget()
}

func (r *Result) ReasonResponseCode() int {
	return 0
}

type Target struct{}

func NewTarget() *Target {
	return &Target{}
}

func (t *Target) TransactionId() string {
	return "transactionId"
}

func (t *Target) AuthorizationCode() string {
	return "authorizedCode"
}
