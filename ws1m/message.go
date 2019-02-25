package ws1m

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"strconv"
	"github.com/valyala/bytebufferpool"
	"sync"
	"runtime"
	"github.com/kataras/iris/core/errors"
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
	wg              sync.WaitGroup
	buf             *bytebufferpool.Pool
	//bufpr *io.PipeReader
	//bufpw *io.PipeWriter
}

func newMessageSerializer(messagePrefix []byte) *messageSerializer {
	//pr, pw := io.Pipe()
	return &messageSerializer{
		prefix:          messagePrefix,
		prefixLen:       len(messagePrefix),
		separatorLen:    len(messageSeparator),
		prefixAndSepIdx: len(messagePrefix) + len(messageSeparator) - 1,
		prefixIdx:       len(messagePrefix) - 1,
		separatorIdx:    len(messageSeparator) - 1,

		buf: new(bytebufferpool.Pool),
		//bufpr: pr,
		//bufpw: pw,
		wg: sync.WaitGroup{},
	}
}

var (
	boolTrueB  = []byte("true")
	boolFalseB = []byte("false")
)

// websocketMessageSerialize serializes a custom websocket message from websocketServer to be delivered to the client
// returns the  string form of the message
// Supported data types are: string, int, bool, bytes and JSON.
func (ms *messageSerializer) serialize(event string, data interface{}) ([]byte, error) {
	//after a series break point check.. (jjhesk) found that this is the best approach..
	//
	// to prevent race condition.
	// because there are two loops running and one is reading and another one writing. the writings always comes first
	// at any conditions.
	runtime.Gosched()
	// need to prevent the out going messages comes first
	b := NewPool().Get()
	//ms.wg.Add(1)
	//go func(b *Buffer, event string, data interface{}, w *sync.WaitGroup) {
	b.AppendString(string(ms.prefix))
	b.AppendString(event)
	b.AppendString(messageSeparator)
	switch v := data.(type) {
	case string:
		b.AppendString(messageTypeString.String())
		b.AppendString(messageSeparator)
		b.AppendString(v)
		//println("string <---------------")
	case int:
		b.AppendString(messageTypeInt.String())
		b.AppendString(messageSeparator)
		binary.Write(b, binary.LittleEndian, v)
	//	println("int <---------------")
	case bool:
		b.AppendString(messageTypeBool.String())
		b.AppendString(messageSeparator)
		if v {
			b.Write(boolTrueB)
		} else {
			b.Write(boolFalseB)
		}
	//	println("bool <---------------")
	case []byte:
		b.AppendString(messageTypeBytes.String())
		b.AppendString(messageSeparator)
		b.Write(v)
	//	println("byte <---------------")
	default:
		//we suppose is json
		res, err := json.Marshal(data)
		if err != nil {
			//return nil, err
		}
		b.AppendString(messageTypeJSON.String())
		b.AppendString(messageSeparator)
		b.Write(res)
	}
	//	w.Done()
	//}(b, event, data, &ms.wg)

	//ms.wg.Wait()
	//	message := b.Bytes()
	//  ms.buf.Put(b)

	// println("🎦  " + b.String())

	//b.Free()


	//todo: check for better perf and try to recycle it if that allocate too much space under stress testing
	return b.Bytes(), nil
}

// websocketMessageSerialize serializes a custom websocket message from websocketServer to be delivered to the client
// returns the  string form of the message
// Supported data types are: string, int, bool, bytes and JSON.
func (ms *messageSerializer) serialize0(event string, data interface{}) ([]byte, error) {

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
			return nil, err
		}
		b.WriteString(messageTypeJSON.String())
		b.WriteByte(messageSeparatorByte)
		b.Write(res)
	}

	message := b.Bytes()
	ms.buf.Put(b)

	// println("🎦  " + string(message))
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
