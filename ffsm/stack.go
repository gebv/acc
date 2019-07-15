package ffsm

// Stack store of actions of event.
type Stack map[StackKey][]ActionLayer

type StackKey struct {
	Src State
	Dst State
}

// Add registration action.
func (r Stack) Add(src State, dst State, p Procedure, name string) {
	if r == nil {
		panic("Stack is empty")
	}

	e := StackKey{Src: src, Dst: dst}
	if r[e] == nil {
		r[e] = []ActionLayer{}
	}
	r[e] = append(r[e], ActionLayer{Func: p, Name: name})
}

// Get return actions of event.
func (r Stack) Get(src, dst State) []ActionLayer {
	if r == nil {
		panic("Stack is empty")
	}
	return r[StackKey{Src: src, Dst: dst}]
}

type ActionLayer struct {
	// Fund procedure function.
	Func Procedure
	// Name action name.
	Name string
}
