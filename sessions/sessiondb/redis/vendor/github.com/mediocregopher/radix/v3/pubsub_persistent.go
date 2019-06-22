package radix

import (
	"sync"
	"time"
)

type persistentPubSub struct {
	dial func() (Conn, error)

	l           sync.Mutex
	curr        PubSubConn
	subs, psubs chanSet
	closeCh     chan struct{}
}

// PersistentPubSub is like PubSub, but instead of taking in an existing Conn to
// wrap it will create one on the fly. If the connection is ever terminated then
// a new one will be created using the connFn (which defaults to DefaultConnFunc
// if nil) and will be reset to the previous connection's state.
//
// This is effectively a way to have a permanent PubSubConn established which
// supports subscribing/unsubscribing but without the hassle of implementing
// reconnect/re-subscribe logic.
//
// None of the methods on the returned PubSubConn will ever return an error,
// they will instead block until a connection can be successfully reinstated.
func PersistentPubSub(network, addr string, connFn ConnFunc) PubSubConn {
	if connFn == nil {
		connFn = DefaultConnFunc
	}
	p := &persistentPubSub{
		dial:    func() (Conn, error) { return connFn(network, addr) },
		subs:    chanSet{},
		psubs:   chanSet{},
		closeCh: make(chan struct{}),
	}
	p.refresh()
	return p
}

func (p *persistentPubSub) refresh() {
	if p.curr != nil {
		p.curr.Close()
	}

	attempt := func() PubSubConn {
		c, err := p.dial()
		if err != nil {
			return nil
		}
		errCh := make(chan error, 1)
		pc := newPubSub(c, errCh)

		for msgCh, channels := range p.subs.inverse() {
			if err := pc.Subscribe(msgCh, channels...); err != nil {
				pc.Close()
				return nil
			}
		}

		for msgCh, patterns := range p.psubs.inverse() {
			if err := pc.PSubscribe(msgCh, patterns...); err != nil {
				pc.Close()
				return nil
			}
		}

		go func() {
			select {
			case <-errCh:
				p.l.Lock()
				// It's possible that one of the methods (e.g. Subscribe)
				// already had the lock, saw the error, and called refresh. This
				// check prevents a double-refresh in that case.
				if p.curr == pc {
					p.refresh()
				}
				p.l.Unlock()
			case <-p.closeCh:
			}
		}()
		return pc
	}

	for {
		if p.curr = attempt(); p.curr != nil {
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
}

func (p *persistentPubSub) Subscribe(msgCh chan<- PubSubMessage, channels ...string) error {
	p.l.Lock()
	defer p.l.Unlock()

	// add first, so if the actual call fails then refresh will catch it
	for _, channel := range channels {
		p.subs.add(channel, msgCh)
	}

	if err := p.curr.Subscribe(msgCh, channels...); err != nil {
		p.refresh()
	}
	return nil
}

func (p *persistentPubSub) Unsubscribe(msgCh chan<- PubSubMessage, channels ...string) error {
	p.l.Lock()
	defer p.l.Unlock()

	// remove first, so if the actual call fails then refresh will catch it
	for _, channel := range channels {
		p.subs.del(channel, msgCh)
	}

	if err := p.curr.Unsubscribe(msgCh, channels...); err != nil {
		p.refresh()
	}
	return nil
}

func (p *persistentPubSub) PSubscribe(msgCh chan<- PubSubMessage, channels ...string) error {
	p.l.Lock()
	defer p.l.Unlock()

	// add first, so if the actual call fails then refresh will catch it
	for _, channel := range channels {
		p.psubs.add(channel, msgCh)
	}

	if err := p.curr.PSubscribe(msgCh, channels...); err != nil {
		p.refresh()
	}
	return nil
}

func (p *persistentPubSub) PUnsubscribe(msgCh chan<- PubSubMessage, channels ...string) error {
	p.l.Lock()
	defer p.l.Unlock()

	// remove first, so if the actual call fails then refresh will catch it
	for _, channel := range channels {
		p.psubs.del(channel, msgCh)
	}

	if err := p.curr.PUnsubscribe(msgCh, channels...); err != nil {
		p.refresh()
	}
	return nil
}

func (p *persistentPubSub) Ping() error {
	p.l.Lock()
	defer p.l.Unlock()

	for {
		if err := p.curr.Ping(); err == nil {
			break
		}
		p.refresh()
	}
	return nil
}

func (p *persistentPubSub) Close() error {
	p.l.Lock()
	defer p.l.Unlock()
	close(p.closeCh)
	return p.curr.Close()
}
