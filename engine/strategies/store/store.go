package store

import (
	"context"
	"sync"

	"github.com/gebv/acca/ffsm"
)

var store = struct {
	s map[StrategyName]Strategy
	m sync.RWMutex
}{
	s: make(map[StrategyName]Strategy),
}

type Strategy interface {
	Dispatch(ctx context.Context, state ffsm.State, payload ffsm.Payload) error
}

func Reg(strName StrategyName, s Strategy) {
	store.m.Lock()
	defer store.m.Unlock()
	store.s[strName] = s
}

func Get(strName StrategyName) Strategy {
	store.m.RLock()
	defer store.m.RUnlock()
	return store.s[strName]
}

type StrategyName string

func (s StrategyName) String() string {
	return string(s)
}
