package wsocketio

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEventFunc(t *testing.T) {
	tests := []struct {
		f        interface{}
		ok       bool
		argTypes []interface{}
	}{
		{1, false, []interface{}{}},
		{func() {}, false, []interface{}{}},
		{func(int) {}, false, []interface{}{}},
		{func() error { return nil }, false, []interface{}{}},

		{func(Conn) {}, true, []interface{}{}},
		{func(Conn, int) {}, true, []interface{}{1}},
		{func(Conn, int) error { return nil }, true, []interface{}{1}},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%#v", test.argTypes), func(t *testing.T) {
			should := assert.New(t)
			must := require.New(t)
			defer func() {
				r := recover()
				must.Equal(test.ok, r == nil)
			}()

			h := newEventFunc(test.f)
			must.Equal(len(test.argTypes), len(h.argTypes))
			for i := range h.argTypes {
				should.Equal(reflect.TypeOf(test.argTypes[i]), h.argTypes[i])
			}
		})
	}
}

func TestNewAckFunc(t *testing.T) {
	tests := []struct {
		f        interface{}
		ok       bool
		argTypes []interface{}
	}{
		{1, false, []interface{}{}},

		{func() {}, true, []interface{}{}},
		{func(int) {}, true, []interface{}{1}},
		{func(int) error { return nil }, true, []interface{}{1}},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%#v", test.argTypes), func(t *testing.T) {
			should := assert.New(t)
			must := require.New(t)
			defer func() {
				r := recover()
				must.Equal(test.ok, r == nil)
			}()

			h := newAckFunc(test.f)
			must.Equal(len(test.argTypes), len(h.argTypes))
			for i := range h.argTypes {
				should.Equal(reflect.TypeOf(test.argTypes[i]), h.argTypes[i])
			}
		})
	}
}

func TestHandlerCall(t *testing.T) {
	tests := []struct {
		f    interface{}
		args []interface{}
		ok   bool
		rets []interface{}
	}{
		{func() {}, []interface{}{1}, false, nil},

		{func() {}, nil, true, nil},
		{func(int) {}, []interface{}{1}, true, nil},
		{func() int { return 1 }, nil, true, []interface{}{1}},
		{func(int) int { return 1 }, []interface{}{1}, true, []interface{}{1}},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%#v", test.f), func(t *testing.T) {
			should := assert.New(t)
			must := require.New(t)

			h := newAckFunc(test.f)
			args := make([]reflect.Value, len(test.args))
			for i := range args {
				args[i] = reflect.ValueOf(test.args[i])
			}
			retV, err := h.Call(args)
			must.Equal(test.ok, err == nil)
			if len(retV) == len(test.rets) && len(test.rets) == 0 {
				return
			}
			rets := make([]interface{}, len(retV))
			for i := range rets {
				rets[i] = retV[i].Interface()
			}
			should.Equal(test.rets, rets)
		})
	}
}
