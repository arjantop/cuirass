package cuirass

import (
	"errors"

	"code.google.com/p/go.net/context"
)

// FallbackNotImplemented is the error returned by Cimmand.Fallback when no fallback
// function is configured.
var FallbackNotImplemented = errors.New("Fallback not implemented")

// A CommandFunc is a function that contains the primary or fallback logic
// for the command.
type CommandFunc func(ctx context.Context, result interface{}) error

// Command is a wrapper for a code that requires latency and fault tolerance
// (typically service call over the network).
type Command struct {
	name          string
	run, fallback CommandFunc
}

// Name returns the name of the command.
func (c *Command) Name() string {
	return c.name
}

// Run executes a primary function to fetch a result.
func (c *Command) Run(ctx context.Context, r interface{}) error {
	return c.run(ctx, r)
}

// Fallback executes the fallback logic when primary function fails.
func (c *Command) Fallback(ctx context.Context, r interface{}) error {
	return c.fallback(ctx, r)
}

// CommandBuilder is a helper used for constructing new Commands.
type CommandBuilder struct {
	name          string
	run, fallback CommandFunc
}

// NewCommand constructs a new CommandBuilder with minimal required command
// implementation (name and primary function).
func NewCommand(name string, run CommandFunc) *CommandBuilder {
	return &CommandBuilder{
		name: name,
		run:  run,
	}
}

// Fallback adds a fallback function to the command being built.
func (b *CommandBuilder) Fallback(fallback CommandFunc) *CommandBuilder {
	b.fallback = fallback
	return b
}

// Build builds a command with all configured parameters.
func (b *CommandBuilder) Build() *Command {
	cmd := &Command{
		name: b.name,
		run:  b.run,
	}
	if b.fallback == nil {
		// If no fallback is configured use a default fallback returning an error.
		cmd.fallback = func(ctx context.Context, r interface{}) error {
			return FallbackNotImplemented
		}
	} else {
		cmd.fallback = b.fallback
	}
	return cmd
}
