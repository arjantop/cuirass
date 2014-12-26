package requestlog

import "golang.org/x/net/context"

type key int

const requestLogKey key = 0

func WithRequestLog(ctx context.Context) context.Context {
	return context.WithValue(ctx, requestLogKey, newRequestLog())
}

func FromContext(ctx context.Context) *RequestLog {
	requestLog, _ := ctx.Value(requestLogKey).(*RequestLog)
	return requestLog
}
