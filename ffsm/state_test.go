package ffsm

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestState_Set(t *testing.T) {
	t.Run("Var", func(t *testing.T) {
		foo := State("foo")
		foo.Set(State("bar"))
		assert.Equal(t, State("bar"), foo)
	})
	t.Run("StructEmpty", func(t *testing.T) {
		type object struct {
			S State
		}
		obj := object{}
		obj.S.Set(State("foo"))
		assert.Equal(t, State("foo"), obj.S)
	})
	t.Run("Struct", func(t *testing.T) {
		type object struct {
			S State
		}
		obj := object{S: State("bar")}
		obj.S.Set(State("foo"))
		assert.Equal(t, State("foo"), obj.S)
	})
	t.Run("StructPtr", func(t *testing.T) {
		type object struct {
			S State
		}
		obj := &object{S: State("bar")}
		obj.S.Set(State("foo"))
		assert.Equal(t, State("foo"), obj.S)
	})
}
