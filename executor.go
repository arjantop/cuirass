package cuirass

import "errors"

var UnknownPanic = errors.New("Unknown panic")

type Executor struct {
}

func NewExecutor() *Executor {
	return &Executor{}
}

func (e *Executor) Exec(cmd Command, result interface{}) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = cmd.Fallback(result)
			if err != nil {
				switch x := r.(type) {
				case error:
					err = x
				case string:
					err = errors.New(x)
				default:
					err = UnknownPanic
				}
			}
		}
	}()
	err = cmd.Run(result)
	if err != nil {
		panic(err)
	}
	return nil
}
