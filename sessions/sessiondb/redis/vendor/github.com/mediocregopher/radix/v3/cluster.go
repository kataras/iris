package radix

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/mediocregopher/radix/v3/resp"
	"github.com/mediocregopher/radix/v3/resp/resp2"
	"github.com/mediocregopher/radix/v3/trace"
)

// dedupe is used to deduplicate a function invocation, so if multiple
// go-routines call it at the same time only the first will actually run it, and
// the others will block until that one is done.
type dedupe struct {
	l sync.Mutex
	s *sync.Once
}

func newDedupe() *dedupe {
	return &dedupe{s: new(sync.Once)}
}

func (d *dedupe) do(fn func()) {
	d.l.Lock()
	s := d.s
	d.l.Unlock()

	s.Do(func() {
		fn()
		d.l.Lock()
		d.s = new(sync.Once)
		d.l.Unlock()
	})
}

////////////////////////////////////////////////////////////////////////////////

// ClusterCanRetryAction is an Action which is aware of Cluster's retry behavior
// in the event of a slot migration. If an Action receives an error from a
// Cluster node which is either MOVED or ASK, and that Action implements
// ClusterCanRetryAction, and the ClusterCanRetry method returns true, then the
// Action will be retried on the correct node.
//
// NOTE that the Actions which are returned by Cmd, FlatCmd, and EvalScript.Cmd
// all implicitly implement this interface.
type ClusterCanRetryAction interface {
	Action
	ClusterCanRetry() bool
}

////////////////////////////////////////////////////////////////////////////////

type clusterOpts struct {
	pf        ClientFunc
	syncEvery time.Duration
	ct        trace.ClusterTrace
}

// ClusterOpt is an optional behavior which can be applied to the NewCluster
// function to effect a Cluster's behavior
type ClusterOpt func(*clusterOpts)

// ClusterPoolFunc tells the Cluster to use the given ClientFunc when creating
// pools of connections to cluster members.
func ClusterPoolFunc(pf ClientFunc) ClusterOpt {
	return func(co *clusterOpts) {
		co.pf = pf
	}
}

// ClusterSyncEvery tells the Cluster to synchronize itself with the cluster's
// topology at the given interval. On every synchronization Cluster will ask the
// cluster for its topology and make/destroy its connections as necessary.
func ClusterSyncEvery(d time.Duration) ClusterOpt {
	return func(co *clusterOpts) {
		co.syncEvery = d
	}
}

// ClusterWithTrace tells the Cluster to trace itself with the given ClusterTrace
// Note that ClusterTrace will block every point that you set to trace.
func ClusterWithTrace(ct trace.ClusterTrace) ClusterOpt {
	return func(co *clusterOpts) {
		co.ct = ct
	}
}

// Cluster contains all information about a redis cluster needed to interact
// with it, including a set of pools to each of its instances. All methods on
// Cluster are thread-safe
type Cluster struct {
	co clusterOpts

	// used to deduplicate calls to sync
	syncDedupe *dedupe

	l              sync.RWMutex
	pools          map[string]Client
	primTopo, topo ClusterTopo

	closeCh   chan struct{}
	closeWG   sync.WaitGroup
	closeOnce sync.Once

	// Any errors encountered internally will be written to this channel. If
	// nothing is reading the channel the errors will be dropped. The channel
	// will be closed when the Close is called.
	ErrCh chan error
}

// NewCluster initializes and returns a Cluster instance. It will try every
// address given until it finds a usable one. From there it uses CLUSTER SLOTS
// to discover the cluster topology and make all the necessary connections.
//
// NewCluster takes in a number of options which can overwrite its default
// behavior. The default options NewCluster uses are:
//
//	ClusterPoolFunc(DefaultClientFunc)
//	ClusterSyncEvery(5 * time.Second)
//
func NewCluster(clusterAddrs []string, opts ...ClusterOpt) (*Cluster, error) {
	c := &Cluster{
		syncDedupe: newDedupe(),
		pools:      map[string]Client{},
		closeCh:    make(chan struct{}),
		ErrCh:      make(chan error, 1),
	}

	defaultClusterOpts := []ClusterOpt{
		ClusterPoolFunc(DefaultClientFunc),
		ClusterSyncEvery(5 * time.Second),
	}

	for _, opt := range append(defaultClusterOpts, opts...) {
		// the other args to NewCluster used to be a ClientFunc, which someone
		// might have left as nil, in which case this now gives a weird panic.
		// Just handle it
		if opt != nil {
			opt(&(c.co))
		}
	}

	// make a pool to base the cluster on
	for _, addr := range clusterAddrs {
		p, err := c.co.pf("tcp", addr)
		if err != nil {
			continue
		}
		c.pools[addr] = p
		break
	}

	if err := c.Sync(); err != nil {
		for _, p := range c.pools {
			p.Close()
		}
		return nil, err
	}

	c.syncEvery(c.co.syncEvery)

	return c, nil
}

func (c *Cluster) err(err error) {
	select {
	case c.ErrCh <- err:
	default:
	}
}

func assertKeysSlot(keys []string) error {
	var ok bool
	var prevKey string
	var slot uint16
	for _, key := range keys {
		thisSlot := ClusterSlot([]byte(key))
		if !ok {
			ok = true
		} else if slot != thisSlot {
			return fmt.Errorf("keys %q and %q do not belong to the same slot", prevKey, key)
		}
		prevKey = key
		slot = thisSlot
	}
	return nil
}

// may return nil, nil if no pool for the addr
func (c *Cluster) rpool(addr string) (Client, error) {
	c.l.RLock()
	defer c.l.RUnlock()
	if addr == "" {
		for _, p := range c.pools {
			return p, nil
		}
		return nil, errors.New("no pools available")
	} else if p, ok := c.pools[addr]; ok {
		return p, nil
	}
	return nil, nil
}

var errUnknownAddress = errors.New("unknown address")

// Client returns a Client for the given address, which could be either the
// primary or one of the secondaries (see Topo method for retrieving known
// addresses).
//
// NOTE that if there is a failover while a Client returned by this method is
// being used the Client may or may not continue to work as expected, depending
// on the nature of the failover.
//
// NOTE the Client should _not_ be closed.
func (c *Cluster) Client(addr string) (Client, error) {
	// rpool allows the address to be "", handle that case manually
	if addr == "" {
		return nil, errUnknownAddress
	}
	cl, err := c.rpool(addr)
	if err != nil {
		return nil, err
	} else if cl == nil {
		return nil, errUnknownAddress
	}
	return cl, nil
}

// if addr is "" returns a random pool. If addr is given but there's no pool for
// it one will be created on-the-fly
func (c *Cluster) pool(addr string) (Client, error) {
	p, err := c.rpool(addr)
	if p != nil || err != nil {
		return p, err
	}

	// if the pool isn't available make it on-the-fly. This behavior isn't
	// _great_, but theoretically the syncEvery process should clean up any
	// extraneous pools which aren't really needed

	// it's important that the cluster pool set isn't locked while this is
	// happening, because this could block for a while
	if p, err = c.co.pf("tcp", addr); err != nil {
		return nil, err
	}

	// we've made a new pool, but we need to double-check someone else didn't
	// make one at the same time and add it in first. If they did, close this
	// one and return that one
	c.l.Lock()
	if p2, ok := c.pools[addr]; ok {
		c.l.Unlock()
		p.Close()
		return p2, nil
	}
	c.pools[addr] = p
	c.l.Unlock()
	return p, nil
}

// Topo returns the Cluster's topology as it currently knows it. See
// ClusterTopo's docs for more on its default order.
func (c *Cluster) Topo() ClusterTopo {
	c.l.RLock()
	defer c.l.RUnlock()
	return c.topo
}

func (c *Cluster) getTopo(p Client) (ClusterTopo, error) {
	var tt ClusterTopo
	err := p.Do(Cmd(&tt, "CLUSTER", "SLOTS"))
	return tt, err
}

// Sync will synchronize the Cluster with the actual cluster, making new pools
// to new instances and removing ones from instances no longer in the cluster.
// This will be called periodically automatically, but you can manually call it
// at any time as well
func (c *Cluster) Sync() error {
	p, err := c.pool("")
	if err != nil {
		return err
	}
	c.syncDedupe.do(func() {
		err = c.sync(p)
	})
	return err
}

func (c *Cluster) traceTopoChanged(prevTopo ClusterTopo, newTopo ClusterTopo) {
	if c.co.ct.TopoChanged != nil {
		var addedNodes []trace.ClusterNodeInfo
		var removedNodes []trace.ClusterNodeInfo
		var changedNodes []trace.ClusterNodeInfo

		prevTopoMap := prevTopo.Map()
		newTopoMap := newTopo.Map()

		for addr, newNode := range newTopoMap {
			if prevNode, ok := prevTopoMap[addr]; ok {
				// Check whether two nodes which have the same address changed its value or not
				if !reflect.DeepEqual(prevNode, newNode) {
					changedNodes = append(changedNodes, trace.ClusterNodeInfo{
						Addr:      newNode.Addr,
						Slots:     newNode.Slots,
						IsPrimary: newNode.SecondaryOfAddr == "",
					})
				}
				// No need to handle this address for finding removed nodes
				delete(prevTopoMap, addr)
			} else {
				// The node's address not found from prevTopo is newly added node
				addedNodes = append(addedNodes, trace.ClusterNodeInfo{
					Addr:      newNode.Addr,
					Slots:     newNode.Slots,
					IsPrimary: newNode.SecondaryOfAddr == "",
				})
			}
		}

		// Find removed nodes, prevTopoMap has reduced
		for addr, prevNode := range prevTopoMap {
			if _, ok := newTopoMap[addr]; !ok {
				removedNodes = append(removedNodes, trace.ClusterNodeInfo{
					Addr:      prevNode.Addr,
					Slots:     prevNode.Slots,
					IsPrimary: prevNode.SecondaryOfAddr == "",
				})
			}
		}

		// Callback when any changes detected
		if len(addedNodes) != 0 || len(removedNodes) != 0 || len(changedNodes) != 0 {
			c.co.ct.TopoChanged(trace.ClusterTopoChanged{
				Added:   addedNodes,
				Removed: removedNodes,
				Changed: changedNodes,
			})
		}
	}
}

// while this method is normally deduplicated by the Sync method's use of
// dedupe it is perfectly thread-safe on its own and can be used whenever.
func (c *Cluster) sync(p Client) error {
	tt, err := c.getTopo(p)
	if err != nil {
		return err
	}

	for _, t := range tt {
		// call pool just to ensure one exists for this addr
		if _, err := c.pool(t.Addr); err != nil {
			return fmt.Errorf("error connecting to %s: %s", t.Addr, err)
		}
	}

	c.traceTopoChanged(c.topo, tt)

	// this is a big bit of code to totally lockdown the cluster for, but at the
	// same time Close _shouldn't_ block significantly
	c.l.Lock()
	defer c.l.Unlock()
	c.topo = tt
	c.primTopo = tt.Primaries()

	tm := tt.Map()
	for addr, p := range c.pools {
		if _, ok := tm[addr]; !ok {
			p.Close()
			delete(c.pools, addr)
		}
	}

	return nil
}

func (c *Cluster) syncEvery(d time.Duration) {
	c.closeWG.Add(1)
	go func() {
		defer c.closeWG.Done()
		t := time.NewTicker(d)
		defer t.Stop()

		for {
			select {
			case <-t.C:
				if err := c.Sync(); err != nil {
					c.err(err)
				}
			case <-c.closeCh:
				return
			}
		}
	}()
}

func (c *Cluster) addrForKey(key string) string {
	s := ClusterSlot([]byte(key))
	c.l.RLock()
	defer c.l.RUnlock()
	for _, t := range c.primTopo {
		for _, slot := range t.Slots {
			if s >= slot[0] && s < slot[1] {
				return t.Addr
			}
		}
	}
	return ""
}

type askConn struct {
	Conn
}

func (ac askConn) Encode(m resp.Marshaler) error {
	if err := ac.Conn.Encode(Cmd(nil, "ASKING")); err != nil {
		return err
	}
	return ac.Conn.Encode(m)
}

func (ac askConn) Decode(um resp.Unmarshaler) error {
	if err := ac.Conn.Decode(resp2.Any{}); err != nil {
		return err
	}
	return ac.Conn.Decode(um)
}

func (ac askConn) Do(a Action) error {
	return a.Run(ac)
}

const doAttempts = 5

// Do performs an Action on a redis instance in the cluster, with the instance
// being determeined by the key returned from the Action's Key() method.
//
// This method handles MOVED and ASK errors automatically in most cases, see
// ClusterCanRetryAction's docs for more.
func (c *Cluster) Do(a Action) error {
	var addr, key string
	keys := a.Keys()
	if len(keys) == 0 {
		// that's ok, key will then just be ""
	} else if err := assertKeysSlot(keys); err != nil {
		return err
	} else {
		key = keys[0]
		addr = c.addrForKey(key)
	}

	return c.doInner(a, addr, key, false, doAttempts)
}

func (c *Cluster) traceRedirected(addr, key string, moved, ask bool, attempts int) {
	if c.co.ct.Redirected != nil {
		c.co.ct.Redirected(trace.ClusterRedirected{Addr: addr, Key: key, Moved: moved, Ask: ask, RedirectCount: doAttempts - attempts})
	}
}

func (c *Cluster) doInner(a Action, addr, key string, ask bool, attempts int) error {
	p, err := c.pool(addr)
	if err != nil {
		return err
	}

	// We only need to use WithConn if we want to send an ASKING command before
	// our Action a. If ask is false we can thus skip the WithConn call, which
	// avoids a few allocations, and execute our Action directly on p. This
	// helps with most calls since ask will only be true when a key gets
	// migrated between nodes.
	thisA := a
	if ask {
		thisA = WithConn(key, func(conn Conn) error {
			return askConn{conn}.Do(a)
		})
	}

	err = p.Do(thisA)
	if err == nil {
		return nil
	}

	// if the error was a MOVED or ASK we can potentially retry
	msg := err.Error()
	moved := strings.HasPrefix(msg, "MOVED ")
	ask = strings.HasPrefix(msg, "ASK ")
	if !moved && !ask {
		return err
	}
	c.traceRedirected(addr, key, moved, ask, attempts)

	// if we get an ASK there's no need to do a sync quite yet, we can continue
	// normally. But MOVED always prompts a sync. In the section after this one
	// we figure out what address to use based on the returned error so the sync
	// isn't used _immediately_, but it still needs to happen.
	//
	// Also, even if the Action isn't a ClusterCanRetryAction we want a MOVED to
	// prompt a Sync
	if moved {
		if serr := c.Sync(); serr != nil {
			return serr
		}
	}

	if ccra, ok := a.(ClusterCanRetryAction); !ok || !ccra.ClusterCanRetry() {
		return err
	}

	msgParts := strings.Split(msg, " ")
	if len(msgParts) < 3 {
		return fmt.Errorf("malformed MOVED/ASK error %q", msg)
	}
	addr = msgParts[2]

	if attempts--; attempts <= 0 {
		return errors.New("cluster action redirected too many times")
	}

	return c.doInner(a, addr, key, ask, attempts)
}

// Close cleans up all goroutines spawned by Cluster and closes all of its
// Pools.
func (c *Cluster) Close() error {
	closeErr := errClientClosed
	c.closeOnce.Do(func() {
		close(c.closeCh)
		c.closeWG.Wait()
		close(c.ErrCh)

		c.l.Lock()
		defer c.l.Unlock()
		var pErr error
		for _, p := range c.pools {
			if err := p.Close(); pErr == nil && err != nil {
				pErr = err
			}
		}
		closeErr = pErr
	})
	return closeErr
}
