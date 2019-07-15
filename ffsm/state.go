package ffsm

const AnyState State = "*"
const UnknownState State = ""

// State name of state.
type State string

// Match return true for equal state.
func (s State) Match(in State) bool {
	return s == in
}

func (s State) String() string {
	return string(s)
}

// Set to set new value.
func (s *State) Set(newState State) {
	*s = newState
}
