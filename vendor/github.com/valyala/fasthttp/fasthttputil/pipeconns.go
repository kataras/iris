package fasthttputil

import (
	"errors"
	"io"
	"net"
	"sync"
	"time"
)

// NewPipeConns returns new bi-directonal connection pipe.
func NewPipeConns() *PipeConns {
	ch1 := acquirePipeChan()
	ch2 := acquirePipeChan()

	pc := &PipeConns{}
	pc.c1.r = ch1
	pc.c1.w = ch2
	pc.c2.r = ch2
	pc.c2.w = ch1
	pc.c1.pc = pc
	pc.c2.pc = pc
	return pc
}

// PipeConns provides bi-directional connection pipe,
// which use in-process memory as a transport.
//
// PipeConns must be created by calling NewPipeConns.
//
// PipeConns has the following additional features comparing to connections
// returned from net.Pipe():
//
//   * It is faster.
//   * It buffers Write calls, so there is no need to have concurrent goroutine
//     calling Read in order to unblock each Write call.
type PipeConns struct {
	c1 pipeConn
	c2 pipeConn
}

// Conn1 returns the first end of bi-directional pipe.
//
// Data written to Conn1 may be read from Conn2.
// Data written to Conn2 may be read from Conn1.
func (pc *PipeConns) Conn1() net.Conn {
	return &pc.c1
}

// Conn2 returns the second end of bi-directional pipe.
//
// Data written to Conn2 may be read from Conn1.
// Data written to Conn1 may be read from Conn2.
func (pc *PipeConns) Conn2() net.Conn {
	return &pc.c2
}

func (pc *PipeConns) release() {
	pc.c1.wlock.Lock()
	pc.c2.wlock.Lock()
	mustRelease := pc.c1.wclosed && pc.c2.wclosed
	pc.c1.wlock.Unlock()
	pc.c2.wlock.Unlock()

	if mustRelease {
		pc.c1.release()
		pc.c2.release()
	}
}

type pipeConn struct {
	r  *pipeChan
	w  *pipeChan
	b  *byteBuffer
	bb []byte

	rlock   sync.Mutex
	rclosed bool

	wlock   sync.Mutex
	wclosed bool

	pc *PipeConns
}

func (c *pipeConn) Write(p []byte) (int, error) {
	b := acquireByteBuffer()
	b.b = append(b.b[:0], p...)

	c.wlock.Lock()
	if c.wclosed {
		c.wlock.Unlock()
		releaseByteBuffer(b)
		return 0, errConnectionClosed
	}
	c.w.ch <- b
	c.wlock.Unlock()

	return len(p), nil
}

func (c *pipeConn) Read(p []byte) (int, error) {
	mayBlock := true
	nn := 0
	for len(p) > 0 {
		n, err := c.read(p, mayBlock)
		nn += n
		if err != nil {
			if !mayBlock && err == errWouldBlock {
				err = nil
			}
			return nn, err
		}
		p = p[n:]
		mayBlock = false
	}

	return nn, nil
}

func (c *pipeConn) read(p []byte, mayBlock bool) (int, error) {
	if len(c.bb) == 0 {
		c.rlock.Lock()

		releaseByteBuffer(c.b)
		c.b = nil

		if c.rclosed {
			c.rlock.Unlock()
			return 0, io.EOF
		}

		if mayBlock {
			c.b = <-c.r.ch
		} else {
			select {
			case c.b = <-c.r.ch:
			default:
				c.rlock.Unlock()
				return 0, errWouldBlock
			}
		}

		if c.b == nil {
			c.rclosed = true
			c.rlock.Unlock()
			return 0, io.EOF
		}
		c.bb = c.b.b
		c.rlock.Unlock()
	}
	n := copy(p, c.bb)
	c.bb = c.bb[n:]

	return n, nil
}

var (
	errWouldBlock       = errors.New("would block")
	errConnectionClosed = errors.New("connection closed")
	errNoDeadlines      = errors.New("deadline not supported")
)

func (c *pipeConn) Close() error {
	c.wlock.Lock()
	if c.wclosed {
		c.wlock.Unlock()
		return errConnectionClosed
	}

	c.wclosed = true
	c.w.ch <- nil
	c.wlock.Unlock()

	c.pc.release()
	return nil
}

func (c *pipeConn) release() {
	c.rlock.Lock()

	releaseByteBuffer(c.b)
	c.b = nil

	if !c.rclosed {
		c.rclosed = true
		for b := range c.r.ch {
			releaseByteBuffer(b)
			if b == nil {
				break
			}
		}
	}
	if c.r != nil {
		releasePipeChan(c.r)
		c.r = nil
		c.w = nil
	}

	c.rlock.Unlock()
}

func (c *pipeConn) LocalAddr() net.Addr {
	return pipeAddr(0)
}

func (c *pipeConn) RemoteAddr() net.Addr {
	return pipeAddr(0)
}

func (c *pipeConn) SetDeadline(t time.Time) error {
	return errNoDeadlines
}

func (c *pipeConn) SetReadDeadline(t time.Time) error {
	return c.SetDeadline(t)
}

func (c *pipeConn) SetWriteDeadline(t time.Time) error {
	return c.SetDeadline(t)
}

type pipeAddr int

func (pipeAddr) Network() string {
	return "pipe"
}

func (pipeAddr) String() string {
	return "pipe"
}

type byteBuffer struct {
	b []byte
}

func acquireByteBuffer() *byteBuffer {
	return byteBufferPool.Get().(*byteBuffer)
}

func releaseByteBuffer(b *byteBuffer) {
	if b != nil {
		byteBufferPool.Put(b)
	}
}

var byteBufferPool = &sync.Pool{
	New: func() interface{} {
		return &byteBuffer{
			b: make([]byte, 1024),
		}
	},
}

func acquirePipeChan() *pipeChan {
	ch := pipeChanPool.Get().(*pipeChan)
	if len(ch.ch) > 0 {
		panic("BUG: non-empty pipeChan acquired")
	}
	return ch
}

func releasePipeChan(ch *pipeChan) {
	if len(ch.ch) > 0 {
		panic("BUG: non-empty pipeChan released")
	}
	pipeChanPool.Put(ch)
}

var pipeChanPool = &sync.Pool{
	New: func() interface{} {
		return &pipeChan{
			ch: make(chan *byteBuffer, 4),
		}
	},
}

type pipeChan struct {
	ch chan *byteBuffer
}
