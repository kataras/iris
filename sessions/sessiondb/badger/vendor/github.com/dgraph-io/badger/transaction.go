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
	"container/heap"
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/dgraph-io/badger/y"
	farm "github.com/dgryski/go-farm"
	"github.com/pkg/errors"
)

type uint64Heap []uint64

func (u uint64Heap) Len() int               { return len(u) }
func (u uint64Heap) Less(i int, j int) bool { return u[i] < u[j] }
func (u uint64Heap) Swap(i int, j int)      { u[i], u[j] = u[j], u[i] }
func (u *uint64Heap) Push(x interface{})    { *u = append(*u, x.(uint64)) }
func (u *uint64Heap) Pop() interface{} {
	old := *u
	n := len(old)
	x := old[n-1]
	*u = old[0 : n-1]
	return x
}

type oracle struct {
	sync.Mutex
	curRead    uint64
	nextCommit uint64

	// These two structures are used to figure out when a commit is done. The minimum done commit is
	// used to update curRead.
	commitMark     uint64Heap
	pendingCommits map[uint64]struct{}

	// commits stores a key fingerprint and latest commit counter for it.
	// refCount is used to clear out commits map to avoid a memory blowup.
	commits  map[uint64]uint64
	refCount int64
}

func (o *oracle) addRef() {
	atomic.AddInt64(&o.refCount, 1)
}

func (o *oracle) decrRef() {
	if count := atomic.AddInt64(&o.refCount, -1); count == 0 {
		// Clear out pendingCommits maps to release memory.
		o.Lock()
		y.AssertTrue(len(o.commitMark) == 0)
		y.AssertTrue(len(o.pendingCommits) == 0)
		if len(o.commits) >= 1000 { // If the map is still small, let it slide.
			o.commits = make(map[uint64]uint64)
		}
		o.Unlock()
	}
}

func (o *oracle) readTs() uint64 {
	return atomic.LoadUint64(&o.curRead)
}

func (o *oracle) commitTs() uint64 {
	o.Lock()
	defer o.Unlock()
	return o.nextCommit
}

// hasConflict must be called while having a lock.
func (o *oracle) hasConflict(txn *Txn) bool {
	if len(txn.reads) == 0 {
		return false
	}
	for _, ro := range txn.reads {
		if ts, has := o.commits[ro]; has && ts > txn.readTs {
			return true
		}
	}
	return false
}

func (o *oracle) newCommitTs(txn *Txn) uint64 {
	o.Lock()
	defer o.Unlock()

	if o.hasConflict(txn) {
		return 0
	}

	var ts uint64
	if txn.commitTs == 0 {
		// This is the general case, when user doesn't specify the read and commit ts.
		ts = o.nextCommit
		o.nextCommit++

	} else {
		// If commitTs is set, use it instead.
		ts = txn.commitTs
		if o.nextCommit <= ts { // Update this to max+1 commit ts, so replay works.
			o.nextCommit = ts + 1
		}
	}

	for _, w := range txn.writes {
		o.commits[w] = ts // Update the commitTs.
	}
	heap.Push(&o.commitMark, ts)
	if _, has := o.pendingCommits[ts]; has {
		panic(fmt.Sprintf("We shouldn't have the commit ts: %d", ts))
	}
	o.pendingCommits[ts] = struct{}{}
	return ts
}

func (o *oracle) doneCommit(cts uint64) {
	o.Lock()
	defer o.Unlock()

	if _, has := o.pendingCommits[cts]; !has {
		panic(fmt.Sprintf("We should already have the commit ts: %d", cts))
	}
	delete(o.pendingCommits, cts)

	var min uint64
	for len(o.commitMark) > 0 {
		ts := o.commitMark[0]
		if _, has := o.pendingCommits[ts]; has {
			// Still waiting for a txn to commit.
			break
		}
		min = ts
		heap.Pop(&o.commitMark)
	}
	if min == 0 {
		return
	}
	atomic.StoreUint64(&o.curRead, min)
	// nextCommit must never be reset.
}

// Txn represents a Badger transaction.
type Txn struct {
	readTs   uint64
	commitTs uint64

	update bool     // update is used to conditionally keep track of reads.
	reads  []uint64 // contains fingerprints of keys read.
	writes []uint64 // contains fingerprints of keys written.

	pendingWrites map[string]*entry // cache stores any writes done by txn.

	db        *DB
	callbacks []func()
	discarded bool
}

// Set sets the provided value for a given key. If key is not present, it is created.
//
// Along with key and value, Set can also take an optional userMeta byte. This byte is stored
// alongside the key, and can be used as an aid to interpret the value or store other contextual
// bits corresponding to the key-value pair.
//
// This would fail with ErrReadOnlyTxn if update flag was set to false when creating the
// transaction.
func (txn *Txn) Set(key, val []byte, userMeta byte) error {
	if !txn.update {
		return ErrReadOnlyTxn
	} else if txn.discarded {
		return ErrDiscardedTxn
	} else if len(key) == 0 {
		return ErrEmptyKey
	} else if len(key) > maxKeySize {
		return exceedsMaxKeySizeError(key)
	} else if int64(len(val)) > txn.db.opt.ValueLogFileSize {
		return exceedsMaxValueSizeError(val, txn.db.opt.ValueLogFileSize)
	}

	fp := farm.Fingerprint64(key) // Avoid dealing with byte arrays.
	txn.writes = append(txn.writes, fp)

	e := &entry{
		Key:      key,
		Value:    val,
		UserMeta: userMeta,
	}
	txn.pendingWrites[string(key)] = e
	return nil
}

// Delete deletes a key. This is done by adding a delete marker for the key at commit timestamp.
// Any reads happening before this timestamp would be unaffected. Any reads after this commit would
// see the deletion.
func (txn *Txn) Delete(key []byte) error {
	if !txn.update {
		return ErrReadOnlyTxn
	} else if txn.discarded {
		return ErrDiscardedTxn
	} else if len(key) == 0 {
		return ErrEmptyKey
	} else if len(key) > maxKeySize {
		return exceedsMaxKeySizeError(key)
	}

	fp := farm.Fingerprint64(key) // Avoid dealing with byte arrays.
	txn.writes = append(txn.writes, fp)

	e := &entry{
		Key:  key,
		Meta: bitDelete,
	}
	txn.pendingWrites[string(key)] = e
	return nil
}

// Get looks for key and returns corresponding Item.
// If key is not found, ErrKeyNotFound is returned.
func (txn *Txn) Get(key []byte) (item *Item, rerr error) {
	if len(key) == 0 {
		return nil, ErrEmptyKey
	} else if txn.discarded {
		return nil, ErrDiscardedTxn
	}

	item = new(Item)
	if txn.update {
		if e, has := txn.pendingWrites[string(key)]; has && bytes.Compare(key, e.Key) == 0 {
			// Fulfill from cache.
			item.meta = e.Meta
			item.val = e.Value
			item.userMeta = e.UserMeta
			item.key = key
			item.status = prefetched
			item.version = txn.readTs
			// We probably don't need to set db on item here.
			return item, nil
		}
		// Only track reads if this is update txn. No need to track read if txn serviced it
		// internally.
		fp := farm.Fingerprint64(key)
		txn.reads = append(txn.reads, fp)
	}

	seek := y.KeyWithTs(key, txn.readTs)
	vs, err := txn.db.get(seek)
	if err != nil {
		return nil, errors.Wrapf(err, "DB::Get key: %q", key)
	}
	if vs.Value == nil && vs.Meta == 0 {
		return nil, ErrKeyNotFound
	}
	if (vs.Meta & bitDelete) != 0 {
		return nil, ErrKeyNotFound
	}

	item.key = key
	item.version = vs.Version
	item.meta = vs.Meta
	item.userMeta = vs.UserMeta
	item.db = txn.db
	item.vptr = vs.Value
	item.txn = txn
	return item, nil
}

// Discard discards a created transaction. This method is very important and must be called. Commit
// method calls this internally, however, calling this multiple times doesn't cause any issues. So,
// this can safely be called via a defer right when transaction is created.
//
// NOTE: If any operations are run on a discarded transaction, ErrDiscardedTxn is returned.
func (txn *Txn) Discard() {
	if txn.discarded { // Avoid a re-run.
		return
	}
	txn.discarded = true

	for _, cb := range txn.callbacks {
		cb()
	}
	if txn.update {
		txn.db.orc.decrRef()
	}
}

// Commit commits the transaction, following these steps:
//
// 1. If there are no writes, return immediately.
//
// 2. Check if read rows were updated since txn started. If so, return ErrConflict.
//
// 3. If no conflict, generate a commit timestamp and update written rows' commit ts.
//
// 4. Batch up all writes, write them to value log and LSM tree.
//
// 5. If callback is provided, Badger will return immediately after checking
// for conflicts. Writes to the database will happen in the background.  If
// there is a conflict, an error will be returned and the callback will not
// run. If there are no conflicts, the callback will be called in the
// background upon successful completion of writes or any error during write.
//
// If error is nil, the transaction is successfully committed. In case of a non-nil error, the LSM
// tree won't be updated, so there's no need for any rollback.
func (txn *Txn) Commit(callback func(error)) error {
	if txn.discarded {
		return ErrDiscardedTxn
	}
	defer txn.Discard()
	if len(txn.writes) == 0 {
		return nil // Nothing to do.
	}

	state := txn.db.orc
	commitTs := state.newCommitTs(txn)
	if commitTs == 0 {
		return ErrConflict
	}
	defer state.doneCommit(commitTs)

	entries := make([]*entry, 0, len(txn.pendingWrites)+1)
	for _, e := range txn.pendingWrites {
		// Suffix the keys with commit ts, so the key versions are sorted in
		// descending order of commit timestamp.
		e.Key = y.KeyWithTs(e.Key, commitTs)
		e.Meta |= bitTxn
		entries = append(entries, e)
	}
	e := &entry{
		Key:   y.KeyWithTs(txnKey, commitTs),
		Value: []byte(strconv.FormatUint(commitTs, 10)),
		Meta:  bitFinTxn,
	}
	entries = append(entries, e)

	if callback == nil {
		// If batchSet failed, LSM would not have been updated. So, no need to rollback anything.

		// TODO: What if some of the txns successfully make it to value log, but others fail.
		// Nothing gets updated to LSM, until a restart happens.
		return txn.db.batchSet(entries)
	}
	return txn.db.batchSetAsync(entries, callback)
}

// CommitAt commits the transaction, following the same logic as Commit(), but at the given
// commit timestamp. This API is only useful for databases built on top of Badger (like Dgraph), and
// can be ignored by most users.
func (txn *Txn) CommitAt(commitTs uint64, callback func(error)) error {
	txn.commitTs = commitTs
	return txn.Commit(callback)
}

// NewTransaction creates a new transaction. Badger supports concurrent execution of transactions,
// providing serializable snapshot isolation, avoiding write skews. Badger achieves this by tracking
// the keys read and at Commit time, ensuring that these read keys weren't concurrently modified by
// another transaction.
//
// For read-only transactions, set update to false. In this mode, we don't track the rows read for
// any changes. Thus, any long running iterations done in this mode wouldn't pay this overhead.
//
// Running transactions concurrently is OK. However, a transaction itself isn't thread safe, and
// should only be run serially. It doesn't matter if a transaction is created by one goroutine and
// passed down to other, as long as the Txn APIs are called serially.
//
// When you create a new transaction, it is absolutely essential to call
// Discard(). This should be done irrespective of what the update param is set
// to. Commit API internally runs Discard, but running it twice wouldn't cause
// any issues.
//
//  txn := db.NewTransaction(false)
//  defer txn.Discard()
//  // Call various APIs.
func (db *DB) NewTransaction(update bool) *Txn {
	txn := &Txn{
		update: update,
		db:     db,
		readTs: db.orc.readTs(),
	}
	if update {
		txn.pendingWrites = make(map[string]*entry)
		txn.db.orc.addRef()
	}
	return txn
}

// NewTransactionAt follows the same logic as NewTransaction, but uses the provided read timestamp.
// This API is only useful for databases built on top of Badger (like Dgraph), and can be ignored by
// most users.
func (db *DB) NewTransactionAt(readTs uint64, update bool) *Txn {
	txn := db.NewTransaction(update)
	txn.readTs = readTs
	return txn
}

// View executes a function creating and managing a read-only transaction for the user. Error
// returned by the function is relayed by the View method.
func (db *DB) View(fn func(txn *Txn) error) error {
	txn := db.NewTransaction(false)
	defer txn.Discard()

	return fn(txn)
}

// Update executes a function, creating and managing a read-write transaction
// for the user. Error returned by the function is relayed by the Update method.
func (db *DB) Update(fn func(txn *Txn) error) error {
	txn := db.NewTransaction(true)
	defer txn.Discard()

	if err := fn(txn); err != nil {
		return err
	}

	return txn.Commit(nil)
}
