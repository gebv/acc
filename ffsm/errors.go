package ffsm

import "errors"

var (
	ErrNotInitalState   = errors.New("Is not set initial value of state")
	ErrNoTransition     = errors.New("No transition")
	ErrCtxCanceled      = errors.New("Context canceled")
	ErrNotRegTransition = errors.New("Not registred transition")
)

type DispatchError struct {
	ActionName        string
	SrcState          State
	DstState          State
	Err               error
	IsPanic           bool
	PanicStackRuntime string
}

func (e DispatchError) Error() string {
	if e.Err == nil {
		return ""
	}
	return e.Err.Error()
}
