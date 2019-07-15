package ffsm

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func Test_Simple2(t *testing.T) {
	// TODO: more tests for DispatchError

	s := &doorTestService{}

	wfOnlyBob := make(Stack)
	wfOnlyBob.Add(CloseDoor, OpenDoor, s.AccessOnlyBob, "close->open")
	wfOnlyBob.Add(OpenDoor, CloseDoor, s.Empty, "open->close")

	wfOnlyBobWithFeedback := make(Stack)
	wfOnlyBobWithFeedback.Add(CloseDoor, OpenDoor, s.AccessOnlyBob, "close->open")
	wfOnlyBobWithFeedback.Add(CloseDoor, OpenDoor, s.SendToChannelFoo, "close->open")
	wfOnlyBobWithFeedback.Add(OpenDoor, CloseDoor, s.Empty, "open->close")

	wfOnlyBobWithInvalidProcedure := make(Stack)
	wfOnlyBobWithInvalidProcedure.Add(CloseDoor, OpenDoor, s.AccessOnlyBob, "close->open")
	wfOnlyBobWithInvalidProcedure.Add(CloseDoor, OpenDoor, nil, "close->open")
	wfOnlyBobWithInvalidProcedure.Add(OpenDoor, CloseDoor, s.Empty, "open->close")

	wfOnlyBobWithPanic := make(Stack)
	wfOnlyBobWithPanic.Add(CloseDoor, OpenDoor, s.AccessOnlyBob, "close->open")
	wfOnlyBobWithPanic.Add(CloseDoor, OpenDoor, s.Panic, "close->open")
	wfOnlyBobWithPanic.Add(OpenDoor, CloseDoor, s.Empty, "open->close")

	wfForceClose := make(Stack)
	wfForceClose.Add(CloseDoor, OpenDoor, s.AccessOnlyBob, "close->open")
	wfForceClose.Add(CloseDoor, OpenDoor, s.ForceCloseDoor, "close->open")

	wfSleep10min := make(Stack)
	wfSleep10min.Add(CloseDoor, OpenDoor, s.Sleep10min, "close->open")

	wfCustomErr := make(Stack)
	wfCustomErr.Add(CloseDoor, OpenDoor, s.ErrorFunction, "close->open")
	wfCustomErr.Add(OpenDoor, CloseDoor, s.Empty, "open->close")

	wfOpenTokTok := make(Stack)
	wfOpenTokTok.Add(CloseDoor, OpenDoor, s.AccessOnlyBob, "close->open")
	wfOpenTokTok.Add(OpenDoor, CloseDoor, s.Empty, "open->close")
	wfOpenTokTok.Add(CloseDoor, TokTokDoor, s.TokTokDoor, "close->toktok")

	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	expiredCtx, cancel := context.WithDeadline(context.Background(), time.Now())
	defer cancel()

	defStateObj := func(initState ...State) EntityState {
		es := &wrapEntityState{State: new(State)}
		if initState != nil {
			es.SetState(initState[0])
		}
		return es
	}

	var (
		FooState = State("foo")
	)

	tests := []struct {
		name        string
		wf          Stack
		stateObj    EntityState
		ctx         context.Context
		initState   State
		finiteState State
		err         error
		hasFeedback bool
		feedback    interface{}
		payload     interface{}
	}{
		{
			stateObj:    defStateObj(),
			name:        "Empty",
			err:         ErrNotInitalState,
			finiteState: UnknownState,
		},
		{
			stateObj:    defStateObj(FooState),
			name:        "CanceledContext",
			ctx:         canceledCtx,
			err:         ErrCtxCanceled,
			finiteState: FooState,
		},
		{
			stateObj:    defStateObj(FooState),
			name:        "CanceledContext_ExpiredCtx",
			ctx:         expiredCtx,
			err:         ErrCtxCanceled,
			finiteState: FooState,
		},
		{
			stateObj:    defStateObj(CloseDoor),
			wf:          wfOnlyBob,
			name:        "ForbiddenForAlisa",
			ctx:         SetNameFromCtx(context.Background(), "alisa"),
			err:         errors.New("Access denied"),
			initState:   OpenDoor,
			finiteState: CloseDoor,
		},
		{
			stateObj:    defStateObj(CloseDoor),
			wf:          wfOnlyBob,
			name:        "OpenedForBob",
			ctx:         SetNameFromCtx(context.Background(), "bob"),
			err:         nil,
			initState:   OpenDoor,
			finiteState: OpenDoor,
		},
		{
			stateObj:    defStateObj(CloseDoor),
			wf:          wfOnlyBobWithPanic,
			name:        "PanicOpenedForBob_ProcedurePanic",
			ctx:         SetNameFromCtx(context.Background(), "bob"),
			err:         errors.New("Recover panic: panic"),
			initState:   OpenDoor,
			finiteState: CloseDoor,
		},
		{
			stateObj:    defStateObj(CloseDoor),
			wf:          wfOnlyBobWithPanic,
			name:        "ForbiddenForAlisa_ProcedurePanic",
			ctx:         SetNameFromCtx(context.Background(), "alise"),
			err:         errors.New("Access denied"),
			initState:   OpenDoor,
			finiteState: CloseDoor,
		},

		{
			stateObj:    defStateObj(CloseDoor),
			wf:          wfOnlyBobWithInvalidProcedure,
			name:        "PanicOpenedForBob_InvalidProcedure",
			ctx:         SetNameFromCtx(context.Background(), "bob"),
			err:         errors.New("Recover panic: runtime error: invalid memory address or nil pointer dereference"),
			initState:   OpenDoor,
			finiteState: CloseDoor,
		},
		{
			stateObj:    defStateObj(CloseDoor),
			wf:          wfOnlyBobWithInvalidProcedure,
			name:        "ForbiddenForAlisa_InvalidProcedure",
			ctx:         SetNameFromCtx(context.Background(), "alise"),
			err:         errors.New("Access denied"),
			initState:   OpenDoor,
			finiteState: CloseDoor,
		},

		{
			stateObj:    defStateObj(CloseDoor),
			wf:          wfForceClose,
			name:        "ForbiddenForAlisa_ForceClose",
			ctx:         SetNameFromCtx(context.Background(), "alise"),
			err:         errors.New("Access denied"),
			initState:   OpenDoor,
			finiteState: CloseDoor,
		},
		{
			stateObj:    defStateObj(CloseDoor),
			wf:          wfForceClose,
			name:        "SuccessForBob_ForceClose",
			ctx:         SetNameFromCtx(context.Background(), "bob"),
			err:         nil,
			initState:   OpenDoor,
			finiteState: CloseDoor,
		},

		{
			stateObj: defStateObj(CloseDoor),
			wf:       wfSleep10min,
			name:     "TimeoutCtx",
			ctx: func() context.Context {
				ctx, _ := context.WithTimeout(context.Background(), time.Millisecond*100)
				return ctx
			}(),
			err:         ErrCtxCanceled,
			initState:   OpenDoor,
			finiteState: CloseDoor,
		},

		{
			stateObj:    defStateObj(CloseDoor),
			wf:          wfOpenTokTok,
			name:        "OpenedForBob_ViaTokTok",
			ctx:         SetNameFromCtx(context.Background(), "bob"),
			err:         nil,
			initState:   TokTokDoor,
			finiteState: OpenDoor,
		},
		{
			stateObj:    defStateObj(CloseDoor),
			wf:          wfOpenTokTok,
			name:        "ForbiddenForAlise_ViaTokTok",
			ctx:         SetNameFromCtx(context.Background(), "alise"),
			err:         errors.New("Access denied"),
			initState:   TokTokDoor,
			finiteState: CloseDoor,
		},
		{
			stateObj:    defStateObj(CloseDoor),
			wf:          wfCustomErr,
			name:        "CustomErrorFromProcedure",
			ctx:         context.Background(),
			err:         errors.New("failed procedure"),
			initState:   OpenDoor,
			finiteState: CloseDoor,
		},
		{
			stateObj:    &Door{State: CloseDoor},
			wf:          wfOnlyBob,
			name:        "Door_Forbidden",
			ctx:         context.Background(),
			err:         errors.New("Access denied"),
			initState:   OpenDoor,
			finiteState: CloseDoor,
		},
		{
			stateObj:    &Door{State: CloseDoor},
			wf:          wfOnlyBob,
			name:        "Door_OpenedForBob",
			ctx:         SetNameFromCtx(context.Background(), "bob"),
			err:         nil,
			initState:   OpenDoor,
			finiteState: OpenDoor,
		},

		{
			stateObj:    &Door{State: CloseDoor},
			wf:          wfOnlyBobWithFeedback,
			name:        "Door_OpenedForBob_WithFeedback",
			ctx:         SetNameFromCtx(context.Background(), "bob"),
			err:         nil,
			initState:   OpenDoor,
			finiteState: OpenDoor,
			hasFeedback: true,
			feedback:    "foo",
		},

		{
			stateObj:    &Door{State: CloseDoor},
			wf:          wfOnlyBobWithFeedback,
			name:        "Door_ForbiddenForAlise_WithFeedback",
			ctx:         SetNameFromCtx(context.Background(), "alise"),
			err:         errors.New("Access denied"),
			initState:   OpenDoor,
			finiteState: CloseDoor,
			hasFeedback: true,
			feedback:    nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := MachineFrom(tt.wf, tt.stateObj)

			wg := sync.WaitGroup{}
			wg.Add(1)
			go func() {
				defer wg.Done()

				var gotMsg interface{}
				select {
				case gotMsg = <-m.Channel():
				case <-time.After(time.Millisecond * 100):
				}
				assert.Equal(t, tt.feedback, gotMsg)
			}()

			err := m.Dispatch(tt.ctx, tt.initState, tt.payload)

			if tt.err != nil {
				assert.Error(t, err)
				assert.EqualError(t, err, tt.err.Error())
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.finiteState, m.CurrentState())
			assert.Equal(t, tt.finiteState, *tt.stateObj.GetState())

			wg.Wait()
		})
	}
}

type Door struct {
	DoorID int64
	State  State
}

func (d *Door) SetState(s State) {
	d.State.Set(s)
}

func (d *Door) GetState() *State {
	return &d.State
}

var _ EntityState = (*Door)(nil)

func BenchmarkSimple1(b *testing.B) {
	s := &doorTestService{}
	wf := make(Stack)
	wf.Add(CloseDoor, OpenDoor, s.AccessOnlyBob, "close->open")
	wf.Add(OpenDoor, CloseDoor, s.Empty, "open->close")
	ctx := SetNameFromCtx(context.Background(), "bob")
	state := new(State)
	m := MachineFrom(wf, state)
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		m.wf = wf
		m.ChangeStateTo(CloseDoor)
		err := m.Dispatch(ctx, OpenDoor, nil)
		if err != nil {
			b.Fatal("dispatch with err", err)
		}
		if m.CurrentState() != OpenDoor {
			b.Fatal("want OpenDoor")
		}
	}
	b.ReportAllocs()
}

const (
	OpenDoor   State = "open"
	TokTokDoor State = "toktok"
	CloseDoor  State = "close"
)

type doorTestService struct {
}

func (s *doorTestService) ErrorFunction(ctx context.Context, payload Payload) (context.Context, error) {
	return ctx, errors.New("failed procedure")
}

func (s *doorTestService) Empty(ctx context.Context, payload Payload) (context.Context, error) {
	return ctx, nil
}

func (s *doorTestService) TokTokDoor(ctx context.Context, payload Payload) (context.Context, error) {
	err := GetMachine(ctx).Dispatch(ctx, OpenDoor, payload)
	return ctx, err
}

func (s *doorTestService) Sleep10min(ctx context.Context, payload Payload) (context.Context, error) {
	time.Sleep(time.Minute * 10)
	return ctx, nil
}

func (s *doorTestService) Panic(ctx context.Context, payload Payload) (context.Context, error) {
	panic("panic")
}

func (s *doorTestService) SendToChannelFoo(ctx context.Context, payload Payload) (context.Context, error) {
	GetMachine(ctx).Channel().Send("foo")
	return ctx, nil
}

func (s *doorTestService) ForceCloseDoor(ctx context.Context, payload Payload) (context.Context, error) {
	GetMachine(ctx).ChangeStateTo(CloseDoor)
	return ctx, nil
}

func (s *doorTestService) AccessOnlyBob(ctx context.Context, payload Payload) (context.Context, error) {
	gotName := GetNameFromCtx(ctx)
	if gotName != "bob" {
		return ctx, errors.New("Access denied")
	}
	return ctx, nil
}

func SetNameFromCtx(ctx context.Context, name string) context.Context {
	return SetStringFromCtx(ctx, "__name", name)
}

func GetNameFromCtx(ctx context.Context) string {
	return StringFromCtx(ctx, "__name")
}

func SetStringFromCtx(ctx context.Context, key string, value interface{}) context.Context {
	return context.WithValue(ctx, key, value)
}

func StringFromCtx(ctx context.Context, key string) string {
	v, ok := ctx.Value(key).(string)
	if !ok {
		return ""
	}
	return v
}
