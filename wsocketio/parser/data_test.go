package parser

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type noBufferStruct struct {
	Str   string            `json:"str"`
	I     int               `json:"i"`
	Array []int             `json:"array"`
	Map   map[string]string `json:"map"`
}

type bufferStruct struct {
	I      int     `json:"i"`
	Buffer *Buffer `json:"buf"`
}

type bufferInnerStruct struct {
	I      int                `json:"i"`
	Buffer *Buffer            `json:"buf"`
	Inner  *bufferInnerStruct `json:"inner,omitempty"`
}

var tests = []struct {
	Name   string
	Header Header
	Event  string
	Var    []interface{}
	Datas  [][]byte
}{
	{"Empty", Header{Connect, "", 0, false}, "", nil, [][]byte{
		[]byte("0"),
	}},
	{"Data", Header{Error, "", 0, false}, "", []interface{}{"error"}, [][]byte{
		[]byte("4[\"error\"]\n"),
	}},
	{"BData", Header{Event, "", 0, false}, "msg", []interface{}{
		&Buffer{Data: []byte{1, 2, 3}},
	}, [][]byte{
		[]byte("51-[\"msg\",{\"_placeholder\":true,\"num\":0}]\n"),
		[]byte{1, 2, 3},
	}},
	{"ID", Header{Connect, "", 0, true}, "", nil, [][]byte{
		[]byte("00"),
	}},
	{"IDData", Header{Ack, "", 13, true}, "", []interface{}{"error"}, [][]byte{
		[]byte("313[\"error\"]\n"),
	}},
	{"IDBData", Header{Ack, "", 13, true}, "", []interface{}{
		&Buffer{
			Data: []byte{1, 2, 3},
		}}, [][]byte{
		[]byte("61-13[{\"_placeholder\":true,\"num\":0}]\n"),
		[]byte{1, 2, 3},
	}},
	{"Namespace", Header{Disconnect, "/woot", 0, false}, "", nil, [][]byte{
		[]byte("1/woot"),
	}},
	{"NamespaceData", Header{Event, "/woot", 0, false}, "msg", []interface{}{
		1,
	}, [][]byte{
		[]byte("2/woot,[\"msg\",1]\n"),
	}},
	{"NamespaceBData", Header{Event, "/woot", 0, false}, "msg", []interface{}{
		&Buffer{Data: []byte{2, 3, 4}},
	}, [][]byte{
		[]byte("51-/woot,[\"msg\",{\"_placeholder\":true,\"num\":0}]\n"),
		[]byte{2, 3, 4},
	}},
	{"NamespaceID", Header{Disconnect, "/woot", 1, true}, "", nil, [][]byte{
		[]byte("1/woot,1"),
	}},
	{"NamespaceIDData", Header{Event, "/woot", 1, true}, "msg", []interface{}{
		1,
	}, [][]byte{
		[]byte("2/woot,1[\"msg\",1]\n"),
	}},
	{"NamespaceIDBData", Header{Event, "/woot", 1, true}, "msg", []interface{}{
		&Buffer{Data: []byte{2, 3, 4}},
	}, [][]byte{
		[]byte("51-/woot,1[\"msg\",{\"_placeholder\":true,\"num\":0}]\n"),
		[]byte{2, 3, 4},
	}},
}

var attachmentTests = []struct {
	buffer         Buffer
	textEncoding   string
	binaryEncoding string
}{
	{
		Buffer{[]byte{1, 255}, false, 0},
		`{"type":"Buffer","data":[1,255]}`,
		`{"_placeholder":true,"num":0}`,
	},
	{
		Buffer{[]byte{}, false, 1},
		`{"type":"Buffer","data":[]}`,
		`{"_placeholder":true,"num":1}`,
	},
	{
		Buffer{nil, false, 2},
		`{"type":"Buffer","data":[]}`,
		`{"_placeholder":true,"num":2}`,
	},
}

func TestAttachmentEncodeText(t *testing.T) {
	should := assert.New(t)
	must := require.New(t)

	for _, test := range attachmentTests {
		a := test.buffer
		a.isBinary = false
		j, err := json.Marshal(a)
		must.Nil(err)
		should.Equal(test.textEncoding, string(j))
	}
}

func TestAttachmentEncodeBinary(t *testing.T) {
	should := assert.New(t)
	must := require.New(t)

	for _, test := range attachmentTests {
		a := test.buffer
		a.isBinary = true
		j, err := json.Marshal(a)
		must.Nil(err)
		should.Equal(test.binaryEncoding, string(j))
	}
}

func TestAttachmentDecodeText(t *testing.T) {
	should := assert.New(t)
	must := require.New(t)

	for _, test := range attachmentTests {
		var a Buffer
		err := json.Unmarshal([]byte(test.textEncoding), &a)
		must.Nil(err)
		should.False(a.isBinary)
		if len(test.buffer.Data) == 0 {
			should.Equal([]byte{}, a.Data)
			continue
		}
		should.Equal(test.buffer.Data, a.Data)
	}
}

func TestAttachmentDecodeBinary(t *testing.T) {
	should := assert.New(t)
	must := require.New(t)

	for _, test := range attachmentTests {
		var a Buffer
		err := json.Unmarshal([]byte(test.binaryEncoding), &a)
		must.Nil(err)
		should.True(a.isBinary)
		should.Equal(test.buffer.num, a.num)
	}
}
