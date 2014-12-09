// Port of Hystrix usage example from: https://github.com/Netflix/Hystrix/blob/master/hystrix-examples/src/main/java/com/netflix/hystrix/examples/demo/HystrixCommandDemo.java
package main

import (
	"bytes"
	"fmt"
	"log"
	"math/big"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"code.google.com/p/go.net/context"
	"github.com/arjantop/cuirass"
	"github.com/arjantop/cuirass/examples/commands"
	"github.com/arjantop/cuirass/metrics"
	"github.com/arjantop/cuirass/requestcache"
	"github.com/arjantop/cuirass/requestlog"
	"github.com/arjantop/vaquita"
)

func main() {
	rand.Seed(time.Now().UnixNano())
	executor := cuirass.NewExecutor(vaquita.NewEmptyMapConfig())
	monitorMetrics(executor)
	for {
		ctx := requestlog.WithRequestLog(requestcache.WithRequestCache(context.Background()))

		simulateRequest(executor, ctx)

		log.Printf("Request => %s", requestlog.FromContext(ctx).String())
	}
}

func monitorMetrics(exec *cuirass.Executor) {
	timer := time.NewTimer(5 * time.Second)
	go func() {
		for {
			select {
			case <-timer.C:
				var b bytes.Buffer
				for _, m := range exec.Metrics().All() {
					b.WriteString("# " + m.CommandName() + ": " + commandMetricsAsString(m))
					b.WriteString("\n")
				}
				fmt.Println(b.String())
				timer.Reset(5 * time.Second)
			}
		}
	}()
}

func commandMetricsAsString(m *metrics.CommandMetrics) string {
	var b bytes.Buffer
	b.WriteString("Requests: " + strconv.Itoa(m.TotalRequests()))
	b.WriteString(" Errors: " + strconv.Itoa(m.ErrorCount()) + " (" + strconv.Itoa(m.ErrorPercentage()) + "%)")
	b.WriteString(" 75th: " + strconv.Itoa(toMilliseconds(m.ExecutionTimePercentile(75))))
	b.WriteString(" 90th: " + strconv.Itoa(toMilliseconds(m.ExecutionTimePercentile(90))))
	b.WriteString(" 99th: " + strconv.Itoa(toMilliseconds(m.ExecutionTimePercentile(99))))
	return b.String()
}

func toMilliseconds(d time.Duration) int {
	return int(d.Nanoseconds() / int64(time.Millisecond))
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
