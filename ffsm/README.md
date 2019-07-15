# FFSM

Working code [see more in tests](machine_state_test.go)

## Concept

The following is concept code.

```golang
// custom service
s := &doorManager{}

// workflow or stateflow
wf := make(Stack)
wf.Add(CloseDoor, OpenDoor, s.AccessOnlyBob, "Bob only have access to door.")
wf.Add(CloseDoor, TokTokDoor, s.TokTokDoor, "Bob only have access to door. (example redirect process via sub-dispatch).")
wf.Add(OpenDoor, CloseDoor, s.Empty, "Anyone can close the door.")

// initial state
initalState := CloseDoor
fsm := NewFSM(wf, &initalState)
err := fsm.Distapch(ctx, OpenDoor, Payload)
if err != nil {
  // fail
}
fsm.CurrentState() // finite state - initalState == fsm.CurrentState()
```
