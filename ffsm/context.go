package ffsm

import "context"

type ctxKey uint

const (
	machineCtxKey        ctxKey = 1
	sourceStateCtxKey    ctxKey = 2
	distanateStateCtxKey ctxKey = 3

	requestIDCtxKey ctxKey = 4
)

func setMachineAndStates(ctx context.Context, m Machine, srcState, dstState State) context.Context {
	ctx = context.WithValue(ctx, machineCtxKey, m)
	ctx = context.WithValue(ctx, sourceStateCtxKey, srcState)
	return context.WithValue(ctx, distanateStateCtxKey, dstState)
}

func SetRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDCtxKey, requestID)
}

func GetMachine(ctx context.Context) Machine {
	return ctx.Value(machineCtxKey).(Machine)
}

func GetSrcState(ctx context.Context) State {
	return ctx.Value(sourceStateCtxKey).(State)
}

func GetDstState(ctx context.Context) State {
	return ctx.Value(distanateStateCtxKey).(State)
}
