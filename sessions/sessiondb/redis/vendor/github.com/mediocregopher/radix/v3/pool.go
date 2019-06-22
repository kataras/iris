package radix

import (
	"errors"
	"io"
	"net"
	"sync"
	"time"

	"github.com/mediocregopher/radix/v3/resp"
	"github.com/mediocregopher/radix/v3/trace"
)

// ErrPoolEmpty is used by Pools created using the PoolOnEmptyErrAfter option
var ErrPoolEmpty = errors.New("connection pool is empty")

var errPoolFull = errors.New("connection pool is full")

// ioErrConn is a Conn which tracks the last net.Error which was seen either
// during an Encode call or a Decode call
type ioErrConn struct {
	Conn

	// The most recent network error which occurred when either reading
	// or writing. A critical network error is basically any non-application
	// level error, e.g. a timeout, disconnect, etc... Close is automatically
	// called on the client when it encounters a critical network error
	lastIOErr error
}

func newIOErrConn(c Conn) *ioErrConn {
	return &ioErrConn{Conn: c}
}

func (ioc *ioErrConn) Encode(m resp.Marshaler) error {
	err := ioc.Conn.Encode(m)
	if nerr, _ := err.(net.Error); nerr != nil {
		ioc.lastIOErr = err
	}
	return err
}

func (ioc *ioErrConn) Decode(m resp.Unmarshaler) error {
	err := ioc.Conn.Decode(m)
	if nerr, _ := err.(net.Error); nerr != nil {
		ioc.lastIOErr = err
	}
	return err
}

func (ioc *ioErrConn) Do(a Action) error {
	return a.Run(ioc)
}

func (ioc *ioErrConn) Close() error {
	ioc.lastIOErr = io.EOF
	return ioc.Conn.Close()
}

////////////////////////////////////////////////////////////////////////////////

type poolOpts struct {
	cf                    ConnFunc
	pingInterval          time.Duration
	refillInterval        time.Duration
	overflowDrainInterval time.Duration
	overflowSize          int
	onEmptyWait           time.Duration
	errOnEmpty            error
	pipelineConcurrency   int
	pipelineLimit         int
	pipelineWindow        time.Duration
	pt                    trace.PoolTrace
}

// PoolOpt is an optional behavior which can be applied to the NewPool function
// to effect a Pool's behavior
type PoolOpt func(*poolOpts)

// PoolConnFunc tells the Pool to use the given ConnFunc when creating new
// Conns to its redis instance. The ConnFunc can be used to set timeouts,
// perform AUTH, or even use custom Conn implementations.
func PoolConnFunc(cf ConnFunc) PoolOpt {
	return func(po *poolOpts) {
		po.cf = cf
	}
}

// PoolPingInterval specifies the interval at which a ping event happens. On
// each ping event the Pool calls the PING redis command over one of it's
// available connections.
//
// Since connections are used in LIFO order, the ping interval * pool size is
// the duration of time it takes to ping every connection once when the pool is
// idle.
//
// A shorter interval means connections are pinged more frequently, but also
// means more traffic with the server.
func PoolPingInterval(d time.Duration) PoolOpt {
	return func(po *poolOpts) {
		po.pingInterval = d
	}
}

// PoolRefillInterval specifies the interval at which a refill event happens. On
// each refill event the Pool checks to see if it is full, and if it's not a
// single connection is created and added to it.
func PoolRefillInterval(d time.Duration) PoolOpt {
	return func(po *poolOpts) {
		po.refillInterval = d
	}
}

// PoolOnEmptyWait effects the Pool's behavior when there are no available
// connections in the Pool. The effect is to cause actions to block as long as
// it takes until a connection becomes available.
func PoolOnEmptyWait() PoolOpt {
	return func(po *poolOpts) {
		po.onEmptyWait = -1
	}
}

// PoolOnEmptyCreateAfter effects the Pool's behavior when there are no
// available connections in the Pool. The effect is to cause actions to block
// until a connection becomes available or until the duration has passed. If the
// duration is passed a new connection is created and used.
//
// If wait is 0 then a new connection is created immediately upon an empty Pool.
func PoolOnEmptyCreateAfter(wait time.Duration) PoolOpt {
	return func(po *poolOpts) {
		po.onEmptyWait = wait
		po.errOnEmpty = nil
	}
}

// PoolOnEmptyErrAfter effects the Pool's behavior when there are no
// available connections in the Pool. The effect is to cause actions to block
// until a connection becomes available or until the duration has passed. If the
// duration is passed then ErrEmptyPool is returned.
//
// If wait is 0 then ErrEmptyPool is returned immediately upon an empty Pool.
func PoolOnEmptyErrAfter(wait time.Duration) PoolOpt {
	return func(po *poolOpts) {
		po.onEmptyWait = wait
		po.errOnEmpty = ErrPoolEmpty
	}
}

// PoolOnFullClose effects the Pool's behavior when it is full. The effect is to
// cause any connection which is being put back into a full pool to be closed
// and discarded.
func PoolOnFullClose() PoolOpt {
	return func(po *poolOpts) {
		po.overflowSize = 0
		po.overflowDrainInterval = 0
	}
}

// PoolOnFullBuffer effects the Pool's behavior when it is full. The effect is
// to give the pool an additional buffer for connections, called the overflow.
// If a connection is being put back into a full pool it will be put into the
// overflow. If the overflow is also full then the connection will be closed and
// discarded.
//
// drainInterval specifies the interval at which a drain event happens. On each
// drain event a connection will be removed from the overflow buffer (if any are
// present in it), closed, and discarded.
//
// If drainInterval is zero then drain events will never occur.
func PoolOnFullBuffer(size int, drainInterval time.Duration) PoolOpt {
	return func(po *poolOpts) {
		po.overflowSize = size
		po.overflowDrainInterval = drainInterval
	}
}

// PoolPipelineConcurrency sets the maximum number of pipelines that can be
// executed concurrently.
//
// If limit is greater than the pool size or less than 1, the limit will be
// set to the pool size.
func PoolPipelineConcurrency(limit int) PoolOpt {
	return func(po *poolOpts) {
		po.pipelineConcurrency = limit
	}
}

// PoolPipelineWindow sets the duration after which internal pipelines will be
// flushed and the maximum number of commands that can be pipelined before
// flushing.
//
// If window is zero then automatic pipelining will be disabled.
// If limit is zero then no limit will be used and pipelines will only be limited
// by the specified time window.
func PoolPipelineWindow(window time.Duration, limit int) PoolOpt {
	return func(po *poolOpts) {
		po.pipelineLimit = limit
		po.pipelineWindow = window
	}
}

// PoolWithTrace tells the Pool to trace itself with the given PoolTrace
// Note that PoolTrace will block every point that you set to trace.
func PoolWithTrace(pt trace.PoolTrace) PoolOpt {
	return func(po *poolOpts) {
		po.pt = pt
	}
}

////////////////////////////////////////////////////////////////////////////////

// Pool is a semi-dynamic pool which holds a fixed number of connections open
// and which implements the Client interface. It takes in a number of options
// which can effect its specific behavior, see the NewPool method.
type Pool struct {
	opts          poolOpts
	network, addr string
	size          int

	l sync.RWMutex
	// totalConns is only really needed by the refill part of the code to ensure
	// it's not overly refilling the pool. It is protected by l.
	totalConns int
	// pool is read-protected by l, and should not be written to or read from
	// when closed is true (closed is also protected by l)
	pool   chan *ioErrConn
	closed bool

	pipeliner *pipeliner

	wg       sync.WaitGroup
	closeCh  chan bool
	initDone chan struct{} // used for tests

	// Any errors encountered internally will be written to this channel. If
	// nothing is reading the channel the errors will be dropped. The channel
	// will be closed when Close is called.
	ErrCh chan error
}

// NewPool creates a *Pool which will keep open at least the given number of
// connections to the redis instance at the given address.
//
// NewPool takes in a number of options which can overwrite its default
// behavior. The default options NewPool uses are:
//
//	PoolConnFunc(DefaultConnFunc)
//	PoolOnEmptyCreateAfter(1 * time.Second)
//	PoolRefillInterval(1 * time.Second)
//	PoolOnFullBuffer((size / 3)+1, 1 * time.Second)
//	PoolPingInterval(5 * time.Second / (size+1))
//	PoolPipelineConcurrency(size)
//	PoolPipelineWindow(150 * time.Microsecond, 0)
//
func NewPool(network, addr string, size int, opts ...PoolOpt) (*Pool, error) {
	p := &Pool{
		network:  network,
		addr:     addr,
		size:     size,
		closeCh:  make(chan bool),
		initDone: make(chan struct{}),
		ErrCh:    make(chan error, 1),
	}

	defaultPoolOpts := []PoolOpt{
		PoolConnFunc(DefaultConnFunc),
		PoolOnEmptyCreateAfter(1 * time.Second),
		PoolRefillInterval(1 * time.Second),
		PoolOnFullBuffer((size/3)+1, 1*time.Second),
		PoolPingInterval(5 * time.Second / time.Duration(size+1)),
		PoolPipelineConcurrency(size),
		// NOTE if 150us is changed the benchmarks need to be updated too
		PoolPipelineWindow(150*time.Microsecond, 0),
	}

	for _, opt := range append(defaultPoolOpts, opts...) {
		// the other args to NewPool used to be a ConnFunc, which someone might
		// have left as nil, in which case this now gives a weird panic. Just
		// handle it
		if opt != nil {
			opt(&(p.opts))
		}
	}

	totalSize := size + p.opts.overflowSize
	p.pool = make(chan *ioErrConn, totalSize)

	// make one Conn synchronously to ensure there's actually a redis instance
	// present. The rest will be created asynchronously.
	ioc, err := p.newConn(false, trace.PoolConnCreatedReasonInitialization) // false in case size is zero
	if err != nil {
		return nil, err
	}
	p.put(ioc)

	p.wg.Add(1)
	go func() {
		startTime := time.Now()
		defer p.wg.Done()
		for i := 0; i < size-1; i++ {
			ioc, err := p.newConn(true, trace.PoolConnCreatedReasonInitialization)
			if err == nil {
				p.put(ioc)
			} else {
				p.err(err)
				// if there was an error connecting to the instance than it
				// might need a little breathing room, redis can sometimes get
				// sad if too many connections are created simultaneously.
				time.Sleep(100 * time.Millisecond)
			}
		}
		close(p.initDone)
		p.traceInitCompleted(time.Since(startTime))
	}()

	// needs to be created before starting any background goroutines to avoid
	// races on p.pipeliner access
	if p.opts.pipelineWindow > 0 {
		if p.opts.pipelineConcurrency < 1 || p.opts.pipelineConcurrency > size {
			p.opts.pipelineConcurrency = size
		}

		p.pipeliner = newPipeliner(
			p,
			p.opts.pipelineConcurrency,
			p.opts.pipelineLimit,
			p.opts.pipelineWindow,
		)
	}
	if p.opts.pingInterval > 0 && size > 0 {
		p.atIntervalDo(p.opts.pingInterval, func() { p.Do(Cmd(nil, "PING")) })
	}
	if p.opts.refillInterval > 0 && size > 0 {
		p.atIntervalDo(p.opts.refillInterval, p.doRefill)
	}
	if p.opts.overflowSize > 0 && p.opts.overflowDrainInterval > 0 {
		p.atIntervalDo(p.opts.overflowDrainInterval, p.doOverflowDrain)
	}
	return p, nil
}

func (p *Pool) traceInitCompleted(elapsedTime time.Duration) {
	if p.opts.pt.InitCompleted != nil {
		p.opts.pt.InitCompleted(trace.PoolInitCompleted{
			PoolCommon:  p.traceCommon(),
			AvailCount:  len(p.pool),
			ElapsedTime: elapsedTime,
		})
	}
}

func (p *Pool) err(err error) {
	select {
	case p.ErrCh <- err:
	default:
	}
}

func (p *Pool) traceCommon() trace.PoolCommon {
	return trace.PoolCommon{
		Network: p.network, Addr: p.addr,
		PoolSize: p.size, BufferSize: p.opts.overflowSize,
	}
}

func (p *Pool) traceConnCreated(connectTime time.Duration, reason trace.PoolConnCreatedReason, err error) {
	if p.opts.pt.ConnCreated != nil {
		p.opts.pt.ConnCreated(trace.PoolConnCreated{
			PoolCommon:  p.traceCommon(),
			Reason:      reason,
			ConnectTime: connectTime,
			Err:         err,
		})
	}
}

func (p *Pool) traceConnClosed(reason trace.PoolConnClosedReason) {
	if p.opts.pt.ConnClosed != nil {
		p.opts.pt.ConnClosed(trace.PoolConnClosed{
			PoolCommon: p.traceCommon(),
			AvailCount: len(p.pool),
			Reason:     reason,
		})
	}
}

// this must always be called with p.l unlocked
func (p *Pool) newConn(errIfFull bool, reason trace.PoolConnCreatedReason) (*ioErrConn, error) {
	start := time.Now()
	c, err := p.opts.cf(p.network, p.addr)
	elapsed := time.Since(start)
	p.traceConnCreated(elapsed, reason, err)
	if err != nil {
		return nil, err
	}
	ioc := newIOErrConn(c)

	// We don't want to wrap the entire function in a lock because dialing might
	// take a while, but we also don't want to be making any new connections if
	// the pool is closed
	p.l.Lock()
	defer p.l.Unlock()
	if p.closed {
		ioc.Close()
		p.traceConnClosed(trace.PoolConnClosedReasonPoolClosed)
		return nil, errClientClosed
	} else if errIfFull && p.totalConns >= p.size {
		ioc.Close()
		p.traceConnClosed(trace.PoolConnClosedReasonPoolFull)
		return nil, errPoolFull
	}
	p.totalConns++

	return ioc, nil
}

func (p *Pool) atIntervalDo(d time.Duration, do func()) {
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		t := time.NewTicker(d)
		defer t.Stop()
		for {
			select {
			case <-t.C:
				do()
			case <-p.closeCh:
				return
			}
		}
	}()
}

func (p *Pool) doRefill() {
	// this is a preliminary check to see if more conns are needed. Technically
	// it's not needed, as newConn will do the same one, but it will also incur
	// creating a connection and fully locking the mutex. We can handle the
	// majority of cases here with a much less expensive read-lock.
	p.l.RLock()
	if p.totalConns >= p.size {
		p.l.RUnlock()
		return
	}
	p.l.RUnlock()

	ioc, err := p.newConn(true, trace.PoolConnCreatedReasonRefill)

	if err == nil {
		p.put(ioc)
	} else if err != errPoolFull {
		p.err(err)
	}
}

func (p *Pool) doOverflowDrain() {
	// the other do* processes inherently handle this case, this one needs to do
	// it manually
	p.l.RLock()

	if p.closed || len(p.pool) <= p.size {
		p.l.RUnlock()
		return
	}

	// pop a connection off and close it, if there's any to pop off
	var ioc *ioErrConn
	select {
	case ioc = <-p.pool:
	default:
		// pool is empty, nothing to drain
	}
	p.l.RUnlock()

	if ioc == nil {
		return
	}

	ioc.Close()
	p.traceConnClosed(trace.PoolConnClosedReasonBufferDrain)
	p.l.Lock()
	p.totalConns--
	p.l.Unlock()
}

func (p *Pool) getExisting() (*ioErrConn, error) {
	p.l.RLock()
	defer p.l.RUnlock()

	if p.closed {
		return nil, errClientClosed
	}

	// Fast-path if the pool is not empty.
	select {
	case ioc := <-p.pool:
		return ioc, nil
	default:
	}

	if p.opts.onEmptyWait == 0 {
		// If we should not wait we return without allocating a timer.
		return nil, p.opts.errOnEmpty
	}

	// only set when we have a timeout, since a nil channel always blocks which
	// is what we want
	var tc <-chan time.Time
	if p.opts.onEmptyWait > 0 {
		t := getTimer(p.opts.onEmptyWait)
		defer putTimer(t)

		tc = t.C
	}

	select {
	case ioc := <-p.pool:
		return ioc, nil
	case <-tc:
		return nil, p.opts.errOnEmpty
	}
}

func (p *Pool) get() (*ioErrConn, error) {
	ioc, err := p.getExisting()
	if err != nil {
		return nil, err
	} else if ioc != nil {
		return ioc, nil
	}

	// at this point everything is unlocked and the conn needs to be created.
	// newConn will handle checking if the pool has been closed since the inner
	// was called.
	return p.newConn(false, trace.PoolConnCreatedReasonPoolEmpty)
}

func (p *Pool) put(ioc *ioErrConn) {
	p.l.RLock()
	if ioc.lastIOErr == nil && !p.closed {
		select {
		case p.pool <- ioc:
			p.l.RUnlock()
			return
		default:
		}
	}

	p.l.RUnlock()
	// the pool might close here, but that's fine, because all that's happening
	// at this point is that the connection is being closed
	ioc.Close()
	p.traceConnClosed(trace.PoolConnClosedReasonPoolClosed)
	p.l.Lock()
	p.totalConns--
	p.l.Unlock()
}

// Do implements the Do method of the Client interface by retrieving a Conn out
// of the pool, calling Run on the given Action with it, and returning the Conn
// to the pool.
//
// If the given Action is a CmdAction, it will be pipelined with other concurrent
// calls to Do, which can improve the performance and resource usage of the Redis
// server, but will increase the latency for some of the Actions. To avoid the
// automatic pipelining you can either set PoolPipelineWindow(0, 0) when creating the
// Pool or use WithConn. Pipelines created manually (via Pipeline) are also excluded
// from this and will be executed as if using WithConn.
//
// Due to a limitation in the implementation, custom CmdAction implementations
// are currently not automatically pipelined.
func (p *Pool) Do(a Action) error {
	startTime := time.Now()
	if p.pipeliner != nil && p.pipeliner.CanDo(a) {
		err := p.pipeliner.Do(a)
		p.traceDoCompleted(time.Since(startTime), err)

		return err
	}

	c, err := p.get()
	if err != nil {
		return err
	}

	err = c.Do(a)
	p.put(c)
	p.traceDoCompleted(time.Since(startTime), err)

	return err
}

func (p *Pool) traceDoCompleted(elapsedTime time.Duration, err error) {
	if p.opts.pt.DoCompleted != nil {
		p.opts.pt.DoCompleted(trace.PoolDoCompleted{
			PoolCommon:  p.traceCommon(),
			AvailCount:  len(p.pool),
			ElapsedTime: elapsedTime,
			Err:         err,
		})
	}
}

// NumAvailConns returns the number of connections currently available in the
// pool, as well as in the overflow buffer if that option is enabled.
func (p *Pool) NumAvailConns() int {
	return len(p.pool)
}

// Close implements the Close method of the Client
func (p *Pool) Close() error {
	p.l.Lock()
	if p.closed {
		p.l.Unlock()
		return errClientClosed
	}
	p.closed = true
	close(p.closeCh)

	// at this point get and put won't work anymore, so it's safe to empty and
	// close the pool channel
emptyLoop:
	for {
		select {
		case ioc := <-p.pool:
			ioc.Close()
			p.traceConnClosed(trace.PoolConnClosedReasonPoolClosed)
			p.totalConns--
		default:
			close(p.pool)
			break emptyLoop
		}
	}
	p.l.Unlock()

	if p.pipeliner != nil {
		if err := p.pipeliner.Close(); err != nil {
			return err
		}
	}

	// by now the pool's go-routines should have bailed, wait to make sure they
	// do
	p.wg.Wait()
	close(p.ErrCh)
	return nil
}
