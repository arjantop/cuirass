package requestcache

import (
	"sync"

	"github.com/arjantop/cuirass"
	"github.com/arjantop/cuirass/requestlog"
)

type cacheKey struct {
	commandName string
	key         string
}

func keyForCommand(cmd *cuirass.Command) cacheKey {
	return cacheKey{cmd.Name(), cmd.CacheKey()}
}

type ExecutedCommand struct {
	response interface{}
	err      error
	info     *requestlog.ExecutionInfo
}

func (c *ExecutedCommand) Response() (interface{}, error) {
	return c.response, c.err
}

func (c *ExecutedCommand) ExecutionInfo() *requestlog.ExecutionInfo {
	return c.info
}

type RequestCache struct {
	cache     map[cacheKey]*ExecutedCommand
	cacheLock *sync.RWMutex
}

func newRequestCache() *RequestCache {
	return &RequestCache{
		cache:     make(map[cacheKey]*ExecutedCommand),
		cacheLock: new(sync.RWMutex),
	}
}

func (c *RequestCache) Get(cmd *cuirass.Command) *ExecutedCommand {
	c.cacheLock.RLock()
	defer c.cacheLock.RUnlock()
	r := c.cache[keyForCommand(cmd)]
	if r != nil {
		return &ExecutedCommand{
			response: r.response,
			err:      r.err,
			info:     r.info,
		}
	}
	return nil
}

func (c *RequestCache) Add(
	cmd *cuirass.Command,
	info *requestlog.ExecutionInfo,
	r interface{},
	err error) bool {

	if !cmd.CanBeCached() {
		return false
	}

	cachedEvents := append(info.Events(), requestlog.ResponseFromCache)
	// Execution time of cached commands is always 0ms.
	cachedInfo := requestlog.NewExecutionInfo(info.CommandName(), 0, cachedEvents)

	c.cacheLock.Lock()
	defer c.cacheLock.Unlock()

	c.cache[keyForCommand(cmd)] = &ExecutedCommand{
		response: r,
		err:      err,
		info:     cachedInfo,
	}
	return true
}
