# Cuirass (in development)

Cuirass is a latency and fault tolerance library inspired by [hystrix](https://github.com/Netflix/Hystrix) written in Go.
It provides isolation when accessing remote systems with support for fallback when things go wrong.
Remote execution is protected with timeouts and circuit breakers to fail fast and let the system recover.

## Run example

Get and run the example:
```
$ https://github.com/arjantop/cuirass.git
$ cd cuirass
$ go run examples/main.go
```

Sample output:
```
2014/11/12 12:15:54 context deadline exceeded
2014/11/12 12:15:54 Request => GetUserAccountCommand[FAILURE, FALLBACK_SUCCESS][4ms], GetPaymentInformationCommand[SUCCESS][6ms], GetUserAccountCommand[RESPONSE_FROM_CACHE][0ms]x2, GetOrderCommand[SUCCESS][64ms], CreditCardCommand[TIMEOUT][1000ms]
2014/11/12 12:15:59 Request => GetUserAccountCommand[SUCCESS][6ms], GetPaymentInformationCommand[SUCCESS][4ms], GetUserAccountCommand[SUCCESS, RESPONSE_FROM_CACHE][0ms]x2, GetOrderCommand[SUCCESS][172ms], CreditCardCommand[SUCCESS][965ms]
2014/11/12 12:16:00 Request => GetUserAccountCommand[SUCCESS][6ms], GetPaymentInformationCommand[SUCCESS][3ms], GetUserAccountCommand[SUCCESS, RESPONSE_FROM_CACHE][0ms]x2, GetOrderCommand[SUCCESS][136ms], CreditCardCommand[SUCCESS][929ms]
2014/11/12 12:16:01 context deadline exceeded
2014/11/12 12:16:01 Request => GetUserAccountCommand[SUCCESS][11ms], GetPaymentInformationCommand[SUCCESS][85ms], GetUserAccountCommand[SUCCESS, RESPONSE_FROM_CACHE][0ms]x2, GetOrderCommand[SUCCESS][149ms], CreditCardCommand[TIMEOUT][1000ms]
```

## TODO

* Request Collapsing/Batching

## Hystrix Dashboard

Cuirass metrics can be monitored using the [hystrix-dashboard](https://github.com/Netflix/Hystrix/tree/master/hystrix-dashboard).

![Hystrix Dashboard](https://raw.github.com/arjantop/cuirass/develop/images/dashboard.jpg)
