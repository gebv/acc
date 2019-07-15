package ffsm

import (
	"context"
	"reflect"
	"testing"
	"time"
)

func Test_WaitString(t *testing.T) {
	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	expiredCtx, _ := context.WithDeadline(context.Background(), time.Now())

	timeoutCtxFn := func(d time.Duration) context.Context {
		ctx, _ := context.WithTimeout(context.Background(), d)
		return ctx
	}

	send := func(msg string) (Channel, func()) {
		ch := make(Channel)
		fn := func() {
			ch <- msg
		}
		return ch, fn
	}

	tests := []struct {
		name    string
		ctx     context.Context
		msg     string
		hasSend bool
		want    string
		ok      bool
	}{
		{
			name:    "CanceledCtx - send=true",
			ctx:     canceledCtx,
			msg:     "foo",
			hasSend: true,
			want:    "",
			ok:      false,
		},
		{
			name:    "CanceledCtx - send=false",
			ctx:     canceledCtx,
			msg:     "foo",
			hasSend: false,
			want:    "",
			ok:      false,
		},

		{
			name:    "ExpiredCtx - send=true",
			ctx:     expiredCtx,
			msg:     "foo",
			hasSend: true,
			want:    "",
			ok:      false,
		},
		{
			name:    "ExpiredCtx - send=false",
			ctx:     expiredCtx,
			msg:     "foo",
			hasSend: false,
			want:    "",
			ok:      false,
		},

		{
			name:    "OK - send=true",
			ctx:     context.Background(),
			msg:     "foo",
			hasSend: true,
			want:    "foo",
			ok:      true,
		},

		{
			name:    "ExpireCtx - send=false",
			ctx:     timeoutCtxFn(time.Millisecond * 100),
			msg:     "foo",
			hasSend: false,
			want:    "",
			ok:      false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ch, fn := send(tt.msg)
			got := WaitString(tt.ctx, ch)

			if tt.hasSend {
				go fn()
			}

			msg, ok := <-got

			if ok != tt.ok {
				t.Error("not expected OK, want", tt.ok)
			}

			if msg != tt.want {
				t.Error("not exptected value, want", tt.want)
			}
		})
	}
}

func Test_WaitStruct(t *testing.T) {
	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	expiredCtx, _ := context.WithDeadline(context.Background(), time.Now())

	timeoutCtxFn := func(d time.Duration) context.Context {
		ctx, _ := context.WithTimeout(context.Background(), d)
		return ctx
	}

	send := func(msg ChannelMsgStruct) (Channel, func()) {
		ch := make(Channel)
		fn := func() {
			ch <- msg
		}
		return ch, fn
	}

	tests := []struct {
		name    string
		ctx     context.Context
		msg     ChannelMsgStruct
		hasSend bool
		want    ChannelMsgStruct
		ok      bool
	}{
		{
			name:    "CanceledCtx - send=true",
			ctx:     canceledCtx,
			msg:     ChannelMsgStruct{"foo"},
			hasSend: true,
			want:    ChannelMsgStruct{""},
			ok:      false,
		},
		{
			name:    "CanceledCtx - send=false",
			ctx:     canceledCtx,
			msg:     ChannelMsgStruct{"foo"},
			hasSend: false,
			want:    ChannelMsgStruct{""},
			ok:      false,
		},

		{
			name:    "ExpiredCtx - send=true",
			ctx:     expiredCtx,
			msg:     ChannelMsgStruct{"foo"},
			hasSend: true,
			want:    ChannelMsgStruct{""},
			ok:      false,
		},
		{
			name:    "ExpiredCtx - send=false",
			ctx:     expiredCtx,
			msg:     ChannelMsgStruct{"foo"},
			hasSend: false,
			want:    ChannelMsgStruct{""},
			ok:      false,
		},

		{
			name:    "OK - send=true",
			ctx:     context.Background(),
			msg:     ChannelMsgStruct{"foo"},
			hasSend: true,
			want:    ChannelMsgStruct{"foo"},
			ok:      true,
		},

		{
			name:    "ExpireCtx - send=false",
			ctx:     timeoutCtxFn(time.Millisecond * 100),
			msg:     ChannelMsgStruct{"foo"},
			hasSend: false,
			want:    ChannelMsgStruct{""},
			ok:      false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ch, fn := send(tt.msg)
			got := WaitStruct(tt.ctx, ch)

			if tt.hasSend {
				go fn()
			}

			msg, ok := <-got

			if ok != tt.ok {
				t.Error("not expected OK, want", tt.ok)
			}

			if !reflect.DeepEqual(msg, tt.want) {
				t.Error("not exptected value, want", tt.want)
			}
		})
	}
}

func Test_WaitInterface(t *testing.T) {
	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	expiredCtx, _ := context.WithDeadline(context.Background(), time.Now())

	timeoutCtxFn := func(d time.Duration) context.Context {
		ctx, _ := context.WithTimeout(context.Background(), d)
		return ctx
	}

	send := func(msg Stringer) (Channel, func()) {
		ch := make(Channel)
		fn := func() {
			ch <- msg
		}
		return ch, fn
	}

	tests := []struct {
		name    string
		ctx     context.Context
		msg     Stringer
		hasSend bool
		want    Stringer
		ok      bool
	}{
		{
			name:    "CanceledCtx - send=true",
			ctx:     canceledCtx,
			msg:     ChannelMsgStruct{"foo"},
			hasSend: true,
			want:    nil,
			ok:      false,
		},
		{
			name:    "CanceledCtx - send=false",
			ctx:     canceledCtx,
			msg:     ChannelMsgStruct{"foo"},
			hasSend: false,
			want:    nil,
			ok:      false,
		},

		{
			name:    "ExpiredCtx - send=true",
			ctx:     expiredCtx,
			msg:     ChannelMsgStruct{"foo"},
			hasSend: true,
			want:    nil,
			ok:      false,
		},
		{
			name:    "ExpiredCtx - send=false",
			ctx:     expiredCtx,
			msg:     ChannelMsgStruct{"foo"},
			hasSend: false,
			want:    nil,
			ok:      false,
		},

		{
			name:    "OK - send=true",
			ctx:     context.Background(),
			msg:     ChannelMsgStruct{"foo"},
			hasSend: true,
			want:    ChannelMsgStruct{"foo"},
			ok:      true,
		},

		{
			name:    "ExpireCtx - send=false",
			ctx:     timeoutCtxFn(time.Millisecond * 100),
			msg:     ChannelMsgStruct{"foo"},
			hasSend: false,
			want:    nil,
			ok:      false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ch, fn := send(tt.msg)
			got := WaitInterface(tt.ctx, ch)

			if tt.hasSend {
				go fn()
			}

			msg, ok := <-got

			if ok != tt.ok {
				t.Error("not expected OK, want", tt.ok)
			}

			if msg == nil && tt.want != nil {
				t.Error("got empty value, want", tt.want.String())
			}

			if tt.want == nil && msg != nil {
				t.Error("want empty value, got", tt.want.String())
			}

			if tt.want != nil && msg != nil {
				if msg.String() != tt.want.String() {
					t.Error("not exptected value, want", tt.want.String())
				}
			}

		})
	}
}
