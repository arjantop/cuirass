package cuirass

import "errors"

var FallbackNotImplemented = errors.New("Fallback not implemented")

type Command interface {
	Name() string
	Run(result interface{}) error
	Fallback(result interface{}) error
}
