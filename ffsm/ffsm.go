package ffsm

import "context"

type Payload interface{}

// Machine machine state.
type Machine interface {
	// Dispatch new state with payload
	Dispatch(ctx context.Context, state State, payload Payload) error

	// ChangeStateTo change state to new value.
	ChangeStateTo(newState State)

	// Channel for feedback.
	Channel() Channel
}

type Procedure func(ctx context.Context, payload Payload) (context.Context, error)

type EntityState interface {
	SetState(s State)
	GetState() *State
}

type wrapEntityState struct {
	State *State
}

func (w *wrapEntityState) SetState(s State) {
	w.State.Set(s)
}

func (w wrapEntityState) GetState() *State {
	return w.State
}

var _ EntityState = (*wrapEntityState)(nil)
