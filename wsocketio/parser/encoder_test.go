package parser

import (
	"bytes"
	"io"
	"reflect"
	"testing"

	engineio "github.com/googollee/go-engine.io"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeWriter struct {
	typ     engineio.FrameType
	current *bytes.Buffer
	types   []engineio.FrameType
	bufs    []*bytes.Buffer
}

func (w *fakeWriter) NextWriter(ft engineio.FrameType) (io.WriteCloser, error) {
	w.current = bytes.NewBuffer(nil)
	w.typ = ft
	return w, nil
}

func (w *fakeWriter) Write(p []byte) (int, error) {
	return w.current.Write(p)
}

func (w *fakeWriter) Close() error {
	w.types = append(w.types, w.typ)
	w.bufs = append(w.bufs, w.current)
	return nil
}

func TestEncoder(t *testing.T) {
	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			should := assert.New(t)
			must := require.New(t)

			w := fakeWriter{}
			encoder := NewEncoder(&w)
			v := test.Var
			if test.Header.Type == Event {
				v = append([]interface{}{test.Event}, test.Var...)
			}
			err := encoder.Encode(test.Header, v)
			must.Nil(err)
			must.Equal(len(test.Datas), len(w.types))
			must.Equal(len(test.Datas), len(w.bufs))
			for i := range w.types {
				if i == 0 {
					should.Equal(engineio.TEXT, w.types[i])
					should.Equal(string(test.Datas[i]), string(w.bufs[i].Bytes()))
					continue
				}
				should.Equal(engineio.BINARY, w.types[i])
				should.Equal(test.Datas[i], w.bufs[i].Bytes())
			}
		})
	}
}

func TestAttachBuffer(t *testing.T) {
	tests := []struct {
		name    string
		data    interface{}
		max     uint64
		binarys [][]byte
	}{
		{"&Buffer", &Buffer{Data: []byte{1, 2}}, 1, [][]byte{[]byte{1, 2}}},
		{"[]interface{}{Buffer}", []interface{}{&Buffer{Data: []byte{1, 2}}}, 1, [][]byte{[]byte{1, 2}}},
		{"[]interface{}{Buffer,Buffer}", []interface{}{
			&Buffer{Data: []byte{1, 2}},
			&Buffer{Data: []byte{3, 4}},
		}, 2, [][]byte{[]byte{1, 2}, []byte{3, 4}}},
		{"[1]interface{}{Buffer}", [...]interface{}{&Buffer{Data: []byte{1, 2}}}, 1, [][]byte{[]byte{1, 2}}},
		{"[2]interface{}{Buffer,Buffer}", [...]interface{}{
			&Buffer{Data: []byte{1, 2}},
			&Buffer{Data: []byte{3, 4}},
		}, 2, [][]byte{[]byte{1, 2}, []byte{3, 4}}},
		{"Struct{Buffer}", struct {
			Data *Buffer
			I    int
		}{
			&Buffer{Data: []byte{1, 2}},
			3,
		}, 1, [][]byte{[]byte{1, 2}}},
		{"map{Buffer}", map[string]interface{}{
			"data": &Buffer{Data: []byte{1, 2}},
			"i":    3,
		}, 1, [][]byte{[]byte{1, 2}}},
	}

	e := Encoder{}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			should := assert.New(t)
			must := require.New(t)
			index := uint64(0)
			b, err := e.attachBuffer(reflect.ValueOf(test.data), &index)
			must.Nil(err)
			should.Equal(test.max, index)
			should.Equal(test.binarys, b)
		})
	}
}
