package requestcache

import (
	"sync"

	"github.com/arjantop/cuirass/requestlog"
)

// cacheKey is a key used for request cace value identification based on command
// name and command's cache key.
type cacheKey struct {
	commandName string
	key         string
}

// newCacheKey constructs a new cache key.
func newCacheKey(name, key string) cacheKey {
	return cacheKey{name, key}
}

// ExecutedCommand holds a response or response error and the execution info
// of the previously executed command.
type ExecutedCommand struct {
	response interface{}
	err      error
	info     requestlog.ExecutionInfo
}

// Response returns a response or error stored from command execution.
func (c *ExecutedCommand) Response() (interface{}, error) {
	return c.response, c.err
}

// ExecutionInfo returns an ExecutionInfo for the executed command.
// Execution time for cached responses is always zero.
func (c *ExecutedCommand) ExecutionInfo() *requestlog.ExecutionInfo {
	return &c.info
}

// ResponseCache is a cache of command responses.
// It is safe to access ResponseCache from multiple thread.
type RequestCache struct {
	cache     map[cacheKey]*ExecutedCommand
	cacheLock *sync.RWMutex
}

// newRequestCache constructs a new empty request cache.
func newRequestCache() *RequestCache {
	return &RequestCache{
		cache:     make(map[cacheKey]*ExecutedCommand),
		cacheLock: new(sync.RWMutex),
	}
}

// Get performs a lookup in the cache for the cached command with the given name
// and key.
// If no previous execution is in the cache nil is returned.
func (c *RequestCache) Get(name, key string) *ExecutedCommand {
	c.cacheLock.RLock()
	defer c.cacheLock.RUnlock()
	r := c.cache[newCacheKey(name, key)]
	if r != nil {
		return &ExecutedCommand{
			response: r.response,
			err:      r.err,
			info:     r.info,
		}
	}
	return nil
}

// Add adds a command response and execution info to the cache to be reused by
// later executions of the command.
func (c *RequestCache) Add(
	name, key string,
	info requestlog.ExecutionInfo,
	r interface{},
	err error) {

	cachedEvents := append(info.Events(), requestlog.ResponseFromCache)
	// Execution time of cached commands is always 0ms.
	cachedInfo := requestlog.NewExecutionInfo(info.CommandName(), 0, cachedEvents)

	c.cacheLock.Lock()
	defer c.cacheLock.Unlock()

	c.cache[newCacheKey(name, key)] = &ExecutedCommand{
		response: r,
		err:      err,
		info:     cachedInfo,
	}
}
