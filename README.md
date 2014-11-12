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

## Hystrix Dashboard

The plan is to make it compatible with [hystrix-dashboard](https://github.com/Netflix/Hystrix/tree/master/hystrix-dashboard).
