// Port of Hystrix usage example from: https://github.com/Netflix/Hystrix/blob/master/hystrix-examples/src/main/java/com/netflix/hystrix/examples/demo/HystrixCommandDemo.java
package main

import (
	"log"
	"math/big"
	"math/rand"
	"net/http"
	"time"

	"code.google.com/p/go.net/context"
	"github.com/arjantop/cuirass"
	"github.com/arjantop/cuirass/examples/commands"
	"github.com/arjantop/cuirass/requestcache"
	"github.com/arjantop/cuirass/requestlog"
	"github.com/arjantop/vaquita"
)

func main() {
	rand.Seed(time.Now().UnixNano())
	executor := cuirass.NewExecutor(vaquita.NewEmptyMapConfig(), 1*time.Second)
	for {
		ctx := requestlog.WithRequestLog(requestcache.WithRequestCache(context.Background()))

		simulateRequest(executor, ctx)

		log.Printf("Request => %s", requestlog.FromContext(ctx).String())
	}
}

func simulateRequest(executor *cuirass.Executor, ctx context.Context) {
	user, err := executor.Exec(ctx, commands.NewGetUserAccountCommand(&http.Cookie{
		Name:  "name",
		Value: "value",
	}))
	if err != nil {
		log.Println(err)
		return
	}

	paymentInformation, err := executor.Exec(ctx, commands.NewGetPaymentInformationCommand(user.(*commands.UserAccount)))
	if err != nil {
		log.Println(err)
		return
	}

	order, err := executor.Exec(ctx, commands.NewGetOrderCommand(executor, rand.Intn(20000)+9100))
	if err != nil {
		log.Println(err)
		return
	}

	amount := new(big.Rat)
	amount.SetString("123.45")

	_, err = executor.Exec(ctx, commands.NewCreditCardCommand(
		executor,
		&commands.AuthorizeNetGateway{},
		order.(*commands.Order),
		paymentInformation.(*commands.PaymentInformation),
		amount))

	if err != nil {
		log.Println(err)
		return
	}
}
