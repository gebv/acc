package ffsm

import (
	"context"
	"fmt"
	"runtime/debug"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

func MachineFrom(wf Stack, obj interface{}) *MachineState {
	mActionDur := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:      "ffsm_action_duration_ms",
			Help:      "Duration of a single action.",
			Subsystem: "ffsm",
			Buckets: []float64{
				float64(time.Millisecond * 100),
				float64(time.Millisecond * 300),
				float64(time.Millisecond * 700),
				float64(time.Millisecond * 1000),
			},
		},
		[]string{"ffsm"},
	)

	mTotalDur := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:      "ffsm_total_duration_ms",
			Help:      "Duration of a total dispatch.",
			Subsystem: "ffsm",
			Buckets: []float64{
				float64(time.Millisecond * 500),
				float64(time.Millisecond * 1000),
				float64(time.Millisecond * 2000),
				float64(time.Millisecond * 5000),
			},
		},
		[]string{"ffsm"},
	)

	mNumReqAct := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name:      "ffsm_exec_action_total",
			Help:      "Number of execute to actions.",
			Subsystem: "ffsm",
		},
		[]string{"ffsm"},
	)

	mTotalReqs := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name:      "ffsm_dispatch_total",
			Help:      "Number of dispatch.",
			Subsystem: "ffsm",
		},
		[]string{"ffsm"},
	)

	switch obj := obj.(type) {
	case nil:
		return &MachineState{
			wf:              wf,
			s:               &wrapEntityState{State: new(State)},
			c:               make(Channel),
			mActionDuration: mActionDur,
			mTotalDuration:  mTotalDur,
			mActionRequest:  mNumReqAct,
			mTotalRequest:   mTotalReqs,
		}
	case *State:
		return &MachineState{
			wf:              wf,
			s:               &wrapEntityState{State: obj},
			c:               make(Channel),
			mActionDuration: mActionDur,
			mTotalDuration:  mTotalDur,
			mActionRequest:  mNumReqAct,
			mTotalRequest:   mTotalReqs,
		}
	case EntityState:
		return &MachineState{
			wf:              wf,
			s:               obj,
			c:               make(Channel),
			mActionDuration: mActionDur,
			mTotalDuration:  mTotalDur,
			mActionRequest:  mNumReqAct,
			mTotalRequest:   mTotalReqs,
		}
	default:
		panic(fmt.Sprintf("Not supported type %T", obj))
	}
	return nil
}

type MachineState struct {
	s  EntityState
	wf Stack
	c  Channel

	mActionDuration *prometheus.HistogramVec
	mTotalDuration  *prometheus.HistogramVec
	mActionRequest  *prometheus.CounterVec
	mTotalRequest   *prometheus.CounterVec

	stateManualChanged bool

	name string
}

func (m *MachineState) SetName(name string) {
	m.name = name
}

// Dispatch attempt to change state from workflow.
func (m *MachineState) Dispatch(ctx context.Context, nextState State, payload Payload) (err error) {
	start := time.Now()

	if m.s.GetState().Match(UnknownState) {
		return ErrNotInitalState
	}
	if m.s.GetState().Match(nextState) {
		return ErrNoTransition
	}

	if ctx.Err() != nil {
		return ErrCtxCanceled
	}

	currState := *m.s.GetState()
	actions := m.wf.Get(currState, nextState)
	if actions == nil {
		return ErrNotRegTransition
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// local workspace
	var fm = MachineFrom(m.wf, nil)
	fm.c = m.c
	fm.name = m.name
	fm.mActionDuration = m.mActionDuration
	fm.mTotalDuration = m.mTotalDuration
	fm.mActionRequest = m.mActionRequest
	fm.mTotalRequest = m.mTotalRequest
	fm.ChangeStateTo(m.CurrentState())
	fm.stateManualChanged = false // to detect change in procedures

	defer m.Channel().Close()

	done := make(chan error, 1)
	go func(ctx context.Context) {
		var err error

		for _, act := range actions {
			actStart := time.Now()
			// if act.Func == nil {
			// 	// skip empty procedure
			// 	continue
			// }
			var actionDoneCh = make(chan error)
			go func() {
				defer func() {
					if r := recover(); r != nil {
						actionDoneCh <- DispatchError{
							ActionName:        act.Name,
							IsPanic:           true,
							SrcState:          currState,
							DstState:          nextState,
							Err:               fmt.Errorf("Recover panic: %v", r),
							PanicStackRuntime: string(debug.Stack()),
						}
						return
					}

					if err != nil {
						actionDoneCh <- err
						return
					}

					close(actionDoneCh)
				}()

				actionCtx := setMachineAndStates(ctx, fm, currState, nextState)
				if actionCtx, err = act.Func(actionCtx, payload); err != nil {
					actionDoneCh <- DispatchError{
						ActionName: act.Name,
						SrcState:   currState,
						DstState:   nextState,
						Err:        err,
					}
					return
				}
			}()

			select {
			case <-ctx.Done():
				return
			case err, ok := <-actionDoneCh:
				if ok && err != nil {
					done <- err
					return
				}
			}

			// TODO: to format name of action
			m.mActionDuration.WithLabelValues(act.Name).Observe(float64(time.Since(actStart).Nanoseconds() / int64(time.Millisecond)))
			m.mActionRequest.WithLabelValues(act.Name).Inc()
		}
		close(done)
	}(ctx)

	select {
	case <-ctx.Done():
		cancel()
		// <-done // TODO: wait for actions to return?
		return ErrCtxCanceled
	case err, ok := <-done:
		if ok && err != nil {
			return err
		}
	}

	// new state
	if fm.stateManualChanged {
		m.ChangeStateTo(fm.CurrentState())
	} else {
		m.ChangeStateTo(nextState)
	}

	m.mActionDuration.WithLabelValues(m.name).Observe(float64(time.Since(start).Nanoseconds() / int64(time.Millisecond)))
	m.mTotalRequest.WithLabelValues(m.name).Inc()

	return nil
}

func (a *MachineState) Describe(ch chan<- *prometheus.Desc) {
	a.mActionDuration.Describe(ch)
}

func (a *MachineState) Collect(ch chan<- prometheus.Metric) {
	a.mActionDuration.Collect(ch)
}

// Channel return channel for feedback from procedures.
func (m *MachineState) Channel() Channel {
	return m.c
}

// CurrentState return current state.
func (m *MachineState) CurrentState() State {
	return *m.s.GetState()
}

// ChangeStateTo change state to new value.
func (m *MachineState) ChangeStateTo(newState State) {
	m.s.SetState(newState)
	m.stateManualChanged = true
}

var _ Machine = (*MachineState)(nil)
var _ prometheus.Collector = (*MachineState)(nil)
