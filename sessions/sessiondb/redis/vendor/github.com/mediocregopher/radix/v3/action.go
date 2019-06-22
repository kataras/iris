package radix

import (
	"bufio"
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"

	"github.com/mediocregopher/radix/v3/resp"
	"github.com/mediocregopher/radix/v3/resp/resp2"
)

// Action can perform one or more tasks using a Conn
type Action interface {
	// Keys returns the keys which will be acted on. Empty slice or nil may be
	// returned if no keys are being acted on.
	// The returned slice must not be modified.
	Keys() []string

	// Run actually performs the Action using the given Conn
	Run(c Conn) error
}

// CmdAction is a specific type of Action for which a command is marshaled and
// sent to the server and the command's response is read and unmarshaled into a
// receiver within the CmdAction.
//
// A CmdAction can be used like an Action, but it can also be used by marshaling
// the command and unmarshaling the response manually.
//
// A CmdAction should not be used again once UnmarshalRESP returns successfully
// from it.
type CmdAction interface {
	Action
	resp.Marshaler
	resp.Unmarshaler
}

var noKeyCmds = map[string]bool{
	"SENTINEL": true,

	"CLUSTER":   true,
	"READONLY":  true,
	"READWRITE": true,
	"ASKING":    true,

	"AUTH":   true,
	"ECHO":   true,
	"PING":   true,
	"QUIT":   true,
	"SELECT": true,
	"SWAPDB": true,

	"KEYS":      true,
	"MIGRATE":   true,
	"OBJECT":    true,
	"RANDOMKEY": true,
	"WAIT":      true,
	"SCAN":      true,

	"EVAL":    true,
	"EVALSHA": true,
	"SCRIPT":  true,

	"BGREWRITEAOF": true,
	"BGSAVE":       true,
	"CLIENT":       true,
	"COMMAND":      true,
	"CONFIG":       true,
	"DBSIZE":       true,
	"DEBUG":        true,
	"FLUSHALL":     true,
	"FLUSHDB":      true,
	"INFO":         true,
	"LASTSAVE":     true,
	"MONITOR":      true,
	"ROLE":         true,
	"SAVE":         true,
	"SHUTDOWN":     true,
	"SLAVEOF":      true,
	"SLOWLOG":      true,
	"SYNC":         true,
	"TIME":         true,

	"DISCARD": true,
	"EXEC":    true,
	"MULTI":   true,
	"UNWATCH": true,
	"WATCH":   true,
}

func cmdString(m resp.Marshaler) string {
	// we go way out of the way here to display the command as it would be sent
	// to redis. This is pretty similar logic to what the stub does as well
	buf := new(bytes.Buffer)
	if err := m.MarshalRESP(buf); err != nil {
		return fmt.Sprintf("error creating string: %q", err.Error())
	}
	var ss []string
	err := resp2.RawMessage(buf.Bytes()).UnmarshalInto(resp2.Any{I: &ss})
	if err != nil {
		return fmt.Sprintf("error creating string: %q", err.Error())
	}
	for i := range ss {
		ss[i] = strconv.QuoteToASCII(ss[i])
	}
	return "[" + strings.Join(ss, " ") + "]"
}

func marshalBulkString(prevErr error, w io.Writer, str string) error {
	if prevErr != nil {
		return prevErr
	}
	return resp2.BulkString{S: str}.MarshalRESP(w)
}

func marshalBulkStringBytes(prevErr error, w io.Writer, b []byte) error {
	if prevErr != nil {
		return prevErr
	}
	return resp2.BulkStringBytes{B: b}.MarshalRESP(w)
}

////////////////////////////////////////////////////////////////////////////////

type cmdAction struct {
	rcv  interface{}
	cmd  string
	args []string

	flat     bool
	flatKey  [1]string // use array to avoid allocation in Keys
	flatArgs []interface{}
}

// BREAM: Benchmarks Rule Everything Around Me
var cmdActionPool sync.Pool

func getCmdAction() *cmdAction {
	if ci := cmdActionPool.Get(); ci != nil {
		return ci.(*cmdAction)
	}
	return new(cmdAction)
}

// Cmd is used to perform a redis command and retrieve a result. It should not
// passed into Do more than once. See the package docs on how results are
// unmarshaled into the receiver.
//
//	if err := client.Do(radix.Cmd(nil, "SET", "foo", "bar")); err != nil {
//		panic(err)
//	}
//
//	var fooVal string
//	if err := client.Do(radix.Cmd(&fooVal, "GET", "foo")); err != nil {
//		panic(err)
//	}
//	fmt.Println(fooVal) // "bar"
//
// If the receiver value of Cmd is a primitive, a slice/map, or a struct then a
// pointer must be passed in. It may also be an io.Writer, an
// encoding.Text/BinaryUnmarshaler, or a resp.Unmarshaler.
func Cmd(rcv interface{}, cmd string, args ...string) CmdAction {
	c := getCmdAction()
	*c = cmdAction{
		rcv:  rcv,
		cmd:  cmd,
		args: args,
	}
	return c
}

// FlatCmd is like Cmd, but the arguments can be of almost any type, and FlatCmd
// will automatically flatten them into a single array of strings. Like Cmd, a
// FlatCmd should not be passed into Do more than once.
//
// FlatCmd does _not_ work for commands whose first parameter isn't a key, or
// (generally) for MSET. Use Cmd for those.
//
//	client.Do(radix.FlatCmd(nil, "SET", "foo", 1))
//	// performs "SET" "foo" "1"
//
//	client.Do(radix.FlatCmd(nil, "SADD", "fooSet", []string{"1", "2", "3"}))
//	// performs "SADD" "fooSet" "1" "2" "3"
//
//	m := map[string]int{"a":1, "b":2, "c":3}
//	client.Do(radix.FlatCmd(nil, "HMSET", "fooHash", m))
//	// performs "HMSET" "foohash" "a" "1" "b" "2" "c" "3"
//
//	// FlatCmd also supports using a resp.LenReader (an io.Reader with a Len()
//	// method) as an argument. *bytes.Buffer is an example of a LenReader,
//	// and the resp package has a NewLenReader function which can wrap an
//	// existing io.Reader. For example, if writing an http.Request body:
//	bl := resp.NewLenReader(req.Body, req.ContentLength)
//	client.Do(radix.FlatCmd(nil, "SET", "fooReq", bl))
//
// FlatCmd also supports encoding.Text/BinaryMarshalers. It does _not_ currently
// support resp.Marshaler.
//
// The receiver to FlatCmd follows the same rules as for Cmd.
func FlatCmd(rcv interface{}, cmd, key string, args ...interface{}) CmdAction {
	c := getCmdAction()
	*c = cmdAction{
		rcv:      rcv,
		cmd:      cmd,
		flat:     true,
		flatKey:  [1]string{key},
		flatArgs: args,
	}
	return c
}

func findStreamsKeys(args []string) []string {
	for i, arg := range args {
		if strings.ToUpper(arg) != "STREAMS" {
			continue
		}

		// after STREAMS only stream keys and IDs can be given and since there must be the same number of keys and ids
		// we can just take half of remaining arguments as keys. If the number of IDs does not match the number of
		// keys the command will fail later when send to Redis so no need for us to handle that case.
		ids := len(args[i+1:]) / 2

		return args[i+1 : len(args)-ids]
	}

	return nil
}

func (c *cmdAction) Keys() []string {
	if c.flat {
		return c.flatKey[:]
	}

	cmd := strings.ToUpper(c.cmd)
	if cmd == "BITOP" && len(c.args) > 1 { // antirez why you do this
		return c.args[1:]
	} else if cmd == "XINFO" {
		if len(c.args) < 2 {
			return nil
		}
		return c.args[1:2]
	} else if cmd == "XGROUP" && len(c.args) > 1 {
		return c.args[1:2]
	} else if cmd == "XREAD" || cmd == "XREADGROUP" { // antirez why you still do this
		return findStreamsKeys(c.args)
	} else if noKeyCmds[cmd] || len(c.args) == 0 {
		return nil
	}
	return c.args[:1]
}

func (c *cmdAction) flatMarshalRESP(w io.Writer) error {
	var err error
	a := resp2.Any{
		I:                     c.flatArgs,
		MarshalBulkString:     true,
		MarshalNoArrayHeaders: true,
	}
	arrL := 2 + a.NumElems()
	err = resp2.ArrayHeader{N: arrL}.MarshalRESP(w)
	err = marshalBulkString(err, w, c.cmd)
	err = marshalBulkString(err, w, c.flatKey[0])
	if err != nil {
		return err
	}
	return a.MarshalRESP(w)
}

func (c *cmdAction) MarshalRESP(w io.Writer) error {
	if c.flat {
		return c.flatMarshalRESP(w)
	}

	err := resp2.ArrayHeader{N: len(c.args) + 1}.MarshalRESP(w)
	err = marshalBulkString(err, w, c.cmd)
	for i := range c.args {
		err = marshalBulkString(err, w, c.args[i])
	}
	return err
}

func (c *cmdAction) UnmarshalRESP(br *bufio.Reader) error {
	if err := (resp2.Any{I: c.rcv}).UnmarshalRESP(br); err != nil {
		return err
	}
	cmdActionPool.Put(c)
	return nil
}

func (c *cmdAction) Run(conn Conn) error {
	if err := conn.Encode(c); err != nil {
		return err
	}
	return conn.Decode(c)
}

func (c *cmdAction) String() string {
	return cmdString(c)
}

func (c *cmdAction) ClusterCanRetry() bool {
	return true
}

////////////////////////////////////////////////////////////////////////////////

// MaybeNil is a type which wraps a receiver. It will first detect if what's
// being received is a nil RESP type (either bulk string or array), and if so
// set Nil to true. If not the return value will be unmarshaled into Rcv
// normally.
type MaybeNil struct {
	Nil bool
	Rcv interface{}
}

// UnmarshalRESP implements the method for the resp.Unmarshaler interface.
func (mn *MaybeNil) UnmarshalRESP(br *bufio.Reader) error {
	var rm resp2.RawMessage
	if err := rm.UnmarshalRESP(br); err != nil {
		return err
	} else if rm.IsNil() {
		mn.Nil = true
		return nil
	}

	return rm.UnmarshalInto(resp2.Any{I: mn.Rcv})
}

////////////////////////////////////////////////////////////////////////////////

// EvalScript contains the body of a script to be used with redis' EVAL
// functionality. Call Cmd on a EvalScript to actually create an Action which
// can be run.
type EvalScript struct {
	script, sum string
	numKeys     int
}

// NewEvalScript initializes a EvalScript instance. numKeys corresponds to the
// number of arguments which will be keys when Cmd is called
func NewEvalScript(numKeys int, script string) EvalScript {
	sumRaw := sha1.Sum([]byte(script))
	sum := hex.EncodeToString(sumRaw[:])
	return EvalScript{
		script:  script,
		sum:     sum,
		numKeys: numKeys,
	}
}

var (
	evalsha = []byte("EVALSHA")
	eval    = []byte("EVAL")
)

type evalAction struct {
	EvalScript
	args []string
	rcv  interface{}

	eval bool
}

// Cmd is like the top-level Cmd but it uses the the EvalScript to perform an
// EVALSHA command (and will automatically fallback to EVAL as necessary). args
// must be at least as long as the numKeys argument of NewEvalScript.
func (es EvalScript) Cmd(rcv interface{}, args ...string) Action {
	if len(args) < es.numKeys {
		panic("not enough arguments passed into EvalScript.Cmd")
	}
	return &evalAction{
		EvalScript: es,
		args:       args,
		rcv:        rcv,
	}
}

func (ec *evalAction) Keys() []string {
	return ec.args[:ec.numKeys]
}

func (ec *evalAction) MarshalRESP(w io.Writer) error {
	// EVAL(SHA) script/sum numkeys args...
	if err := (resp2.ArrayHeader{N: 3 + len(ec.args)}).MarshalRESP(w); err != nil {
		return err
	}

	var err error
	if ec.eval {
		err = marshalBulkStringBytes(err, w, eval)
		err = marshalBulkString(err, w, ec.script)
	} else {
		err = marshalBulkStringBytes(err, w, evalsha)
		err = marshalBulkString(err, w, ec.sum)
	}

	err = marshalBulkString(err, w, strconv.Itoa(ec.numKeys))
	for i := range ec.args {
		err = marshalBulkString(err, w, ec.args[i])
	}
	return err
}

func (ec *evalAction) Run(conn Conn) error {
	run := func(eval bool) error {
		ec.eval = eval
		if err := conn.Encode(ec); err != nil {
			return err
		}
		return conn.Decode(resp2.Any{I: ec.rcv})
	}

	err := run(false)
	if err != nil && strings.HasPrefix(err.Error(), "NOSCRIPT") {
		err = run(true)
	}
	return err
}

func (ec *evalAction) ClusterCanRetry() bool {
	return true
}

////////////////////////////////////////////////////////////////////////////////

type pipeline []CmdAction

// Pipeline returns an Action which first writes multiple commands to a Conn in
// a single write, then reads their responses in a single read. This reduces
// network delay into a single round-trip.
//
// NOTE that, while a Pipeline performs all commands on a single Conn, it
// shouldn't be used for MULTI/EXEC transactions, because if there's an error it
// won't discard the incomplete transaction. Use WithConn or EvalScript for
// transactional functionality instead.
func Pipeline(cmds ...CmdAction) Action {
	return pipeline(cmds)
}

func (p pipeline) Keys() []string {
	m := map[string]bool{}
	for _, rc := range p {
		for _, k := range rc.Keys() {
			m[k] = true
		}
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func (p pipeline) Run(c Conn) error {
	if err := c.Encode(p); err != nil {
		return err
	}
	for _, cmd := range p {
		if err := c.Decode(cmd); err != nil {
			return err
		}
	}
	return nil
}

// MarshalRESP implements the resp.Marshaler interface, so that the pipeline can pass itself to the Conn.Encode method
// instead of calling Conn.Encode for each CmdAction in the pipeline.
//
// This helps with Conn implementations that flush their underlying buffers after each call to Encode, like the default
// default Conn implementation (connWrap) does, making better use of internal buffering and automatic flushing as well
// as reducing the number of syscalls that both the client and Redis need to do.
//
// Without this, using the default Conn implementation, big pipelines can easily spend much of their time just in
// flushing (in one case measured, up to 40%).
func (p pipeline) MarshalRESP(w io.Writer) error {
	for _, cmd := range p {
		if err := cmd.MarshalRESP(w); err != nil {
			return err
		}
	}

	return nil
}

////////////////////////////////////////////////////////////////////////////////

type withConn struct {
	key [1]string // use array to avoid allocation in Keys
	fn  func(Conn) error
}

// WithConn is used to perform a set of independent Actions on the same Conn.
// key should be a key which one or more of the inner Actions is acting on, or
// "" if no keys are being acted on. The callback function is what should
// actually carry out the inner actions, and the error it returns will be
// passed back up immediately.
//
// NOTE that WithConn only ensures all inner Actions are performed on the same
// Conn, it doesn't make them transactional. Use MULTI/WATCH/EXEC within a
// WithConn for transactions, or use EvalScript
func WithConn(key string, fn func(Conn) error) Action {
	return &withConn{[1]string{key}, fn}
}

func (wc *withConn) Keys() []string {
	return wc.key[:]
}

func (wc *withConn) Run(c Conn) error {
	return wc.fn(c)
}
