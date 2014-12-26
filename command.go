package cuirass

import (
	"errors"

	"github.com/arjantop/vaquita"
	"golang.org/x/net/context"
)

// FallbackNotImplemented is the error returned by Cimmand.Fallback when no fallback
// function is configured.
var FallbackNotImplemented = errors.New("Fallback not implemented")

// A CommandFunc is a function that contains the primary or fallback logic
// for the command.
type CommandFunc func(ctx context.Context) (interface{}, error)

// Command is a wrapper for a code that requires latency and fault tolerance
// (typically service call over the network).
type Command struct {
	name, group   string
	run, fallback CommandFunc
	cacheKey      string
}

// Name returns the name of the command.
func (c *Command) Name() string {
	return c.name
}

// Group returns the name of the group the command belongs to.
func (c *Command) Group() string {
	return c.group
}

// Run executes a primary function to fetch a result.
func (c *Command) Run(ctx context.Context) (interface{}, error) {
	return c.run(ctx)
}

// Fallback executes the fallback logic when primary function fails.
func (c *Command) Fallback(ctx context.Context) (interface{}, error) {
	return c.fallback(ctx)
}

// IsCacheable returns true id the response of the command execution can be cached.
func (c *Command) IsCacheable() bool {
	return c.cacheKey != ""
}

// CacheKey returns a key used for request caching.
func (c *Command) CacheKey() string {
	return c.cacheKey
}

func (c *Command) Properties(cfg vaquita.DynamicConfig) *CommandProperties {
	return GetProperties(cfg, c.Name(), c.Group())
}

// CommandBuilder is a helper used for constructing new Commands.
type CommandBuilder struct {
	name, group   string
	run, fallback CommandFunc
	cacheKey      string
}

// NewCommand constructs a new CommandBuilder with minimal required command
// implementation (name and primary function).
func NewCommand(name string, run CommandFunc) *CommandBuilder {
	return &CommandBuilder{
		name:  name,
		group: name,
		run:   run,
	}
}

// Group sets a group name for a command. The default group name is the name
// of the command. Group name is used for limiting the concurrent execution
// of the command for the entire group.
func (b *CommandBuilder) Group(name string) *CommandBuilder {
	b.group = name
	return b
}

// Fallback adds a fallback function to the command being built.
func (b *CommandBuilder) Fallback(fallback CommandFunc) *CommandBuilder {
	b.fallback = fallback
	return b
}

// CacheKey sets a cache key to the command being build. This means that the
// command response can be cached and reused for the execution of the same command
// with the same key.
func (b *CommandBuilder) CacheKey(cacheKey string) *CommandBuilder {
	b.cacheKey = cacheKey
	return b
}

// Build builds a command with all configured parameters.
func (b *CommandBuilder) Build() *Command {
	cmd := &Command{
		name:     b.name,
		group:    b.group,
		run:      b.run,
		cacheKey: b.cacheKey,
		fallback: b.fallback,
	}
	if b.fallback == nil {
		// If no fallback is configured use a default fallback returning an error.
		cmd.fallback = func(ctx context.Context) (interface{}, error) {
			return nil, FallbackNotImplemented
		}
	}
	return cmd
}
