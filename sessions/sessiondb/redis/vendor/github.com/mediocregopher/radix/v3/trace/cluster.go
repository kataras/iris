// Package trace contains all the types provided for tracing within the radix
// package. With tracing a user is able to pull out fine-grained runtime events
// as they happen, which is useful for gathering metrics, logging, performance
// analysis, etc...
//
// BIG LOUD DISCLAIMER DO NOT IGNORE THIS: while the main radix package is
// stable and will always remain backwards compatible, trace is still under
// active development and may undergo changes to its types and other features.
// The methods in the main radix package which invoke trace types are guaranteed
// to remain stable.
package trace

////////////////////////////////////////////////////////////////////////////////

type ClusterTrace struct {
	// TopoChanged is called when the cluster's topology changed.
	TopoChanged func(ClusterTopoChanged)
	// Redirected is called when radix.Do responded 'MOVED' or 'ASKED'.
	Redirected func(ClusterRedirected)
}

type ClusterNodeInfo struct {
	Addr      string
	Slots     [][2]uint16
	IsPrimary bool
}

type ClusterTopoChanged struct {
	Added   []ClusterNodeInfo
	Removed []ClusterNodeInfo
	Changed []ClusterNodeInfo
}

type ClusterRedirected struct {
	Addr          string
	Key           string
	Moved, Ask    bool
	RedirectCount int
}
