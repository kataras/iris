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

type fakeReader struct {
	datas [][]byte
	index int
	buf   *bytes.Buffer
}

func (r *fakeReader) NextReader() (engineio.FrameType, io.ReadCloser, error) {
	if r.index >= len(r.datas) {
		return 0, nil, io.EOF
	}
	r.buf = bytes.NewBuffer(r.datas[r.index])
	ft := engineio.BINARY
	if r.index == 0 {
		ft = engineio.TEXT
	}
	return ft, r, nil
}

func (r *fakeReader) Read(p []byte) (int, error) {
	return r.buf.Read(p)
}

func (r *fakeReader) Close() error {
	r.index++
	return nil
}

func TestDecoder(t *testing.T) {
	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			should := assert.New(t)
			must := require.New(t)

			r := fakeReader{datas: test.Datas}
			decoder := NewDecoder(&r)
			defer func() {
				decoder.DiscardLast()
				decoder.Close()
			}()
			var header Header
			var event string
			err := decoder.DecodeHeader(&header, &event)
			must.Nil(err, "decode header error: %s", err)
			should.Equal(test.Header, header)
			should.Equal(test.Event, event)
			types := make([]reflect.Type, len(test.Var))
			for i := range types {
				types[i] = reflect.TypeOf(test.Var[i])
			}
			ret, err := decoder.DecodeArgs(types)
			must.Nil(err, "decode args error: %s", err)
			vars := make([]interface{}, len(ret))
			for i := range vars {
				vars[i] = ret[i].Interface()
			}
			if len(vars) == 0 {
				vars = nil
			}
			should.Equal(test.Var, vars)
		})
	}
}
