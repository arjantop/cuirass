package requestcache

import "golang.org/x/net/context"

type key int

const requestCacheKey key = 0

func WithRequestCache(ctx context.Context) context.Context {
	return context.WithValue(ctx, requestCacheKey, newRequestCache())
}

func FromContext(ctx context.Context) *RequestCache {
	requestCache, _ := ctx.Value(requestCacheKey).(*RequestCache)
	return requestCache
}
