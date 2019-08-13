package ffsm

import (
	"sync"
	"testing"
)

func TestStack_AddManual(t *testing.T) {
	r := make(Stack)
	name := "a->b"
	r.Add(State("a"), State("b"), nil, name)
	if len(r) == 0 {
		t.Fatal("empty Stack")
	}
	if len(r[StackKey{Src: State("a"), Dst: State("b")}]) == 0 {
		t.Fatal("empty actions by event")
	}
	if len(r[StackKey{Src: State("not exists"), Dst: State("not exists")}]) != 0 {
		t.Fatal("want empty actions by not exists event")
	}
	if r[StackKey{Src: State("a"), Dst: State("b")}][0].Name != name {
		t.Fatal("not equal name action")
	}
}

func TestAsync_Get(t *testing.T) {
	r := make(Stack)
	r.Add(State("a"), State("b"), nil, "a->b#1")
	r.Add(State("a"), State("b"), nil, "a->b#2")
	r.Add(State("b"), State("c"), nil, "b->c")
	r.Add(State("c"), State("a"), nil, "c->a")
	wg := sync.WaitGroup{}
	for n := 0; n < 1000; n++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			actions := r.Get(State("a"), State("b"))
			if len(actions) != 2 {
				t.Fatal("not expected len actions")
			}
			for _, action := range actions {
				if action.Name[:4] != "a->b" {
					t.Fatal("not expected action name")
				}
			}
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			actions := r.Get(State("not exists"), State("not exists"))
			if len(actions) != 0 {
				t.Fatal("not expected len actions")
			}
		}()
	}
	wg.Wait()
}
