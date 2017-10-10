/*
 * Copyright 2017 Dgraph Labs, Inc. and Contributors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package badger

import (
	"bytes"
	"fmt"
	"sync"

	"github.com/dgraph-io/badger/y"
	farm "github.com/dgryski/go-farm"
)

type prefetchStatus uint8

const (
	empty prefetchStatus = iota
	prefetched
)

// Item is returned during iteration. Both the Key() and Value() output is only valid until
// iterator.Next() is called.
type Item struct {
	status   prefetchStatus
	err      error
	wg       sync.WaitGroup
	db       *DB
	key      []byte
	vptr     []byte
	meta     byte
	userMeta byte
	val      []byte
	slice    *y.Slice // Used only during prefetching.
	next     *Item
	version  uint64
	txn      *Txn
}

// ToString returns a string representation of Item
func (item *Item) ToString() string {
	return fmt.Sprintf("key=%q, version=%d, meta=%x", item.Key(), item.Version(), item.meta)

}

// Key returns the key.
//
// Key is only valid as long as item is valid, or transaction is valid.  If you need to use it
// outside its validity, please copy it.
func (item *Item) Key() []byte {
	return item.key
}

// Version returns the commit timestamp of the item.
func (item *Item) Version() uint64 {
	return item.version
}

// Value retrieves the value of the item from the value log.
//
// The returned value is only valid as long as item is valid, or transaction is valid. So, if you
// need to use it outside, please parse or copy it.
func (item *Item) Value() ([]byte, error) {
	item.wg.Wait()
	if item.status == prefetched {
		return item.val, item.err
	}
	buf, cb, err := item.yieldItemValue()
	if cb != nil {
		item.txn.callbacks = append(item.txn.callbacks, cb)
	}
	return buf, err
}

func (item *Item) hasValue() bool {
	if item.meta == 0 && item.vptr == nil {
		// key not found
		return false
	}
	if (item.meta & bitDelete) != 0 {
		// Tombstone encountered.
		return false
	}
	return true
}

func (item *Item) yieldItemValue() ([]byte, func(), error) {
	if !item.hasValue() {
		return nil, nil, nil
	}

	if item.slice == nil {
		item.slice = new(y.Slice)
	}

	if (item.meta & bitValuePointer) == 0 {
		val := item.slice.Resize(len(item.vptr))
		copy(val, item.vptr)
		return val, nil, nil
	}

	var vp valuePointer
	vp.Decode(item.vptr)
	return item.db.vlog.Read(vp)
}

func runCallback(cb func()) {
	if cb != nil {
		cb()
	}
}

func (item *Item) prefetchValue() {
	val, cb, err := item.yieldItemValue()
	defer runCallback(cb)

	item.err = err
	item.status = prefetched
	if val == nil {
		return
	}
	buf := item.slice.Resize(len(val))
	copy(buf, val)
	item.val = buf
}

// EstimatedSize returns approximate size of the key-value pair.
//
// This can be called while iterating through a store to quickly estimate the
// size of a range of key-value pairs (without fetching the corresponding
// values).
func (item *Item) EstimatedSize() int64 {
	if !item.hasValue() {
		return 0
	}
	if (item.meta & bitValuePointer) == 0 {
		return int64(len(item.key) + len(item.vptr))
	}
	var vp valuePointer
	vp.Decode(item.vptr)
	return int64(vp.Len) // includes key length.
}

// UserMeta returns the userMeta set by the user. Typically, this byte, optionally set by the user
// is used to interpret the value.
func (item *Item) UserMeta() byte {
	return item.userMeta
}

// TODO: Switch this to use linked list container in Go.
type list struct {
	head *Item
	tail *Item
}

func (l *list) push(i *Item) {
	i.next = nil
	if l.tail == nil {
		l.head = i
		l.tail = i
		return
	}
	l.tail.next = i
	l.tail = i
}

func (l *list) pop() *Item {
	if l.head == nil {
		return nil
	}
	i := l.head
	if l.head == l.tail {
		l.tail = nil
		l.head = nil
	} else {
		l.head = i.next
	}
	i.next = nil
	return i
}

// IteratorOptions is used to set options when iterating over Badger key-value
// stores.
//
// This package provides DefaultIteratorOptions which contains options that
// should work for most applications. Consider using that as a starting point
// before customizing it for your own needs.
type IteratorOptions struct {
	// Indicates whether we should prefetch values during iteration and store them.
	PrefetchValues bool
	// How many KV pairs to prefetch while iterating. Valid only if PrefetchValues is true.
	PrefetchSize int
	Reverse      bool // Direction of iteration. False is forward, true is backward.
	AllVersions  bool // Fetch all valid versions of the same key.
}

// DefaultIteratorOptions contains default options when iterating over Badger key-value stores.
var DefaultIteratorOptions = IteratorOptions{
	PrefetchValues: true,
	PrefetchSize:   100,
	Reverse:        false,
	AllVersions:    false,
}

// Iterator helps iterating over the KV pairs in a lexicographically sorted order.
type Iterator struct {
	iitr   *y.MergeIterator
	txn    *Txn
	readTs uint64

	opt   IteratorOptions
	item  *Item
	data  list
	waste list

	lastKey []byte // Used to skip over multiple versions of the same key.
}

// NewIterator returns a new iterator. Depending upon the options, either only keys, or both
// key-value pairs would be fetched. The keys are returned in lexicographically sorted order.
// Using prefetch is highly recommended if you're doing a long running iteration.
// Avoid long running iterations in update transactions.
func (txn *Txn) NewIterator(opt IteratorOptions) *Iterator {
	tables, decr := txn.db.getMemTables()
	defer decr()
	txn.db.vlog.incrIteratorCount()
	var iters []y.Iterator
	for i := 0; i < len(tables); i++ {
		iters = append(iters, tables[i].NewUniIterator(opt.Reverse))
	}
	iters = txn.db.lc.appendIterators(iters, opt.Reverse) // This will increment references.
	res := &Iterator{
		txn:    txn,
		iitr:   y.NewMergeIterator(iters, opt.Reverse),
		opt:    opt,
		readTs: txn.readTs,
	}
	return res
}

func (it *Iterator) newItem() *Item {
	item := it.waste.pop()
	if item == nil {
		item = &Item{slice: new(y.Slice), db: it.txn.db, txn: it.txn}
	}
	return item
}

// Item returns pointer to the current key-value pair.
// This item is only valid until it.Next() gets called.
func (it *Iterator) Item() *Item {
	tx := it.txn
	if tx.update {
		// Track reads if this is an update txn.
		tx.reads = append(tx.reads, farm.Fingerprint64(it.item.Key()))
	}
	return it.item
}

// Valid returns false when iteration is done.
func (it *Iterator) Valid() bool { return it.item != nil }

// ValidForPrefix returns false when iteration is done
// or when the current key is not prefixed by the specified prefix.
func (it *Iterator) ValidForPrefix(prefix []byte) bool {
	return it.item != nil && bytes.HasPrefix(it.item.key, prefix)
}

// Close would close the iterator. It is important to call this when you're done with iteration.
func (it *Iterator) Close() {
	it.iitr.Close()
	// TODO: We could handle this error.
	_ = it.txn.db.vlog.decrIteratorCount()
}

// Next would advance the iterator by one. Always check it.Valid() after a Next()
// to ensure you have access to a valid it.Item().
func (it *Iterator) Next() {
	// Reuse current item
	it.item.wg.Wait() // Just cleaner to wait before pushing to avoid doing ref counting.
	it.waste.push(it.item)

	// Set next item to current
	it.item = it.data.pop()

	for it.iitr.Valid() {
		if it.parseItem() {
			// parseItem calls one extra next.
			// This is used to deal with the complexity of reverse iteration.
			break
		}
	}
}

// parseItem is a complex function because it needs to handle both forward and reverse iteration
// implementation. We store keys such that their versions are sorted in descending order. This makes
// forward iteration efficient, but revese iteration complicated. This tradeoff is better because
// forward iteration is more common than reverse.
//
// This function advances the iterator.
func (it *Iterator) parseItem() bool {
	mi := it.iitr
	key := mi.Key()

	setItem := func(item *Item) {
		if it.item == nil {
			it.item = item
		} else {
			it.data.push(item)
		}
	}

	// Skip badger keys.
	if bytes.HasPrefix(key, badgerPrefix) {
		mi.Next()
		return false
	}

	// Skip any versions which are beyond the readTs.
	version := y.ParseTs(key)
	if version > it.readTs {
		mi.Next()
		return false
	}

	if it.opt.AllVersions {
		item := it.newItem()
		it.fill(item)
		setItem(item)
		mi.Next()
		return true
	}

	// If iterating in forward direction, then just checking the last key against current key would
	// be sufficient.
	if !it.opt.Reverse {
		if y.SameKey(it.lastKey, key) {
			mi.Next()
			return false
		}
		// Only track in forward direction.
		// We should update lastKey as soon as we find a different key in our snapshot.
		// Consider keys: a 5, b 7 (del), b 5. When iterating, lastKey = a.
		// Then we see b 7, which is deleted. If we don't store lastKey = b, we'll then return b 5,
		// which is wrong. Therefore, update lastKey here.
		it.lastKey = y.Safecopy(it.lastKey, mi.Key())
	}

FILL:
	// If deleted, advance and return.
	if mi.Value().Meta&bitDelete > 0 {
		mi.Next()
		return false
	}

	item := it.newItem()
	it.fill(item)
	// fill item based on current cursor position. All Next calls have returned, so reaching here
	// means no Next was called.

	mi.Next()                           // Advance but no fill item yet.
	if !it.opt.Reverse || !mi.Valid() { // Forward direction, or invalid.
		setItem(item)
		return true
	}

	// Reverse direction.
	nextTs := y.ParseTs(mi.Key())
	mik := y.ParseKey(mi.Key())
	if nextTs <= it.readTs && bytes.Compare(mik, item.key) == 0 {
		// This is a valid potential candidate.
		goto FILL
	}
	// Ignore the next candidate. Return the current one.
	setItem(item)
	return true
}

func (it *Iterator) fill(item *Item) {
	vs := it.iitr.Value()
	item.meta = vs.Meta
	item.userMeta = vs.UserMeta

	item.version = y.ParseTs(it.iitr.Key())
	item.key = y.Safecopy(item.key, y.ParseKey(it.iitr.Key()))

	item.vptr = y.Safecopy(item.vptr, vs.Value)
	item.val = nil
	if it.opt.PrefetchValues {
		item.wg.Add(1)
		go func() {
			// FIXME we are not handling errors here.
			item.prefetchValue()
			item.wg.Done()
		}()
	}
}

func (it *Iterator) prefetch() {
	prefetchSize := 2
	if it.opt.PrefetchValues && it.opt.PrefetchSize > 1 {
		prefetchSize = it.opt.PrefetchSize
	}

	i := it.iitr
	var count int
	it.item = nil
	for i.Valid() {
		if !it.parseItem() {
			continue
		}
		count++
		if count == prefetchSize {
			break
		}
	}
}

// Seek would seek to the provided key if present. If absent, it would seek to the next smallest key
// greater than provided if iterating in the forward direction. Behavior would be reversed is
// iterating backwards.
func (it *Iterator) Seek(key []byte) {
	for i := it.data.pop(); i != nil; i = it.data.pop() {
		i.wg.Wait()
		it.waste.push(i)
	}

	it.lastKey = it.lastKey[:0]
	if len(key) == 0 {
		it.iitr.Rewind()
		it.prefetch()
		return
	}

	if !it.opt.Reverse {
		key = y.KeyWithTs(key, it.txn.readTs)
	} else {
		key = y.KeyWithTs(key, 0)
	}
	it.iitr.Seek(key)
	it.prefetch()
}

// Rewind would rewind the iterator cursor all the way to zero-th position, which would be the
// smallest key if iterating forward, and largest if iterating backward. It does not keep track of
// whether the cursor started with a Seek().
func (it *Iterator) Rewind() {
	i := it.data.pop()
	for i != nil {
		i.wg.Wait() // Just cleaner to wait before pushing. No ref counting needed.
		it.waste.push(i)
		i = it.data.pop()
	}

	it.lastKey = it.lastKey[:0]
	it.iitr.Rewind()
	it.prefetch()
}
