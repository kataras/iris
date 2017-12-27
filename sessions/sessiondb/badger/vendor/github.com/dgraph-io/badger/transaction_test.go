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
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dgraph-io/badger/options"
	"github.com/dgraph-io/badger/y"

	"github.com/stretchr/testify/require"
)

func TestTxnSimple(t *testing.T) {
	runBadgerTest(t, nil, func(t *testing.T, db *DB) {

		txn := db.NewTransaction(true)

		for i := 0; i < 10; i++ {
			k := []byte(fmt.Sprintf("key=%d", i))
			v := []byte(fmt.Sprintf("val=%d", i))
			txn.Set(k, v)
		}

		item, err := txn.Get([]byte("key=8"))
		require.NoError(t, err)

		val, err := item.Value()
		require.NoError(t, err)
		require.Equal(t, []byte("val=8"), val)

		require.Error(t, ErrManagedTxn, txn.CommitAt(100, nil))
		require.NoError(t, txn.Commit(nil))
	})
}

func TestTxnCommitAsync(t *testing.T) {
	runBadgerTest(t, nil, func(t *testing.T, db *DB) {

		txn := db.NewTransaction(true)
		key := func(i int) []byte {
			return []byte(fmt.Sprintf("key=%d", i))
		}
		for i := 0; i < 40; i++ {
			err := txn.Set(key(i), []byte(strconv.Itoa(100)))
			require.NoError(t, err)
		}
		require.NoError(t, txn.Commit(nil))
		txn.Discard()

		var done uint64
		go func() {
			// Keep checking balance variant
			for atomic.LoadUint64(&done) == 0 {
				txn := db.NewTransaction(false)
				totalBalance := 0
				for i := 0; i < 40; i++ {
					item, err := txn.Get(key(i))
					require.NoError(t, err)
					val, err := item.Value()
					require.NoError(t, err)
					bal, err := strconv.Atoi(string(val))
					require.NoError(t, err)
					totalBalance += bal
				}
				require.Equal(t, totalBalance, 4000)
				txn.Discard()
			}
		}()

		var wg sync.WaitGroup
		wg.Add(100)
		for i := 0; i < 100; i++ {
			go func() {
				txn := db.NewTransaction(true)
				delta := rand.Intn(100)
				for i := 0; i < 20; i++ {
					err := txn.Set(key(i), []byte(strconv.Itoa(100-delta)))
					require.NoError(t, err)
				}
				for i := 20; i < 40; i++ {
					err := txn.Set(key(i), []byte(strconv.Itoa(100+delta)))
					require.NoError(t, err)
				}
				// We are only doing writes, so there won't be any conflicts.
				require.NoError(t, txn.Commit(func(err error) {}))
				txn.Discard()
				wg.Done()
			}()
		}
		wg.Wait()
		atomic.StoreUint64(&done, 1)
		time.Sleep(time.Millisecond * 10) // allow goroutine to complete.
	})
}

func TestTxnVersions(t *testing.T) {
	runBadgerTest(t, nil, func(t *testing.T, db *DB) {
		k := []byte("key")
		for i := 1; i < 10; i++ {
			txn := db.NewTransaction(true)

			txn.Set(k, []byte(fmt.Sprintf("valversion=%d", i)))
			require.NoError(t, txn.Commit(nil))
			require.Equal(t, uint64(i), db.orc.readTs())
		}

		checkIterator := func(itr *Iterator, i int) {
			count := 0
			for itr.Rewind(); itr.Valid(); itr.Next() {
				item := itr.Item()
				require.Equal(t, k, item.Key())

				val, err := item.Value()
				require.NoError(t, err)
				exp := fmt.Sprintf("valversion=%d", i)
				require.Equal(t, exp, string(val), "i=%d", i)
				count++
			}
			require.Equal(t, 1, count, "i=%d", i) // Should only loop once.
		}

		checkAllVersions := func(itr *Iterator, i int) {
			var version uint64
			if itr.opt.Reverse {
				version = 1
			} else {
				version = uint64(i)
			}

			count := 0
			for itr.Rewind(); itr.Valid(); itr.Next() {
				item := itr.Item()
				require.Equal(t, k, item.Key())
				require.Equal(t, version, item.Version())

				val, err := item.Value()
				require.NoError(t, err)
				exp := fmt.Sprintf("valversion=%d", version)
				require.Equal(t, exp, string(val), "v=%d", version)
				count++

				if itr.opt.Reverse {
					version++
				} else {
					version--
				}
			}
			require.Equal(t, i, count, "i=%d", i) // Should loop as many times as i.
		}

		for i := 1; i < 10; i++ {
			txn := db.NewTransaction(true)
			txn.readTs = uint64(i) // Read version at i.

			item, err := txn.Get(k)
			require.NoError(t, err)

			val, err := item.Value()
			require.NoError(t, err)
			require.Equal(t, []byte(fmt.Sprintf("valversion=%d", i)), val,
				"Expected versions to match up at i=%d", i)

			// Try retrieving the latest version forward and reverse.
			itr := txn.NewIterator(DefaultIteratorOptions)
			checkIterator(itr, i)

			opt := DefaultIteratorOptions
			opt.Reverse = true
			itr = txn.NewIterator(opt)
			checkIterator(itr, i)

			// Now try retrieving all versions forward and reverse.
			opt = DefaultIteratorOptions
			opt.AllVersions = true
			itr = txn.NewIterator(opt)
			checkAllVersions(itr, i)

			opt = DefaultIteratorOptions
			opt.AllVersions = true
			opt.Reverse = true
			itr = txn.NewIterator(opt)
			checkAllVersions(itr, i)

			txn.Discard()
		}
		txn := db.NewTransaction(true)
		defer txn.Discard()
		item, err := txn.Get(k)
		require.NoError(t, err)

		val, err := item.Value()
		require.NoError(t, err)
		require.Equal(t, []byte("valversion=9"), val)
	})
}

func TestTxnWriteSkew(t *testing.T) {
	runBadgerTest(t, nil, func(t *testing.T, db *DB) {
		// Accounts
		ax := []byte("x")
		ay := []byte("y")

		// Set balance to $100 in each account.
		txn := db.NewTransaction(true)
		defer txn.Discard()
		val := []byte(strconv.Itoa(100))
		txn.Set(ax, val)
		txn.Set(ay, val)
		require.NoError(t, txn.Commit(nil))
		require.Equal(t, uint64(1), db.orc.readTs())

		getBal := func(txn *Txn, key []byte) (bal int) {
			item, err := txn.Get(key)
			require.NoError(t, err)

			val, err := item.Value()
			require.NoError(t, err)
			bal, err = strconv.Atoi(string(val))
			require.NoError(t, err)
			return bal
		}

		// Start two transactions, each would read both accounts and deduct from one account.
		txn1 := db.NewTransaction(true)

		sum := getBal(txn1, ax)
		sum += getBal(txn1, ay)
		require.Equal(t, 200, sum)
		txn1.Set(ax, []byte("0")) // Deduct 100 from ax.

		// Let's read this back.
		sum = getBal(txn1, ax)
		require.Equal(t, 0, sum)
		sum += getBal(txn1, ay)
		require.Equal(t, 100, sum)
		// Don't commit yet.

		txn2 := db.NewTransaction(true)

		sum = getBal(txn2, ax)
		sum += getBal(txn2, ay)
		require.Equal(t, 200, sum)
		txn2.Set(ay, []byte("0")) // Deduct 100 from ay.

		// Let's read this back.
		sum = getBal(txn2, ax)
		require.Equal(t, 100, sum)
		sum += getBal(txn2, ay)
		require.Equal(t, 100, sum)

		// Commit both now.
		require.NoError(t, txn1.Commit(nil))
		require.Error(t, txn2.Commit(nil)) // This should fail.

		require.Equal(t, uint64(2), db.orc.readTs())
	})
}

// a3, a2, b4 (del), b3, c2, c1
// Read at ts=4 -> a3, c2
// Read at ts=4(Uncomitted) -> a3, b4
// Read at ts=3 -> a3, b3, c2
// Read at ts=2 -> a2, c2
// Read at ts=1 -> c1
func TestTxnIterationEdgeCase(t *testing.T) {
	runBadgerTest(t, nil, func(t *testing.T, db *DB) {
		ka := []byte("a")
		kb := []byte("b")
		kc := []byte("c")

		// c1
		txn := db.NewTransaction(true)
		txn.Set(kc, []byte("c1"))
		require.NoError(t, txn.Commit(nil))
		require.Equal(t, uint64(1), db.orc.readTs())

		// a2, c2
		txn = db.NewTransaction(true)
		txn.Set(ka, []byte("a2"))
		txn.Set(kc, []byte("c2"))
		require.NoError(t, txn.Commit(nil))
		require.Equal(t, uint64(2), db.orc.readTs())

		// b3
		txn = db.NewTransaction(true)
		txn.Set(ka, []byte("a3"))
		txn.Set(kb, []byte("b3"))
		require.NoError(t, txn.Commit(nil))
		require.Equal(t, uint64(3), db.orc.readTs())

		// b4, c4(del) (Uncomitted)
		txn4 := db.NewTransaction(true)
		require.NoError(t, txn4.Set(kb, []byte("b4")))
		require.NoError(t, txn4.Delete(kc))
		require.Equal(t, uint64(3), db.orc.readTs())

		// b4 (del)
		txn = db.NewTransaction(true)
		txn.Delete(kb)
		require.NoError(t, txn.Commit(nil))
		require.Equal(t, uint64(4), db.orc.readTs())

		checkIterator := func(itr *Iterator, expected []string) {
			var i int
			for itr.Rewind(); itr.Valid(); itr.Next() {
				item := itr.Item()
				val, err := item.Value()
				require.NoError(t, err)
				require.Equal(t, expected[i], string(val), "readts=%d", itr.readTs)
				i++
			}
			require.Equal(t, len(expected), i)
		}

		txn = db.NewTransaction(true)
		defer txn.Discard()
		itr := txn.NewIterator(DefaultIteratorOptions)
		itr5 := txn4.NewIterator(DefaultIteratorOptions)
		checkIterator(itr, []string{"a3", "c2"})
		checkIterator(itr5, []string{"a3", "b4"})

		rev := DefaultIteratorOptions
		rev.Reverse = true
		itr = txn.NewIterator(rev)
		itr5 = txn4.NewIterator(rev)
		checkIterator(itr, []string{"c2", "a3"})
		checkIterator(itr5, []string{"b4", "a3"})

		txn.readTs = 3
		itr = txn.NewIterator(DefaultIteratorOptions)
		checkIterator(itr, []string{"a3", "b3", "c2"})
		itr = txn.NewIterator(rev)
		checkIterator(itr, []string{"c2", "b3", "a3"})

		txn.readTs = 2
		itr = txn.NewIterator(DefaultIteratorOptions)
		checkIterator(itr, []string{"a2", "c2"})
		itr = txn.NewIterator(rev)
		checkIterator(itr, []string{"c2", "a2"})

		txn.readTs = 1
		itr = txn.NewIterator(DefaultIteratorOptions)
		checkIterator(itr, []string{"c1"})
		itr = txn.NewIterator(rev)
		checkIterator(itr, []string{"c1"})
	})
}

// a2, a3, b4 (del), b3, c2, c1
// Read at ts=4 -> a3, c2
// Read at ts=3 -> a3, b3, c2
// Read at ts=2 -> a2, c2
// Read at ts=1 -> c1
func TestTxnIterationEdgeCase2(t *testing.T) {
	runBadgerTest(t, nil, func(t *testing.T, db *DB) {
		ka := []byte("a")
		kb := []byte("aa")
		kc := []byte("aaa")

		// c1
		txn := db.NewTransaction(true)
		txn.Set(kc, []byte("c1"))
		require.NoError(t, txn.Commit(nil))
		require.Equal(t, uint64(1), db.orc.readTs())

		// a2, c2
		txn = db.NewTransaction(true)
		txn.Set(ka, []byte("a2"))
		txn.Set(kc, []byte("c2"))
		require.NoError(t, txn.Commit(nil))
		require.Equal(t, uint64(2), db.orc.readTs())

		// b3
		txn = db.NewTransaction(true)
		txn.Set(ka, []byte("a3"))
		txn.Set(kb, []byte("b3"))
		require.NoError(t, txn.Commit(nil))
		require.Equal(t, uint64(3), db.orc.readTs())

		// b4 (del)
		txn = db.NewTransaction(true)
		txn.Delete(kb)
		require.NoError(t, txn.Commit(nil))
		require.Equal(t, uint64(4), db.orc.readTs())

		checkIterator := func(itr *Iterator, expected []string) {
			var i int
			for itr.Rewind(); itr.Valid(); itr.Next() {
				item := itr.Item()
				val, err := item.Value()
				require.NoError(t, err)
				require.Equal(t, expected[i], string(val), "readts=%d", itr.readTs)
				i++
			}
			require.Equal(t, len(expected), i)
		}
		txn = db.NewTransaction(true)
		defer txn.Discard()
		rev := DefaultIteratorOptions
		rev.Reverse = true

		itr := txn.NewIterator(DefaultIteratorOptions)
		checkIterator(itr, []string{"a3", "c2"})
		itr = txn.NewIterator(rev)
		checkIterator(itr, []string{"c2", "a3"})

		txn.readTs = 5
		itr = txn.NewIterator(DefaultIteratorOptions)
		itr.Seek(ka)
		require.True(t, itr.Valid())
		require.Equal(t, itr.item.Key(), ka)
		itr.Seek(kc)
		require.True(t, itr.Valid())
		require.Equal(t, itr.item.Key(), kc)

		itr = txn.NewIterator(rev)
		itr.Seek(ka)
		require.True(t, itr.Valid())
		require.Equal(t, itr.item.Key(), ka)
		itr.Seek(kc)
		require.True(t, itr.Valid())
		require.Equal(t, itr.item.Key(), kc)

		txn.readTs = 3
		itr = txn.NewIterator(DefaultIteratorOptions)
		checkIterator(itr, []string{"a3", "b3", "c2"})
		itr = txn.NewIterator(rev)
		checkIterator(itr, []string{"c2", "b3", "a3"})

		txn.readTs = 2
		itr = txn.NewIterator(DefaultIteratorOptions)
		checkIterator(itr, []string{"a2", "c2"})
		itr = txn.NewIterator(rev)
		checkIterator(itr, []string{"c2", "a2"})

		txn.readTs = 1
		itr = txn.NewIterator(DefaultIteratorOptions)
		checkIterator(itr, []string{"c1"})
		itr = txn.NewIterator(rev)
		checkIterator(itr, []string{"c1"})
	})
}

func TestTxnIterationEdgeCase3(t *testing.T) {
	runBadgerTest(t, nil, func(t *testing.T, db *DB) {
		kb := []byte("abc")
		kc := []byte("acd")
		kd := []byte("ade")

		// c1
		txn := db.NewTransaction(true)
		txn.Set(kc, []byte("c1"))
		require.NoError(t, txn.Commit(nil))
		require.Equal(t, uint64(1), db.orc.readTs())

		// b2
		txn = db.NewTransaction(true)
		txn.Set(kb, []byte("b2"))
		require.NoError(t, txn.Commit(nil))
		require.Equal(t, uint64(2), db.orc.readTs())

		txn2 := db.NewTransaction(true)
		require.NoError(t, txn2.Set(kd, []byte("d2")))
		require.NoError(t, txn2.Delete(kc))

		txn = db.NewTransaction(true)
		defer txn.Discard()
		rev := DefaultIteratorOptions
		rev.Reverse = true

		itr := txn.NewIterator(DefaultIteratorOptions)
		itr.Seek([]byte("ab"))
		require.True(t, itr.Valid())
		require.Equal(t, itr.item.Key(), kb)
		itr.Seek([]byte("ac"))
		require.True(t, itr.Valid())
		require.Equal(t, itr.item.Key(), kc)
		itr.Seek(nil)
		require.True(t, itr.Valid())
		require.Equal(t, itr.item.Key(), kb)
		itr.Seek([]byte("ac"))
		itr.Rewind()
		itr.Seek(nil)
		require.True(t, itr.Valid())
		require.Equal(t, itr.item.Key(), kb)
		itr.Seek([]byte("ac"))
		require.True(t, itr.Valid())
		require.Equal(t, itr.item.Key(), kc)

		//  Keys: "abc", "ade"
		// Read pending writes.
		itr = txn2.NewIterator(DefaultIteratorOptions)
		itr.Seek([]byte("ab"))
		require.True(t, itr.Valid())
		require.Equal(t, itr.item.Key(), kb)
		itr.Seek([]byte("ac"))
		require.True(t, itr.Valid())
		require.Equal(t, itr.item.Key(), kd)
		itr.Seek(nil)
		require.True(t, itr.Valid())
		require.Equal(t, itr.item.Key(), kb)
		itr.Seek([]byte("ac"))
		itr.Rewind()
		itr.Seek(nil)
		require.True(t, itr.Valid())
		require.Equal(t, itr.item.Key(), kb)
		itr.Seek([]byte("ad"))
		require.True(t, itr.Valid())
		require.Equal(t, itr.item.Key(), kd)

		itr = txn.NewIterator(rev)
		itr.Seek([]byte("ac"))
		require.True(t, itr.Valid())
		require.Equal(t, itr.item.Key(), kb)
		itr.Seek([]byte("ad"))
		require.True(t, itr.Valid())
		require.Equal(t, itr.item.Key(), kc)
		itr.Seek(nil)
		require.True(t, itr.Valid())
		require.Equal(t, itr.item.Key(), kc)
		itr.Seek([]byte("ac"))
		itr.Rewind()
		require.True(t, itr.Valid())
		require.Equal(t, itr.item.Key(), kc)
		itr.Seek([]byte("ad"))
		require.True(t, itr.Valid())
		require.Equal(t, itr.item.Key(), kc)

		//  Keys: "abc", "ade"
		itr = txn2.NewIterator(rev)
		itr.Seek([]byte("ad"))
		require.True(t, itr.Valid())
		require.Equal(t, itr.item.Key(), kb)
		itr.Seek([]byte("ae"))
		require.True(t, itr.Valid())
		require.Equal(t, itr.item.Key(), kd)
		itr.Seek(nil)
		require.True(t, itr.Valid())
		require.Equal(t, itr.item.Key(), kd)
		itr.Seek([]byte("ab"))
		itr.Rewind()
		require.True(t, itr.Valid())
		require.Equal(t, itr.item.Key(), kd)
		itr.Seek([]byte("ac"))
		require.True(t, itr.Valid())
		require.Equal(t, itr.item.Key(), kb)
	})
}

func TestIteratorAllVersionsWithDeleted(t *testing.T) {
	runBadgerTest(t, nil, func(t *testing.T, db *DB) {
		// Write two keys
		err := db.Update(func(txn *Txn) error {
			txn.Set([]byte("answer1"), []byte("42"))
			txn.Set([]byte("answer2"), []byte("43"))
			return nil
		})
		require.NoError(t, err)

		// Delete the specific key version from underlying db directly
		err = db.View(func(txn *Txn) error {
			item, err := txn.Get([]byte("answer1"))
			require.NoError(t, err)
			err = txn.db.batchSet([]*Entry{
				{
					Key:  y.KeyWithTs(item.key, item.version),
					meta: bitDelete,
				},
			})
			require.NoError(t, err)
			return err
		})
		require.NoError(t, err)

		opts := DefaultIteratorOptions
		opts.AllVersions = true
		opts.PrefetchValues = false

		// Verify that deleted shows up when AllVersions is set.
		err = db.View(func(txn *Txn) error {
			it := txn.NewIterator(opts)
			var count int
			for it.Rewind(); it.Valid(); it.Next() {
				count++
				item := it.Item()
				if count == 1 {
					require.Equal(t, []byte("answer1"), item.Key())
					require.True(t, item.meta&bitDelete > 0)
				} else {
					require.Equal(t, []byte("answer2"), item.Key())
				}
			}
			require.Equal(t, 2, count)
			return nil
		})
		require.NoError(t, err)
	})
}

func TestIteratorAllVersionsWithDeleted2(t *testing.T) {
	runBadgerTest(t, nil, func(t *testing.T, db *DB) {
		// Set and delete alternatively
		for i := 0; i < 4; i++ {
			err := db.Update(func(txn *Txn) error {
				if i%2 == 0 {
					txn.Set([]byte("key"), []byte("value"))
					return nil
				}
				txn.Delete([]byte("key"))
				return nil
			})
			require.NoError(t, err)
		}

		opts := DefaultIteratorOptions
		opts.AllVersions = true
		opts.PrefetchValues = false

		// Verify that deleted shows up when AllVersions is set.
		err := db.View(func(txn *Txn) error {
			it := txn.NewIterator(opts)
			var count int
			for it.Rewind(); it.Valid(); it.Next() {
				item := it.Item()
				require.Equal(t, []byte("key"), item.Key())
				if count%2 != 0 {
					val, err := item.Value()
					require.NoError(t, err)
					require.Equal(t, val, []byte("value"))
				} else {
					require.True(t, item.meta&bitDelete > 0)
				}
				count++
			}
			require.Equal(t, 4, count)
			return nil
		})
		require.NoError(t, err)
	})
}

func TestManagedDB(t *testing.T) {
	dir, err := ioutil.TempDir("", "badger")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	opt := getTestOptions(dir)
	kv, err := OpenManaged(opt)
	require.NoError(t, err)
	defer kv.Close()

	key := func(i int) []byte {
		return []byte(fmt.Sprintf("key-%02d", i))
	}

	val := func(i int) []byte {
		return []byte(fmt.Sprintf("val-%d", i))
	}

	// Don't allow these APIs in ManagedDB
	require.Panics(t, func() { kv.NewTransaction(false) })

	err = kv.Update(func(tx *Txn) error { return nil })
	require.Equal(t, ErrManagedTxn, err)

	err = kv.View(func(tx *Txn) error { return nil })
	require.Equal(t, ErrManagedTxn, err)

	// Write data at t=3.
	txn := kv.NewTransactionAt(3, true)
	for i := 0; i <= 3; i++ {
		require.NoError(t, txn.Set(key(i), val(i)))
	}
	require.Error(t, ErrManagedTxn, txn.Commit(nil))
	require.NoError(t, txn.CommitAt(3, nil))

	// Read data at t=2.
	txn = kv.NewTransactionAt(2, false)
	for i := 0; i <= 3; i++ {
		_, err := txn.Get(key(i))
		require.Equal(t, ErrKeyNotFound, err)
	}
	txn.Discard()

	// Read data at t=3.
	txn = kv.NewTransactionAt(3, false)
	for i := 0; i <= 3; i++ {
		item, err := txn.Get(key(i))
		require.NoError(t, err)
		require.Equal(t, uint64(3), item.Version())
		v, err := item.Value()
		require.NoError(t, err)
		require.Equal(t, val(i), v)
	}
	txn.Discard()

	// Write data at t=7.
	txn = kv.NewTransactionAt(6, true)
	for i := 0; i <= 7; i++ {
		_, err := txn.Get(key(i))
		if err == nil {
			continue // Don't overwrite existing keys.
		}
		require.NoError(t, txn.Set(key(i), val(i)))
	}
	require.NoError(t, txn.CommitAt(7, nil))

	// Read data at t=9.
	txn = kv.NewTransactionAt(9, false)
	for i := 0; i <= 9; i++ {
		item, err := txn.Get(key(i))
		if i <= 7 {
			require.NoError(t, err)
		} else {
			require.Equal(t, ErrKeyNotFound, err)
		}

		if i <= 3 {
			require.Equal(t, uint64(3), item.Version())
		} else if i <= 7 {
			require.Equal(t, uint64(7), item.Version())
		}
		if i <= 7 {
			v, err := item.Value()
			require.NoError(t, err)
			require.Equal(t, val(i), v)
		}
	}
	txn.Discard()
}

func TestArmV7Issue311Fix(t *testing.T) {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	config := DefaultOptions
	config.TableLoadingMode = options.MemoryMap
	config.ValueLogFileSize = 16 << 20
	config.LevelOneSize = 8 << 20
	config.MaxTableSize = 2 << 20
	config.Dir = dir
	config.ValueDir = dir
	config.SyncWrites = false

	db, err := Open(config)
	if err != nil {
		t.Fatalf("cannot open db at location %s: %v", dir, err)
	}

	err = db.View(func(txn *Txn) error { return nil })
	if err != nil {
		t.Fatal(err)
	}

	err = db.Update(func(txn *Txn) error {
		return txn.Set([]byte{0x11}, []byte{0x22})
	})
	if err != nil {
		t.Fatal(err)
	}

	err = db.Update(func(txn *Txn) error {
		return txn.Set([]byte{0x11}, []byte{0x22})
	})

	if err != nil {
		t.Fatal(err)
	}

	if err = db.Close(); err != nil {
		t.Fatal(err)
	}
}
