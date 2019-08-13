package ffsm

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestChannel(t *testing.T) {
	t.Run("ClosedChannel", func(t *testing.T) {
		ch := make(Channel)
		close(ch)

		wg := sync.WaitGroup{}
		wg.Add(1)
		go func() {
			defer wg.Done()

			select {
			case <-ch:
			case <-time.After(time.Millisecond * 10):
				return
			}
		}()

		t.Run("Send", func(t *testing.T) {
			assert.False(t, ch.Send("abc"))
		})
		wg.Wait()
	})

	t.Run("ClosedChannel", func(t *testing.T) {
		ch := make(Channel)
		ch.Close()

		wg := sync.WaitGroup{}
		wg.Add(1)
		go func() {
			defer wg.Done()

			select {
			case <-ch:
			case <-time.After(time.Millisecond * 10):
				return
			}
		}()

		t.Run("Send", func(t *testing.T) {
			assert.False(t, ch.Send("abc"))
		})
		wg.Wait()
	})

	t.Run("Successfull", func(t *testing.T) {
		ch := make(Channel)
		msg := "abc"

		wg := sync.WaitGroup{}
		wg.Add(1)
		go func() {
			defer wg.Done()

			var got AbstractMessageOfChannel
			select {
			case got = <-ch:
			case <-time.After(time.Millisecond * 10):
			}
			assert.Equal(t, got, msg)
		}()

		time.Sleep(time.Millisecond)

		t.Run("Send", func(t *testing.T) {
			assert.True(t, ch.Send(msg))
		})
		wg.Wait()
	})
}

func BenchmarkChannel(b *testing.B) {
	ch := make(Channel)

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		wg := sync.WaitGroup{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-ch
		}()
		time.Sleep(time.Millisecond)

		ch.Send("ok")

		wg.Wait()
	}
	b.ReportAllocs()
}
