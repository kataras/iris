package websocket

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"strconv"

	"github.com/kataras/iris/core/errors"
	"github.com/valyala/bytebufferpool"
)

type (
	messageType uint8
)

func (m messageType) String() string {
	return strconv.Itoa(int(m))
}

func (m messageType) Name() string {
	switch m {
	case messageTypeString:
		return "string"
	case messageTypeInt:
		return "int"
	case messageTypeBool:
		return "bool"
	case messageTypeBytes:
		return "[]byte"
	case messageTypeJSON:
		return "json"
	default:
		return "Invalid(" + m.String() + ")"
	}
}

// The same values are exists on client side too.
const (
	messageTypeString messageType = iota
	messageTypeInt
	messageTypeBool
	messageTypeBytes
	messageTypeJSON
)

const (
	messageSeparator = ";"
)

var messageSeparatorByte = messageSeparator[0]

type messageSerializer struct {
	prefix []byte

	prefixLen       int
	separatorLen    int
	prefixAndSepIdx int
	prefixIdx       int
	separatorIdx    int

	buf *bytebufferpool.Pool
}

func newMessageSerializer(messagePrefix []byte) *messageSerializer {
	return &messageSerializer{
		prefix:          messagePrefix,
		prefixLen:       len(messagePrefix),
		separatorLen:    len(messageSeparator),
		prefixAndSepIdx: len(messagePrefix) + len(messageSeparator) - 1,
		prefixIdx:       len(messagePrefix) - 1,
		separatorIdx:    len(messageSeparator) - 1,

		buf: new(bytebufferpool.Pool),
	}
}

var (
	boolTrueB  = []byte("true")
	boolFalseB = []byte("false")
)

// websocketMessageSerialize serializes a custom websocket message from websocketServer to be delivered to the client
// returns the string form of the message
// Supported data types are: string, int, bool, bytes and JSON.
func (ms *messageSerializer) serialize(event string, data interface{}) ([]byte, error) {
	b := ms.buf.Get()
	b.Write(ms.prefix)
	b.WriteString(event)
	b.WriteByte(messageSeparatorByte)

	switch v := data.(type) {
	case string:
		b.WriteString(messageTypeString.String())
		b.WriteByte(messageSeparatorByte)
		b.WriteString(v)
	case int:
		b.WriteString(messageTypeInt.String())
		b.WriteByte(messageSeparatorByte)
		binary.Write(b, binary.LittleEndian, v)
	case bool:
		b.WriteString(messageTypeBool.String())
		b.WriteByte(messageSeparatorByte)
		if v {
			b.Write(boolTrueB)
		} else {
			b.Write(boolFalseB)
		}
	case []byte:
		b.WriteString(messageTypeBytes.String())
		b.WriteByte(messageSeparatorByte)
		b.Write(v)
	default:
		//we suppose is json
		res, err := json.Marshal(data)
		if err != nil {
			ms.buf.Put(b)
			return nil, err
		}
		b.WriteString(messageTypeJSON.String())
		b.WriteByte(messageSeparatorByte)
		b.Write(res)
	}

	message := b.Bytes()
	ms.buf.Put(b)

	return message, nil
}

var errInvalidTypeMessage = errors.New("Type %s is invalid for message: %s")

// deserialize deserializes a custom websocket message from the client
// ex: iris-websocket-message;chat;4;themarshaledstringfromajsonstruct will return 'hello' as string
// Supported data types are: string, int, bool, bytes and JSON.
func (ms *messageSerializer) deserialize(event []byte, websocketMessage []byte) (interface{}, error) {
	dataStartIdx := ms.prefixAndSepIdx + len(event) + 3
	if len(websocketMessage) <= dataStartIdx {
		return nil, errors.New("websocket invalid message: " + string(websocketMessage))
	}

	typ, err := strconv.Atoi(string(websocketMessage[ms.prefixAndSepIdx+len(event)+1 : ms.prefixAndSepIdx+len(event)+2])) // in order to iris-websocket-message;user;-> 4
	if err != nil {
		return nil, err
	}

	data := websocketMessage[dataStartIdx:] // in order to iris-websocket-message;user;4; -> themarshaledstringfromajsonstruct

	switch messageType(typ) {
	case messageTypeString:
		return string(data), nil
	case messageTypeInt:
		msg, err := strconv.Atoi(string(data))
		if err != nil {
			return nil, err
		}
		return msg, nil
	case messageTypeBool:
		if bytes.Equal(data, boolTrueB) {
			return true, nil
		}
		return false, nil
	case messageTypeBytes:
		return data, nil
	case messageTypeJSON:
		var msg interface{}
		err := json.Unmarshal(data, &msg)
		return msg, err
	default:
		return nil, errInvalidTypeMessage.Format(messageType(typ).Name(), websocketMessage)
	}
}

// getWebsocketCustomEvent return empty string when the websocketMessage is native message
func (ms *messageSerializer) getWebsocketCustomEvent(websocketMessage []byte) []byte {
	if len(websocketMessage) < ms.prefixAndSepIdx {
		return nil
	}
	s := websocketMessage[ms.prefixAndSepIdx:]
	evt := s[:bytes.IndexByte(s, messageSeparatorByte)]
	return evt
}
