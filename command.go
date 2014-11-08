package cuirass

import (
	"errors"

	"code.google.com/p/go.net/context"
)

var FallbackNotImplemented = errors.New("Fallback not implemented")

type Command interface {
	Name() string
	Run(ctx context.Context, result interface{}) error
	Fallback(ctx context.Context, result interface{}) error
}
