package radix

import (
	"strings"
)

type clusterScanner struct {
	cluster *Cluster
	opts    ScanOpts

	addrs       []string
	currScanner Scanner
	lastErr     error
}

// NewScanner will return a Scanner which will scan over every node in the
// cluster. This will panic if the ScanOpt's Command isn't "SCAN".
//
// If the cluster topology changes during a scan the Scanner may or may not
// error out due to it, depending on the nature of the change.
func (c *Cluster) NewScanner(o ScanOpts) Scanner {
	if strings.ToUpper(o.Command) != "SCAN" {
		panic("Cluster.NewScanner can only perform SCAN operations")
	}

	var addrs []string
	for _, node := range c.Topo().Primaries() {
		addrs = append(addrs, node.Addr)
	}

	cs := &clusterScanner{
		cluster: c,
		opts:    o,
		addrs:   addrs,
	}
	cs.nextScanner()

	return cs
}

func (cs *clusterScanner) closeCurr() {
	if cs.currScanner != nil {
		if err := cs.currScanner.Close(); err != nil && cs.lastErr == nil {
			cs.lastErr = err
		}
		cs.currScanner = nil
	}
}

func (cs *clusterScanner) scannerForAddr(addr string) bool {
	client, _ := cs.cluster.rpool(addr)
	if client != nil {
		cs.closeCurr()
		cs.currScanner = NewScanner(client, cs.opts)
		return true
	}
	return false
}

func (cs *clusterScanner) nextScanner() {
	for {
		if len(cs.addrs) == 0 {
			cs.closeCurr()
			return
		}
		addr := cs.addrs[0]
		cs.addrs = cs.addrs[1:]
		if cs.scannerForAddr(addr) {
			return
		}
	}
}

func (cs *clusterScanner) Next(res *string) bool {
	for {
		if cs.currScanner == nil {
			return false
		} else if out := cs.currScanner.Next(res); out {
			return true
		}
		cs.nextScanner()
	}
}

func (cs *clusterScanner) Close() error {
	cs.closeCurr()
	return cs.lastErr
}
