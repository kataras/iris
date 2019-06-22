// Package radix implements all functionality needed to work with redis and all
// things related to it, including redis cluster, pubsub, sentinel, scanning,
// lua scripting, and more.
//
// Creating a client
//
// For a single node redis instance use NewPool to create a connection pool. The
// connection pool is thread-safe and will automatically create, reuse, and
// recreate connections as needed:
//
//	pool, err := radix.NewPool("tcp", "127.0.0.1:6379", 10)
//	if err != nil {
//		// handle error
//	}
//
// If you're using sentinel or cluster you should use NewSentinel or NewCluster
// (respectively) to create your client instead.
//
// Commands
//
// Any redis command can be performed by passing a Cmd into a Client's Do
// method. Each Cmd should only be used once. The return from the Cmd can be
// captured into any appopriate go primitive type, or a slice or map if the
// command returns an array.
//
//	err := client.Do(radix.Cmd(nil, "SET", "foo", "someval"))
//
//	var fooVal string
//	err := client.Do(radix.Cmd(&fooVal, "GET", "foo"))
//
//	var fooValB []byte
//	err := client.Do(radix.Cmd(&fooValB, "GET", "foo"))
//
//	var barI int
//	err := client.Do(radix.Cmd(&barI, "INCR", "bar"))
//
//	var bazEls []string
//	err := client.Do(radix.Cmd(&bazEls, "LRANGE", "baz", "0", "-1"))
//
//	var buzMap map[string]string
//	err := client.Do(radix.Cmd(&buzMap, "HGETALL", "buz"))
//
// FlatCmd can also be used if you wish to use non-string arguments like
// integers, slices, maps, or structs, and have them automatically be flattened
// into a single string slice.
//
// Struct Scanning
//
// Cmd and FlatCmd can also unmarshal results into a struct. The results must be
// a key/value array, such as that returned by HGETALL. Exported field names
// will be used as keys, unless the fields have the "redis" tag:
//
//	type MyType struct {
//		Foo string               // Will be populated with the value for key "Foo"
//		Bar string `redis:"BAR"` // Will be populated with the value for key "BAR"
//		Baz string `redis:"-"`   // Will not be populated
//	}
//
// Embedded structs will inline that struct's fields into the parent's:
//
//	type MyOtherType struct {
//		// adds fields "Foo" and "BAR" (from above example) to MyOtherType
//		MyType
//		Biz int
//	}
//
// The same rules for field naming apply when a struct is passed into FlatCmd as
// an argument.
//
// Actions
//
// Cmd and FlatCmd both implement the Action interface. Other Actions include
// Pipeline, WithConn, and EvalScript.Cmd. Any of these may be passed into any
// Client's Do method.
//
//	var fooVal string
//	p := radix.Pipeline(
//		radix.FlatCmd(nil, "SET", "foo", 1),
//		radix.Cmd(&fooVal, "GET", "foo"),
//	)
//	if err := client.Do(p); err != nil {
//		panic(err)
//	}
//	fmt.Printf("fooVal: %q\n", fooVal)
//
// Transactions
//
// There are two ways to perform transactions in redis. The first is with the
// MULTI/EXEC commands, which can be done using the WithConn Action (see its
// example). The second is using EVAL with lua scripting, which can be done
// using the EvalScript Action (again, see its example).
//
// EVAL with lua scripting is recommended in almost all cases. It only requires
// a single round-trip, it's infinitely more flexible than MULTI/EXEC, it's
// simpler to code, and for complex transactions, which would otherwise need a
// WATCH statement with MULTI/EXEC, it's significantly faster.
//
// AUTH and other settings via ConnFunc and ClientFunc
//
// All the client creation functions (e.g. NewPool) take in either a ConnFunc or
// a ClientFunc via their options. These can be used in order to set up timeouts
// on connections, perform authentication commands, or even implement custom
// pools.
//
//	// this is a ConnFunc which will set up a connection which is authenticated
//	// and has a 1 minute timeout on all operations
//	customConnFunc := func(network, addr string) (radix.Conn, error) {
//		return radix.Dial(network, addr,
//			radix.DialTimeout(1 * time.Minute),
//			radix.DialAuthPass("mySuperSecretPassword"),
//		)
//	}
//
//	// this pool will use our ConnFunc for all connections it creates
//	pool, err := radix.NewPool("tcp", redisAddr, 10, PoolConnFunc(customConnFunc))
//
//	// this cluster will use the ClientFunc to create a pool to each node in the
//	// cluster. The pools also use our customConnFunc, but have more connections
//	poolFunc := func(network, addr string) (radix.Client, error) {
//		return radix.NewPool(network, addr, 100, PoolConnFunc(customConnFunc))
//	}
//	cluster, err := radix.NewCluster([]string{redisAddr1, redisAddr2}, ClusterPoolFunc(poolFunc))
//
// Custom implementations
//
// All interfaces in this package were designed such that they could have custom
// implementations. There is no dependency within radix that demands any
// interface be implemented by a particular underlying type, so feel free to
// create your own Pools or Conns or Actions or whatever makes your life easier.
//
package radix

import (
	"bufio"
	"crypto/tls"
	"errors"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/mediocregopher/radix/v3/resp"
)

var errClientClosed = errors.New("client is closed")

// Client describes an entity which can carry out Actions, e.g. a connection
// pool for a single redis instance or the cluster client.
//
// Implementations of Client are expected to be thread-safe, except in cases
// like Conn where they specify otherwise.
type Client interface {
	// Do performs an Action, returning any error.
	Do(Action) error

	// Once Close() is called all future method calls on the Client will return
	// an error
	Close() error
}

// ClientFunc is a function which can be used to create a Client for a single
// redis instance on the given network/address.
type ClientFunc func(network, addr string) (Client, error)

// DefaultClientFunc is a ClientFunc which will return a Client for a redis
// instance using sane defaults.
var DefaultClientFunc = func(network, addr string) (Client, error) {
	return NewPool(network, addr, 4)
}

// Conn is a Client wrapping a single network connection which synchronously
// reads/writes data using the redis resp protocol.
//
// A Conn can be used directly as a Client, but in general you probably want to
// use a *Pool instead
type Conn interface {
	// The Do method of a Conn is _not_ expected to be thread-safe.
	Client

	// Encode and Decode may be called at the same time by two different
	// go-routines, but each should only be called once at a time (i.e. two
	// routines shouldn't call Encode at the same time, same with Decode).
	//
	// Encode and Decode should _not_ be called at the same time as Do.
	//
	// If either Encode or Decode encounter a net.Error the Conn will be
	// automatically closed.
	//
	// Encode is expected to encode an entire resp message, not a partial one.
	// In other words, when sending commands to redis, Encode should only be
	// called once per command. Similarly, Decode is expected to decode an
	// entire resp response.
	Encode(resp.Marshaler) error
	Decode(resp.Unmarshaler) error

	// Returns the underlying network connection, as-is. Read, Write, and Close
	// should not be called on the returned Conn.
	NetConn() net.Conn
}

type connWrap struct {
	net.Conn
	brw *bufio.ReadWriter
}

// NewConn takes an existing net.Conn and wraps it to support the Conn interface
// of this package. The Read and Write methods on the original net.Conn should
// not be used after calling this method.
func NewConn(conn net.Conn) Conn {
	return &connWrap{
		Conn: conn,
		brw:  bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn)),
	}
}

func (cw *connWrap) Do(a Action) error {
	return a.Run(cw)
}

func (cw *connWrap) Encode(m resp.Marshaler) error {
	if err := m.MarshalRESP(cw.brw); err != nil {
		return err
	}
	return cw.brw.Flush()
}

func (cw *connWrap) Decode(u resp.Unmarshaler) error {
	return u.UnmarshalRESP(cw.brw.Reader)
}

func (cw *connWrap) NetConn() net.Conn {
	return cw.Conn
}

// ConnFunc is a function which returns an initialized, ready-to-be-used Conn.
// Functions like NewPool or NewCluster take in a ConnFunc in order to allow for
// things like calls to AUTH on each new connection, setting timeouts, custom
// Conn implementations, etc... See the package docs for more details.
type ConnFunc func(network, addr string) (Conn, error)

// DefaultConnFunc is a ConnFunc which will return a Conn for a redis instance
// using sane defaults.
var DefaultConnFunc = func(network, addr string) (Conn, error) {
	return Dial(network, addr)
}

func wrapDefaultConnFunc(addr string) ConnFunc {
	_, opts := parseRedisURL(addr)
	return func(network, addr string) (Conn, error) {
		return Dial(network, addr, opts...)
	}
}

////////////////////////////////////////////////////////////////////////////////

type dialOpts struct {
	connectTimeout, readTimeout, writeTimeout time.Duration
	authPass                                  string
	selectDB                                  string
	useTLSConfig                              bool
	tlsConfig                                 *tls.Config
}

// DialOpt is an optional behavior which can be applied to the Dial function to
// effect its behavior, or the behavior of the Conn it creates.
type DialOpt func(*dialOpts)

// DialConnectTimeout determines the timeout value to pass into net.DialTimeout
// when creating the connection. If not set then net.Dial is called instead.
func DialConnectTimeout(d time.Duration) DialOpt {
	return func(do *dialOpts) {
		do.connectTimeout = d
	}
}

// DialReadTimeout determines the deadline to set when reading from a dialed
// connection. If not set then SetReadDeadline is never called.
func DialReadTimeout(d time.Duration) DialOpt {
	return func(do *dialOpts) {
		do.readTimeout = d
	}
}

// DialWriteTimeout determines the deadline to set when writing to a dialed
// connection. If not set then SetWriteDeadline is never called.
func DialWriteTimeout(d time.Duration) DialOpt {
	return func(do *dialOpts) {
		do.writeTimeout = d
	}
}

// DialTimeout is the equivalent to using DialConnectTimeout, DialReadTimeout,
// and DialWriteTimeout all with the same value.
func DialTimeout(d time.Duration) DialOpt {
	return func(do *dialOpts) {
		DialConnectTimeout(d)(do)
		DialReadTimeout(d)(do)
		DialWriteTimeout(d)(do)
	}
}

// DialAuthPass will cause Dial to perform an AUTH command once the connection
// is created, using the given pass.
//
// If this is set and a redis URI is passed to Dial which also has a password
// set, this takes precedence.
func DialAuthPass(pass string) DialOpt {
	return func(do *dialOpts) {
		do.authPass = pass
	}
}

// DialSelectDB will cause Dial to perform a SELECT command once the connection
// is created, using the given database index.
//
// If this is set and a redis URI is passed to Dial which also has a database
// index set, this takes precedence.
func DialSelectDB(db int) DialOpt {
	return func(do *dialOpts) {
		do.selectDB = strconv.Itoa(db)
	}
}

// DialUseTLS will cause Dial to perform a TLS handshake using the provided
// config. If config is nil the config is interpreted as equivalent to the zero
// configuration. See https://golang.org/pkg/crypto/tls/#Config
func DialUseTLS(config *tls.Config) DialOpt {
	return func(do *dialOpts) {
		do.tlsConfig = config
		do.useTLSConfig = true
	}
}

type timeoutConn struct {
	net.Conn
	readTimeout, writeTimeout time.Duration
}

func (tc *timeoutConn) Read(b []byte) (int, error) {
	if tc.readTimeout > 0 {
		tc.Conn.SetReadDeadline(time.Now().Add(tc.readTimeout))
	}
	return tc.Conn.Read(b)
}

func (tc *timeoutConn) Write(b []byte) (int, error) {
	if tc.writeTimeout > 0 {
		tc.Conn.SetWriteDeadline(time.Now().Add(tc.writeTimeout))
	}
	return tc.Conn.Write(b)
}

var defaultDialOpts = []DialOpt{
	DialTimeout(10 * time.Second),
}

func parseRedisURL(urlStr string) (string, []DialOpt) {
	// do a quick check before we bust out url.Parse, in case that is very
	// unperformant
	if !strings.HasPrefix(urlStr, "redis://") {
		return urlStr, nil
	}

	u, err := url.Parse(urlStr)
	if err != nil {
		return urlStr, nil
	}

	var opts []DialOpt
	q := u.Query()
	if p, ok := u.User.Password(); ok {
		opts = append(opts, DialAuthPass(p))
	} else if qpw := q.Get("password"); qpw != "" {
		opts = append(opts, DialAuthPass(qpw))
	}

	dbStr := q.Get("db")
	if u.Path != "" && u.Path != "/" {
		dbStr = u.Path[1:]
	}

	if dbStr, err := strconv.Atoi(dbStr); err == nil {
		opts = append(opts, DialSelectDB(dbStr))
	}

	return u.Host, opts
}

// Dial is a ConnFunc which creates a Conn using net.Dial and NewConn. It takes
// in a number of options which can overwrite its default behavior as well.
//
// In place of a host:port address, Dial also accepts a URI, as per:
// 	https://www.iana.org/assignments/uri-schemes/prov/redis
// If the URI has an AUTH password or db specified Dial will attempt to perform
// the AUTH and/or SELECT as well.
//
// If either DialAuthPass or DialSelectDB is used it overwrites the associated
// value passed in by the URI.
//
// The default options Dial uses are:
//
//	DialTimeout(10 * time.Second)
//
func Dial(network, addr string, opts ...DialOpt) (Conn, error) {
	var do dialOpts
	for _, opt := range defaultDialOpts {
		opt(&do)
	}
	addr, addrOpts := parseRedisURL(addr)
	for _, opt := range addrOpts {
		opt(&do)
	}
	for _, opt := range opts {
		opt(&do)
	}

	var netConn net.Conn
	var err error
	dialer := net.Dialer{}
	if do.connectTimeout > 0 {
		dialer.Timeout = do.connectTimeout
	}
	if do.useTLSConfig {
		netConn, err = tls.DialWithDialer(&dialer, network, addr, do.tlsConfig)
	} else {
		netConn, err = dialer.Dial(network, addr)
	}

	if err != nil {
		return nil, err
	}

	// If the netConn is a net.TCPConn (or some wrapper for it) and so can have
	// keepalive enabled, do so with a sane (though slightly aggressive)
	// default.
	{
		type keepaliveConn interface {
			SetKeepAlive(bool) error
			SetKeepAlivePeriod(time.Duration) error
		}

		if kaConn, ok := netConn.(keepaliveConn); ok {
			if err = kaConn.SetKeepAlive(true); err != nil {
				netConn.Close()
				return nil, err
			} else if err = kaConn.SetKeepAlivePeriod(10 * time.Second); err != nil {
				netConn.Close()
				return nil, err
			}
		}
	}

	conn := NewConn(&timeoutConn{
		readTimeout:  do.readTimeout,
		writeTimeout: do.writeTimeout,
		Conn:         netConn,
	})

	if do.authPass != "" {
		if err := conn.Do(Cmd(nil, "AUTH", do.authPass)); err != nil {
			conn.Close()
			return nil, err
		}
	}

	if do.selectDB != "" {
		if err := conn.Do(Cmd(nil, "SELECT", do.selectDB)); err != nil {
			conn.Close()
			return nil, err
		}
	}

	return conn, nil
}
