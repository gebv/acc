package ffsm

import (
	"context"
)

func WaitString(ctx context.Context, ch Channel) chan string {
	done := make(chan string, 1)
	if ctx.Err() != nil {
		close(done)
		return done
	}

	go func() {
		select {
		case <-ctx.Done():
			close(done)
			return
		case msg := <-ch:
			switch msg := msg.(type) {
			case string:
				done <- msg
			default:
				close(done)
			}
		}
	}()
	return done
}

type ChannelMsgStruct struct {
	Foo string
}

func (s ChannelMsgStruct) String() string {
	return s.Foo
}

func WaitStruct(ctx context.Context, ch Channel) chan ChannelMsgStruct {
	done := make(chan ChannelMsgStruct, 1)
	if ctx.Err() != nil {
		close(done)
		return done
	}

	go func() {
		select {
		case <-ctx.Done():
			close(done)
			return
		case msg := <-ch:
			switch msg := msg.(type) {
			case ChannelMsgStruct:
				done <- msg
			default:
				close(done)
			}
		}
	}()
	return done
}

type Stringer interface {
	String() string
}

func WaitInterface(ctx context.Context, ch Channel) chan Stringer {
	done := make(chan Stringer, 1)
	if ctx.Err() != nil {
		close(done)
		return done
	}

	go func() {
		select {
		case <-ctx.Done():
			close(done)
			return
		case msg := <-ch:
			switch msg := msg.(type) {
			case Stringer:
				done <- msg
			default:
				close(done)
			}
		}
	}()
	return done
}
