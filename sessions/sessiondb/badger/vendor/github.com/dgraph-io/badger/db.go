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
	"container/heap"
	"expvar"
	"log"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"golang.org/x/net/trace"

	"github.com/dgraph-io/badger/skl"
	"github.com/dgraph-io/badger/table"
	"github.com/dgraph-io/badger/y"
	"github.com/pkg/errors"
)

var (
	badgerPrefix = []byte("!badger!")     // Prefix for internal keys used by badger.
	head         = []byte("!badger!head") // For storing value offset for replay.
	txnKey       = []byte("!badger!txn")  // For indicating end of entries in txn.
)

type closers struct {
	updateSize *y.Closer
	compactors *y.Closer
	memtable   *y.Closer
	writes     *y.Closer
	valueGC    *y.Closer
}

// DB provides the various functions required to interact with Badger.
// DB is thread-safe.
type DB struct {
	sync.RWMutex // Guards list of inmemory tables, not individual reads and writes.

	dirLockGuard *DirectoryLockGuard
	// nil if Dir and ValueDir are the same
	valueDirGuard *DirectoryLockGuard

	closers   closers
	elog      trace.EventLog
	mt        *skl.Skiplist   // Our latest (actively written) in-memory table
	imm       []*skl.Skiplist // Add here only AFTER pushing to flushChan.
	opt       Options
	manifest  *manifestFile
	lc        *levelsController
	vlog      valueLog
	vptr      valuePointer // less than or equal to a pointer to the last vlog value put into mt
	writeCh   chan *request
	flushChan chan flushTask // For flushing memtables.

	// Incremented in the non-concurrently accessed write loop.  But also accessed outside. So
	// we use an atomic op.
	lastUsedCommitTs uint64

	orc *oracle
}

const (
	kvWriteChCapacity = 1000
)

func replayFunction(out *DB) func(entry, valuePointer) error {
	type txnEntry struct {
		nk []byte
		v  y.ValueStruct
	}

	var txn []txnEntry
	var lastCommit uint64

	toLSM := func(nk []byte, vs y.ValueStruct) {
		for err := out.ensureRoomForWrite(); err != nil; err = out.ensureRoomForWrite() {
			out.elog.Printf("Replay: Making room for writes")
			time.Sleep(10 * time.Millisecond)
		}
		out.mt.Put(nk, vs)
	}

	first := true
	return func(e entry, vp valuePointer) error { // Function for replaying.
		if first {
			out.elog.Printf("First key=%s\n", e.Key)
		}
		first = false

		if out.orc.curRead < y.ParseTs(e.Key) {
			out.orc.curRead = y.ParseTs(e.Key)
		}

		nk := make([]byte, len(e.Key))
		copy(nk, e.Key)
		var nv []byte
		meta := e.Meta
		if out.shouldWriteValueToLSM(e) {
			nv = make([]byte, len(e.Value))
			copy(nv, e.Value)
		} else {
			nv = make([]byte, vptrSize)
			vp.Encode(nv)
			meta = meta | bitValuePointer
		}

		v := y.ValueStruct{
			Value:    nv,
			Meta:     meta,
			UserMeta: e.UserMeta,
		}

		if e.Meta&bitFinTxn > 0 {
			txnTs, err := strconv.ParseUint(string(e.Value), 10, 64)
			if err != nil {
				return errors.Wrapf(err, "Unable to parse txn fin: %q", e.Value)
			}
			y.AssertTrue(lastCommit == txnTs)
			y.AssertTrue(len(txn) > 0)
			// Got the end of txn. Now we can store them.
			for _, t := range txn {
				toLSM(t.nk, t.v)
			}
			txn = txn[:0]
			lastCommit = 0

		} else if e.Meta&bitTxn == 0 {
			// This entry is from a rewrite.
			toLSM(nk, v)

			// We shouldn't get this entry in the middle of a transaction.
			y.AssertTrue(lastCommit == 0)
			y.AssertTrue(len(txn) == 0)

		} else {
			txnTs := y.ParseTs(nk)
			if lastCommit == 0 {
				lastCommit = txnTs
			}
			y.AssertTrue(lastCommit == txnTs)
			te := txnEntry{nk: nk, v: v}
			txn = append(txn, te)
		}
		return nil
	}
}

// Open returns a new DB object.
func Open(opt Options) (db *DB, err error) {
	opt.maxBatchSize = (15 * opt.MaxTableSize) / 100
	opt.maxBatchCount = opt.maxBatchSize / int64(skl.MaxNodeSize)

	for _, path := range []string{opt.Dir, opt.ValueDir} {
		dirExists, err := exists(path)
		if err != nil {
			return nil, y.Wrapf(err, "Invalid Dir: %q", path)
		}
		if !dirExists {
			return nil, ErrInvalidDir
		}
	}
	absDir, err := filepath.Abs(opt.Dir)
	if err != nil {
		return nil, err
	}
	absValueDir, err := filepath.Abs(opt.ValueDir)
	if err != nil {
		return nil, err
	}

	dirLockGuard, err := AcquireDirectoryLock(opt.Dir, lockFile)
	if err != nil {
		return nil, err
	}
	defer func() {
		if dirLockGuard != nil {
			_ = dirLockGuard.Release()
		}
	}()
	var valueDirLockGuard *DirectoryLockGuard
	if absValueDir != absDir {
		valueDirLockGuard, err = AcquireDirectoryLock(opt.ValueDir, lockFile)
		if err != nil {
			return nil, err
		}
	}
	defer func() {
		if valueDirLockGuard != nil {
			_ = valueDirLockGuard.Release()
		}
	}()
	if !(opt.ValueLogFileSize <= 2<<30 && opt.ValueLogFileSize >= 1<<20) {
		return nil, ErrValueLogSize
	}
	manifestFile, manifest, err := openOrCreateManifestFile(opt.Dir)
	if err != nil {
		return nil, err
	}
	defer func() {
		if manifestFile != nil {
			_ = manifestFile.close()
		}
	}()

	orc := &oracle{
		nextCommit:     1,
		pendingCommits: make(map[uint64]struct{}),
		commits:        make(map[uint64]uint64),
	}
	heap.Init(&orc.commitMark)

	db = &DB{
		imm:           make([]*skl.Skiplist, 0, opt.NumMemtables),
		flushChan:     make(chan flushTask, opt.NumMemtables),
		writeCh:       make(chan *request, kvWriteChCapacity),
		opt:           opt,
		manifest:      manifestFile,
		elog:          trace.NewEventLog("Badger", "DB"),
		dirLockGuard:  dirLockGuard,
		valueDirGuard: valueDirLockGuard,
		orc:           orc,
	}

	db.closers.updateSize = y.NewCloser(1)
	go db.updateSize(db.closers.updateSize)
	db.mt = skl.NewSkiplist(arenaSize(&opt))

	// newLevelsController potentially loads files in directory.
	if db.lc, err = newLevelsController(db, &manifest); err != nil {
		return nil, err
	}

	db.closers.compactors = y.NewCloser(1)
	db.lc.startCompact(db.closers.compactors)

	db.closers.memtable = y.NewCloser(1)
	go db.flushMemtable(db.closers.memtable) // Need levels controller to be up.

	if err = db.vlog.Open(db, opt); err != nil {
		return nil, err
	}

	headKey := y.KeyWithTs(head, math.MaxUint64)
	// Need to pass with timestamp, lsm get removes the last 8 bytes and compares key
	vs, err := db.get(headKey)
	if err != nil {
		return nil, errors.Wrap(err, "Retrieving head")
	}
	db.orc.curRead = vs.Version
	var vptr valuePointer
	if len(vs.Value) > 0 {
		vptr.Decode(vs.Value)
	}

	// lastUsedCasCounter will either be the value stored in !badger!head, or some subsequently
	// written value log entry that we replay.  (Subsequent value log entries might be _less_
	// than lastUsedCasCounter, if there was value log gc so we have to max() values while
	// replaying.)
	// out.lastUsedCasCounter = item.casCounter
	// TODO: Figure this out. This would update the read timestamp, and set nextCommitTs.

	replayCloser := y.NewCloser(1)
	go db.doWrites(replayCloser)

	if err = db.vlog.Replay(vptr, replayFunction(db)); err != nil {
		return db, err
	}

	replayCloser.SignalAndWait() // Wait for replay to be applied first.
	// Now that we have the curRead, we can update the nextCommit.
	db.orc.nextCommit = db.orc.curRead + 1

	// Mmap writable log
	lf := db.vlog.filesMap[db.vlog.maxFid]
	if err = lf.mmap(2 * db.vlog.opt.ValueLogFileSize); err != nil {
		return db, errors.Wrapf(err, "Unable to mmap RDWR log file")
	}

	db.writeCh = make(chan *request, kvWriteChCapacity)
	db.closers.writes = y.NewCloser(1)
	go db.doWrites(db.closers.writes)

	db.closers.valueGC = y.NewCloser(1)
	go db.vlog.waitOnGC(db.closers.valueGC)

	valueDirLockGuard = nil
	dirLockGuard = nil
	manifestFile = nil
	return db, nil
}

// Close closes a DB. It's crucial to call it to ensure all the pending updates
// make their way to disk.
func (db *DB) Close() (err error) {
	db.elog.Printf("Closing database")
	// Stop value GC first.
	db.closers.valueGC.SignalAndWait()

	// Stop writes next.
	db.closers.writes.SignalAndWait()

	// Now close the value log.
	if vlogErr := db.vlog.Close(); err == nil {
		err = errors.Wrap(vlogErr, "DB.Close")
	}

	// Make sure that block writer is done pushing stuff into memtable!
	// Otherwise, you will have a race condition: we are trying to flush memtables
	// and remove them completely, while the block / memtable writer is still
	// trying to push stuff into the memtable. This will also resolve the value
	// offset problem: as we push into memtable, we update value offsets there.
	if !db.mt.Empty() {
		db.elog.Printf("Flushing memtable")
		for {
			pushedFlushTask := func() bool {
				db.Lock()
				defer db.Unlock()
				y.AssertTrue(db.mt != nil)
				select {
				case db.flushChan <- flushTask{db.mt, db.vptr}:
					db.imm = append(db.imm, db.mt) // Flusher will attempt to remove this from s.imm.
					db.mt = nil                    // Will segfault if we try writing!
					db.elog.Printf("pushed to flush chan\n")
					return true
				default:
					// If we fail to push, we need to unlock and wait for a short while.
					// The flushing operation needs to update s.imm. Otherwise, we have a deadlock.
					// TODO: Think about how to do this more cleanly, maybe without any locks.
				}
				return false
			}()
			if pushedFlushTask {
				break
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
	db.flushChan <- flushTask{nil, valuePointer{}} // Tell flusher to quit.

	db.closers.memtable.Wait()
	db.elog.Printf("Memtable flushed")

	db.closers.compactors.SignalAndWait()
	db.elog.Printf("Compaction finished")

	if lcErr := db.lc.close(); err == nil {
		err = errors.Wrap(lcErr, "DB.Close")
	}
	db.elog.Printf("Waiting for closer")
	db.closers.updateSize.SignalAndWait()

	db.elog.Finish()

	if guardErr := db.dirLockGuard.Release(); err == nil {
		err = errors.Wrap(guardErr, "DB.Close")
	}
	if db.valueDirGuard != nil {
		if guardErr := db.valueDirGuard.Release(); err == nil {
			err = errors.Wrap(guardErr, "DB.Close")
		}
	}
	if manifestErr := db.manifest.close(); err == nil {
		err = errors.Wrap(manifestErr, "DB.Close")
	}

	// Fsync directories to ensure that lock file, and any other removed files whose directory
	// we haven't specifically fsynced, are guaranteed to have their directory entry removal
	// persisted to disk.
	if syncErr := syncDir(db.opt.Dir); err == nil {
		err = errors.Wrap(syncErr, "DB.Close")
	}
	if syncErr := syncDir(db.opt.ValueDir); err == nil {
		err = errors.Wrap(syncErr, "DB.Close")
	}

	return err
}

const (
	lockFile = "LOCK"
)

// When you create or delete a file, you have to ensure the directory entry for the file is synced
// in order to guarantee the file is visible (if the system crashes).  (See the man page for fsync,
// or see https://github.com/coreos/etcd/issues/6368 for an example.)
func syncDir(dir string) error {
	f, err := OpenDir(dir)
	if err != nil {
		return errors.Wrapf(err, "While opening directory: %s.", dir)
	}
	err = f.Sync()
	closeErr := f.Close()
	if err != nil {
		return errors.Wrapf(err, "While syncing directory: %s.", dir)
	}
	return errors.Wrapf(closeErr, "While closing directory: %s.", dir)
}

// getMemtables returns the current memtables and get references.
func (db *DB) getMemTables() ([]*skl.Skiplist, func()) {
	db.RLock()
	defer db.RUnlock()

	tables := make([]*skl.Skiplist, len(db.imm)+1)

	// Get mutable memtable.
	tables[0] = db.mt
	tables[0].IncrRef()

	// Get immutable memtables.
	last := len(db.imm) - 1
	for i := range db.imm {
		tables[i+1] = db.imm[last-i]
		tables[i+1].IncrRef()
	}
	return tables, func() {
		for _, tbl := range tables {
			tbl.DecrRef()
		}
	}
}

// get returns the value in memtable or disk for given key.
// Note that value will include meta byte.
func (db *DB) get(key []byte) (y.ValueStruct, error) {
	tables, decr := db.getMemTables() // Lock should be released.
	defer decr()

	y.NumGets.Add(1)
	for i := 0; i < len(tables); i++ {
		vs := tables[i].Get(key)
		y.NumMemtableGets.Add(1)
		if vs.Meta != 0 || vs.Value != nil {
			return vs, nil
		}
	}
	return db.lc.get(key)
}

func (db *DB) updateOffset(ptrs []valuePointer) {
	var ptr valuePointer
	for i := len(ptrs) - 1; i >= 0; i-- {
		p := ptrs[i]
		if !p.IsZero() {
			ptr = p
			break
		}
	}
	if ptr.IsZero() {
		return
	}

	db.Lock()
	defer db.Unlock()
	y.AssertTrue(!ptr.Less(db.vptr))
	db.vptr = ptr
}

var requestPool = sync.Pool{
	New: func() interface{} {
		return new(request)
	},
}

func (db *DB) shouldWriteValueToLSM(e entry) bool {
	return len(e.Value) < db.opt.ValueThreshold
}

func (db *DB) writeToLSM(b *request) error {
	if len(b.Ptrs) != len(b.Entries) {
		return errors.Errorf("Ptrs and Entries don't match: %+v", b)
	}

	for i, entry := range b.Entries {
		if entry.Meta&bitFinTxn != 0 {
			continue
		}
		if db.shouldWriteValueToLSM(*entry) { // Will include deletion / tombstone case.
			db.mt.Put(entry.Key,
				y.ValueStruct{
					Value:    entry.Value,
					Meta:     entry.Meta,
					UserMeta: entry.UserMeta,
				})
		} else {
			var offsetBuf [vptrSize]byte
			db.mt.Put(entry.Key,
				y.ValueStruct{
					Value:    b.Ptrs[i].Encode(offsetBuf[:]),
					Meta:     entry.Meta | bitValuePointer,
					UserMeta: entry.UserMeta,
				})
		}
	}
	return nil
}

// writeRequests is called serially by only one goroutine.
func (db *DB) writeRequests(reqs []*request) error {
	if len(reqs) == 0 {
		return nil
	}

	done := func(err error) {
		for _, r := range reqs {
			r.Err = err
			r.Wg.Done()
		}
	}

	db.elog.Printf("writeRequests called. Writing to value log")

	err := db.vlog.write(reqs)
	if err != nil {
		done(err)
		return err
	}

	db.elog.Printf("Writing to memtable")
	var count int
	for _, b := range reqs {
		if len(b.Entries) == 0 {
			continue
		}
		count += len(b.Entries)
		for err := db.ensureRoomForWrite(); err != nil; err = db.ensureRoomForWrite() {
			db.elog.Printf("Making room for writes")
			// We need to poll a bit because both hasRoomForWrite and the flusher need access to s.imm.
			// When flushChan is full and you are blocked there, and the flusher is trying to update s.imm,
			// you will get a deadlock.
			time.Sleep(10 * time.Millisecond)
		}
		if err != nil {
			done(err)
			return errors.Wrap(err, "writeRequests")
		}
		if err := db.writeToLSM(b); err != nil {
			done(err)
			return errors.Wrap(err, "writeRequests")
		}
		db.updateOffset(b.Ptrs)
	}
	done(nil)
	db.elog.Printf("%d entries written", count)
	return nil
}

func (db *DB) doWrites(lc *y.Closer) {
	defer lc.Done()
	pendingCh := make(chan struct{}, 1)

	writeRequests := func(reqs []*request) {
		if err := db.writeRequests(reqs); err != nil {
			log.Printf("ERROR in Badger::writeRequests: %v", err)
		}
		<-pendingCh
	}

	// This variable tracks the number of pending writes.
	reqLen := new(expvar.Int)
	y.PendingWrites.Set(db.opt.Dir, reqLen)

	reqs := make([]*request, 0, 10)
	for {
		var r *request
		select {
		case r = <-db.writeCh:
		case <-lc.HasBeenClosed():
			goto closedCase
		}

		for {
			reqs = append(reqs, r)
			reqLen.Set(int64(len(reqs)))

			if len(reqs) >= 3*kvWriteChCapacity {
				pendingCh <- struct{}{} // blocking.
				goto writeCase
			}

			select {
			// Either push to pending, or continue to pick from writeCh.
			case r = <-db.writeCh:
			case pendingCh <- struct{}{}:
				goto writeCase
			case <-lc.HasBeenClosed():
				goto closedCase
			}
		}

	closedCase:
		close(db.writeCh)
		for r := range db.writeCh { // Flush the channel.
			reqs = append(reqs, r)
		}

		pendingCh <- struct{}{} // Push to pending before doing a write.
		writeRequests(reqs)
		return

	writeCase:
		go writeRequests(reqs)
		reqs = make([]*request, 0, 10)
		reqLen.Set(0)
	}
}

func (db *DB) sendToWriteCh(entries []*entry) (*request, error) {
	var count, size int64
	for _, e := range entries {
		size += int64(db.opt.estimateSize(e))
		count++
	}
	if count >= db.opt.maxBatchCount || size >= db.opt.maxBatchSize {
		return nil, ErrTxnTooBig
	}

	// We can only service one request because we need each txn to be stored in a contigous section.
	// Txns should not interleave among other txns or rewrites.
	req := requestPool.Get().(*request)
	req.Entries = entries
	req.Wg = sync.WaitGroup{}
	req.Wg.Add(1)
	db.writeCh <- req
	y.NumPuts.Add(int64(len(entries)))

	return req, nil
}

// batchSet applies a list of badger.Entry. If a request level error occurs it
// will be returned.
//   Check(kv.BatchSet(entries))
func (db *DB) batchSet(entries []*entry) error {
	req, err := db.sendToWriteCh(entries)
	if err != nil {
		return err
	}

	req.Wg.Wait()
	req.Entries = nil
	err = req.Err
	requestPool.Put(req)
	return err
}

// batchSetAsync is the asynchronous version of batchSet. It accepts a callback
// function which is called when all the sets are complete. If a request level
// error occurs, it will be passed back via the callback.
//   err := kv.BatchSetAsync(entries, func(err error)) {
//      Check(err)
//   }
func (db *DB) batchSetAsync(entries []*entry, f func(error)) error {
	req, err := db.sendToWriteCh(entries)
	if err != nil {
		return err
	}
	go func() {
		req.Wg.Wait()
		err := req.Err
		req.Entries = nil
		requestPool.Put(req)
		// Write is complete. Let's call the callback function now.
		f(err)
	}()
	return nil
}

var errNoRoom = errors.New("No room for write")

// ensureRoomForWrite is always called serially.
func (db *DB) ensureRoomForWrite() error {
	var err error
	db.Lock()
	defer db.Unlock()
	if db.mt.MemSize() < db.opt.MaxTableSize {
		return nil
	}

	y.AssertTrue(db.mt != nil) // A nil mt indicates that DB is being closed.
	select {
	case db.flushChan <- flushTask{db.mt, db.vptr}:
		db.elog.Printf("Flushing value log to disk if async mode.")
		// Ensure value log is synced to disk so this memtable's contents wouldn't be lost.
		err = db.vlog.sync()
		if err != nil {
			return err
		}

		db.elog.Printf("Flushing memtable, mt.size=%d size of flushChan: %d\n",
			db.mt.MemSize(), len(db.flushChan))
		// We manage to push this task. Let's modify imm.
		db.imm = append(db.imm, db.mt)
		db.mt = skl.NewSkiplist(arenaSize(&db.opt))
		// New memtable is empty. We certainly have room.
		return nil
	default:
		// We need to do this to unlock and allow the flusher to modify imm.
		return errNoRoom
	}
}

func arenaSize(opt *Options) int64 {
	return opt.MaxTableSize + opt.maxBatchSize + opt.maxBatchCount*int64(skl.MaxNodeSize)
}

// WriteLevel0Table flushes memtable. It drops deleteValues.
func writeLevel0Table(s *skl.Skiplist, f *os.File) error {
	iter := s.NewIterator()
	defer iter.Close()
	b := table.NewTableBuilder()
	defer b.Close()
	for iter.SeekToFirst(); iter.Valid(); iter.Next() {
		if err := b.Add(iter.Key(), iter.Value()); err != nil {
			return err
		}
	}
	_, err := f.Write(b.Finish())
	return err
}

type flushTask struct {
	mt   *skl.Skiplist
	vptr valuePointer
}

func (db *DB) flushMemtable(lc *y.Closer) error {
	defer lc.Done()

	for ft := range db.flushChan {
		if ft.mt == nil {
			return nil
		}

		if !ft.mt.Empty() {
			// Store badger head even if vptr is zero, need it for readTs
			db.elog.Printf("Storing offset: %+v\n", ft.vptr)
			offset := make([]byte, vptrSize)
			ft.vptr.Encode(offset)

			// Pick the max commit ts, so in case of crash, our read ts would be higher than all the
			// commits.
			headTs := y.KeyWithTs(head, db.orc.commitTs())
			ft.mt.Put(headTs, y.ValueStruct{Value: offset})
		}

		fileID := db.lc.reserveFileID()
		fd, err := y.CreateSyncedFile(table.NewFilename(fileID, db.opt.Dir), true)
		if err != nil {
			return y.Wrap(err)
		}

		// Don't block just to sync the directory entry.
		dirSyncCh := make(chan error)
		go func() { dirSyncCh <- syncDir(db.opt.Dir) }()

		err = writeLevel0Table(ft.mt, fd)
		dirSyncErr := <-dirSyncCh

		if err != nil {
			db.elog.Errorf("ERROR while writing to level 0: %v", err)
			return err
		}
		if dirSyncErr != nil {
			db.elog.Errorf("ERROR while syncing level directory: %v", dirSyncErr)
			return err
		}

		tbl, err := table.OpenTable(fd, db.opt.TableLoadingMode)
		if err != nil {
			db.elog.Printf("ERROR while opening table: %v", err)
			return err
		}
		// We own a ref on tbl.
		err = db.lc.addLevel0Table(tbl) // This will incrRef (if we don't error, sure)
		tbl.DecrRef()                   // Releases our ref.
		if err != nil {
			return err
		}

		// Update s.imm. Need a lock.
		db.Lock()
		y.AssertTrue(ft.mt == db.imm[0]) //For now, single threaded.
		db.imm = db.imm[1:]
		ft.mt.DecrRef() // Return memory.
		db.Unlock()
	}
	return nil
}

func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return true, err
}

func (db *DB) updateSize(lc *y.Closer) {
	defer lc.Done()

	metricsTicker := time.NewTicker(5 * time.Minute)
	defer metricsTicker.Stop()

	newInt := func(val int64) *expvar.Int {
		v := new(expvar.Int)
		v.Add(val)
		return v
	}

	totalSize := func(dir string) (int64, int64) {
		var lsmSize, vlogSize int64
		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			ext := filepath.Ext(path)
			if ext == ".sst" {
				lsmSize += info.Size()
			} else if ext == ".vlog" {
				vlogSize += info.Size()
			}
			return nil
		})
		if err != nil {
			db.elog.Printf("Got error while calculating total size of directory: %s", dir)
		}
		return lsmSize, vlogSize
	}

	for {
		select {
		case <-metricsTicker.C:
			lsmSize, vlogSize := totalSize(db.opt.Dir)
			y.LSMSize.Set(db.opt.Dir, newInt(lsmSize))
			// If valueDir is different from dir, we'd have to do another walk.
			if db.opt.ValueDir != db.opt.Dir {
				_, vlogSize = totalSize(db.opt.ValueDir)
			}
			y.VlogSize.Set(db.opt.Dir, newInt(vlogSize))
		case <-lc.HasBeenClosed():
			return
		}
	}
}

// RunValueLogGC would trigger a value log garbage collection with no guarantees that a call would
// result in a space reclaim. Every run would in the best case rewrite only one log file. So,
// repeated calls may be necessary.
//
// The way it currently works is that it would randomly pick up a value log file, and sample it. If
// the sample shows that we can discard at least discardRatio space of that file, it would be
// rewritten. Else, an ErrNoRewrite error would be returned indicating that the GC didn't result in
// any file rewrite.
//
// We recommend setting discardRatio to 0.5, thus indicating that a file be rewritten if half the
// space can be discarded.  This results in a lifetime value log write amplification of 2 (1 from
// original write + 0.5 rewrite + 0.25 + 0.125 + ... = 2). Setting it to higher value would result
// in fewer space reclaims, while setting it to a lower value would result in more space reclaims at
// the cost of increased activity on the LSM tree. discardRatio must be in the range (0.0, 1.0),
// both endpoints excluded, otherwise an ErrInvalidRequest is returned.
//
// Only one GC is allowed at a time. If another value log GC is running, or DB has been closed, this
// would return an ErrRejected.
//
// Note: Every time GC is run, it would produce a spike of activity on the LSM tree.
func (db *DB) RunValueLogGC(discardRatio float64) error {
	if discardRatio >= 1.0 || discardRatio <= 0.0 {
		return ErrInvalidRequest
	}
	return db.vlog.runGC(discardRatio)
}
