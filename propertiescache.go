package cuirass

import (
	"sync"

	"github.com/arjantop/vaquita"
)

type key struct {
	name  string
	group string
	cfg   vaquita.DynamicConfig
}

var (
	cache = make(map[key]*CommandProperties)
	lock  = new(sync.Mutex)
)

func GetProperties(cfg vaquita.DynamicConfig, commandName, commandGroup string) *CommandProperties {
	lock.Lock()
	if p, ok := cache[key{commandName, commandGroup, cfg}]; ok {
		lock.Unlock()
		return p
	}
	p := newCommandProperties(cfg, commandName, commandGroup)
	cache[key{commandName, commandGroup, cfg}] = p
	lock.Unlock()
	return p
}
