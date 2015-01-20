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

	"github.com/arjantop/cuirass"
	"github.com/arjantop/cuirass/examples/commands"
	"github.com/arjantop/cuirass/metrics"
	"github.com/arjantop/cuirass/metricsstream"
	"github.com/arjantop/cuirass/requestcache"
	"github.com/arjantop/cuirass/requestlog"
	"github.com/arjantop/vaquita"
	"golang.org/x/net/context"
)

func main() {
	rand.Seed(time.Now().UnixNano())
	executor := cuirass.NewExecutor(vaquita.NewMapConfig(map[string]string{
		"cuirass.command.CreditCardCommand.execution.isolation.thread.timeoutInMilliseconds":     "3000",
		"cuirass.command.GetUserAccountCommand.execution.isolation.thread.timeoutInMilliseconds": "50",
	}))
	monitorMetrics(executor)
	http.HandleFunc("/payment", func(w http.ResponseWriter, r *http.Request) {
		ctx := requestlog.WithRequestLog(requestcache.WithRequestCache(context.Background()))

		simulateRequest(executor, ctx)

		fmt.Fprintf(w, "Request => %s\n", requestlog.FromContext(ctx).String())
	})
	http.Handle("/cuirass.stream", metricsstream.NewMetricsStream(executor))
	log.Fatal(http.ListenAndServe(":8989", nil))
}

func monitorMetrics(exec *cuirass.CommandExecutor) {
	timer := time.NewTimer(10 * time.Second)
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
				timer.Reset(10 * time.Second)
			}
		}
	}()
}

func commandMetricsAsString(m *metrics.CommandMetrics) string {
	var b bytes.Buffer
	b.WriteString("Requests: " + strconv.Itoa(m.TotalRequests()))
	b.WriteString(" Errors: " + strconv.Itoa(m.ErrorCount()) + " (" + strconv.Itoa(m.ErrorPercentage()) + "%)")
	b.WriteString(" Mean: " + strconv.Itoa(toMilliseconds(m.ExecutionTimeMean())))
	b.WriteString(" 75th: " + strconv.Itoa(toMilliseconds(m.ExecutionTimePercentile(75))))
	b.WriteString(" 90th: " + strconv.Itoa(toMilliseconds(m.ExecutionTimePercentile(90))))
	b.WriteString(" 99th: " + strconv.Itoa(toMilliseconds(m.ExecutionTimePercentile(99))))
	return b.String()
}

func toMilliseconds(d time.Duration) int {
	return int(d.Nanoseconds() / int64(time.Millisecond))
}

func simulateRequest(executor *cuirass.CommandExecutor, ctx context.Context) {
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
