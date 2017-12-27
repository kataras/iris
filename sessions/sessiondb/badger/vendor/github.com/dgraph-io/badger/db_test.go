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
	"encoding/binary"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/dgraph-io/badger/options"

	"github.com/dgraph-io/badger/y"
	"github.com/stretchr/testify/require"
)

var mmap = flag.Bool("vlog_mmap", true, "Specify if value log must be memory-mapped")

func getTestOptions(dir string) Options {
	opt := DefaultOptions
	opt.MaxTableSize = 1 << 15 // Force more compaction.
	opt.LevelOneSize = 4 << 15 // Force more compaction.
	opt.Dir = dir
	opt.ValueDir = dir
	opt.SyncWrites = false
	if !*mmap {
		opt.ValueLogLoadingMode = options.FileIO
	}
	return opt
}

func getItemValue(t *testing.T, item *Item) (val []byte) {
	v, err := item.Value()
	if err != nil {
		t.Error(err)
	}
	if v == nil {
		return nil
	}
	another, err := item.ValueCopy(nil)
	require.NoError(t, err)
	require.Equal(t, v, another)
	return v
}

func txnSet(t *testing.T, kv *DB, key []byte, val []byte, meta byte) {
	txn := kv.NewTransaction(true)
	require.NoError(t, txn.SetWithMeta(key, val, meta))
	require.NoError(t, txn.Commit(nil))
}

func txnDelete(t *testing.T, kv *DB, key []byte) {
	txn := kv.NewTransaction(true)
	require.NoError(t, txn.Delete(key))
	require.NoError(t, txn.Commit(nil))
}

// Opens a badger db and runs a a test on it.
func runBadgerTest(t *testing.T, opts *Options, test func(t *testing.T, db *DB)) {
	dir, err := ioutil.TempDir("", "badger")
	require.NoError(t, err)
	defer os.RemoveAll(dir)
	if opts == nil {
		opts = new(Options)
		*opts = getTestOptions(dir)
	}
	db, err := Open(*opts)
	require.NoError(t, err)
	defer db.Close()
	test(t, db)
}

func TestWrite(t *testing.T) {
	runBadgerTest(t, nil, func(t *testing.T, db *DB) {
		for i := 0; i < 100; i++ {
			txnSet(t, db, []byte(fmt.Sprintf("key%d", i)), []byte(fmt.Sprintf("val%d", i)), 0x00)
		}
	})
}

func TestUpdateAndView(t *testing.T) {
	runBadgerTest(t, nil, func(t *testing.T, db *DB) {
		err := db.Update(func(txn *Txn) error {
			for i := 0; i < 10; i++ {
				err := txn.Set([]byte(fmt.Sprintf("key%d", i)), []byte(fmt.Sprintf("val%d", i)))
				if err != nil {
					return err
				}
			}
			return nil
		})
		require.NoError(t, err)

		err = db.View(func(txn *Txn) error {
			for i := 0; i < 10; i++ {
				item, err := txn.Get([]byte(fmt.Sprintf("key%d", i)))
				if err != nil {
					return err
				}

				val, err := item.Value()
				if err != nil {
					return err
				}
				expected := []byte(fmt.Sprintf("val%d", i))
				require.Equal(t, expected, val,
					"Invalid value for key %q. expected: %q, actual: %q",
					item.Key(), expected, val)
			}
			return nil
		})
		require.NoError(t, err)
	})
}

func TestConcurrentWrite(t *testing.T) {
	runBadgerTest(t, nil, func(t *testing.T, db *DB) {
		// Not a benchmark. Just a simple test for concurrent writes.
		n := 20
		m := 500
		var wg sync.WaitGroup
		for i := 0; i < n; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				for j := 0; j < m; j++ {
					txnSet(t, db, []byte(fmt.Sprintf("k%05d_%08d", i, j)),
						[]byte(fmt.Sprintf("v%05d_%08d", i, j)), byte(j%127))
				}
			}(i)
		}
		wg.Wait()

		t.Log("Starting iteration")

		opt := IteratorOptions{}
		opt.Reverse = false
		opt.PrefetchSize = 10
		opt.PrefetchValues = true

		txn := db.NewTransaction(true)
		it := txn.NewIterator(opt)
		defer it.Close()
		var i, j int
		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			k := item.Key()
			if k == nil {
				break // end of iteration.
			}

			require.EqualValues(t, fmt.Sprintf("k%05d_%08d", i, j), string(k))
			v := getItemValue(t, item)
			require.EqualValues(t, fmt.Sprintf("v%05d_%08d", i, j), string(v))
			require.Equal(t, item.UserMeta(), byte(j%127))
			j++
			if j == m {
				i++
				j = 0
			}
		}
		require.EqualValues(t, n, i)
		require.EqualValues(t, 0, j)
	})
}

func TestGet(t *testing.T) {
	runBadgerTest(t, nil, func(t *testing.T, db *DB) {
		txnSet(t, db, []byte("key1"), []byte("val1"), 0x08)

		txn := db.NewTransaction(false)
		item, err := txn.Get([]byte("key1"))
		require.NoError(t, err)
		require.EqualValues(t, "val1", getItemValue(t, item))
		require.Equal(t, byte(0x08), item.UserMeta())
		txn.Discard()

		txnSet(t, db, []byte("key1"), []byte("val2"), 0x09)

		txn = db.NewTransaction(false)
		item, err = txn.Get([]byte("key1"))
		require.NoError(t, err)
		require.EqualValues(t, "val2", getItemValue(t, item))
		require.Equal(t, byte(0x09), item.UserMeta())
		txn.Discard()

		txnDelete(t, db, []byte("key1"))

		txn = db.NewTransaction(false)
		_, err = txn.Get([]byte("key1"))
		require.Equal(t, ErrKeyNotFound, err)
		txn.Discard()

		txnSet(t, db, []byte("key1"), []byte("val3"), 0x01)

		txn = db.NewTransaction(false)
		item, err = txn.Get([]byte("key1"))
		require.NoError(t, err)
		require.EqualValues(t, "val3", getItemValue(t, item))
		require.Equal(t, byte(0x01), item.UserMeta())

		longVal := make([]byte, 1000)
		txnSet(t, db, []byte("key1"), longVal, 0x00)

		txn = db.NewTransaction(false)
		item, err = txn.Get([]byte("key1"))
		require.NoError(t, err)
		require.EqualValues(t, longVal, getItemValue(t, item))
		txn.Discard()
	})
}

func TestGetAfterDelete(t *testing.T) {
	runBadgerTest(t, nil, func(t *testing.T, db *DB) {
		// populate with one entry
		key := []byte("key")
		txnSet(t, db, key, []byte("val1"), 0x00)
		require.NoError(t, db.Update(func(txn *Txn) error {
			err := txn.Delete(key)
			require.NoError(t, err)

			_, err = txn.Get(key)
			require.Equal(t, ErrKeyNotFound, err)
			return nil
		}))
	})
}

func TestTxnTooBig(t *testing.T) {
	runBadgerTest(t, nil, func(t *testing.T, db *DB) {
		data := func(i int) []byte {
			return []byte(fmt.Sprintf("%b", i))
		}
		//	n := 500000
		n := 1000
		txn := db.NewTransaction(true)
		for i := 0; i < n; {
			if err := txn.Set(data(i), data(i)); err != nil {
				require.NoError(t, txn.Commit(nil))
				txn = db.NewTransaction(true)
			} else {
				i++
			}
		}
		require.NoError(t, txn.Commit(nil))

		txn = db.NewTransaction(true)
		for i := 0; i < n; {
			if err := txn.Delete(data(i)); err != nil {
				require.NoError(t, txn.Commit(nil))
				txn = db.NewTransaction(true)
			} else {
				i++
			}
		}
		require.NoError(t, txn.Commit(nil))
	})
}

func TestGetAfterPurge(t *testing.T) {
	dir, err := ioutil.TempDir("", "badger")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	opts := getTestOptions(dir)
	opts.ValueLogFileSize = 15 << 20
	db, err := OpenManaged(opts)
	require.NoError(t, err)
	defer db.Close()

	data := func(i int) []byte {
		return []byte(fmt.Sprintf("%b", i))
	}
	n := 80
	m := 45 // Increasing would cause ErrTxnTooBig
	sz := 32 << 10
	v := make([]byte, sz)
	for i := 0; i < n; i += 2 {
		version := uint64(i)
		txn := db.NewTransactionAt(version, true)
		for j := 0; j < m; j++ {
			require.NoError(t, txn.Set(data(j), v))
		}
		require.NoError(t, txn.CommitAt(version+1, nil))
	}

	for j := 10; j < m; j++ {
		err := db.PurgeVersionsBelow(data(j), uint64(80))
		require.NoError(t, err)
	}
	for i := 0; i < 10; i++ {
		txn := db.NewTransactionAt(80, false)
		item, err := txn.Get(data(i))
		require.NoError(t, err)
		require.Equal(t, item.Version(), uint64(79))
		txn.Discard()
	}
	err = db.RunValueLogGC(0.2)
	require.NoError(t, err)

	for i := 10; i < m; i++ {
		txn := db.NewTransactionAt(80, false)
		_, err := txn.Get(data(i))
		require.Equal(t, err, ErrKeyNotFound)
		txn.Discard()
	}

	for i := 0; i < 10; i++ {
		txn := db.NewTransactionAt(80, false)
		item, err := txn.Get(data(i))
		require.NoError(t, err)
		require.Equal(t, item.Version(), uint64(79))
		txn.Discard()
	}

}

// Put a lot of data to move some data to disk.
// WARNING: This test might take a while but it should pass!
func TestGetMore(t *testing.T) {
	runBadgerTest(t, nil, func(t *testing.T, db *DB) {

		data := func(i int) []byte {
			return []byte(fmt.Sprintf("%b", i))
		}
		//	n := 500000
		n := 10000
		m := 45 // Increasing would cause ErrTxnTooBig
		for i := 0; i < n; i += m {
			txn := db.NewTransaction(true)
			for j := i; j < i+m && j < n; j++ {
				require.NoError(t, txn.Set(data(j), data(j)))
			}
			require.NoError(t, txn.Commit(nil))
		}
		require.NoError(t, db.validate())

		for i := 0; i < n; i++ {
			txn := db.NewTransaction(false)
			item, err := txn.Get(data(i))
			if err != nil {
				t.Error(err)
			}
			require.EqualValues(t, string(data(i)), string(getItemValue(t, item)))
			txn.Discard()
		}

		// Overwrite
		for i := 0; i < n; i += m {
			txn := db.NewTransaction(true)
			for j := i; j < i+m && j < n; j++ {
				require.NoError(t, txn.Set(data(j),
					// Use a long value that will certainly exceed value threshold.
					[]byte(fmt.Sprintf("zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz%9d", j))))
			}
			require.NoError(t, txn.Commit(nil))
		}
		require.NoError(t, db.validate())

		for i := 0; i < n; i++ {
			expectedValue := fmt.Sprintf("zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz%9d", i)
			k := data(i)
			txn := db.NewTransaction(false)
			item, err := txn.Get(k)
			if err != nil {
				t.Error(err)
			}
			got := string(getItemValue(t, item))
			if expectedValue != got {

				vs, err := db.get(y.KeyWithTs(k, math.MaxUint64))
				require.NoError(t, err)
				fmt.Printf("wanted=%q Item: %s\n", k, item.ToString())
				fmt.Printf("on re-run, got version: %+v\n", vs)

				txn := db.NewTransaction(false)
				itr := txn.NewIterator(DefaultIteratorOptions)
				for itr.Seek(k); itr.Valid(); itr.Next() {
					item := itr.Item()
					fmt.Printf("item=%s\n", item.ToString())
					if !bytes.Equal(item.Key(), k) {
						break
					}
				}
				itr.Close()
				txn.Discard()
			}
			require.EqualValues(t, expectedValue, string(getItemValue(t, item)), "wanted=%q Item: %s\n", k, item.ToString())
			txn.Discard()
		}

		// "Delete" key.
		for i := 0; i < n; i += m {
			if (i % 10000) == 0 {
				fmt.Printf("Deleting i=%d\n", i)
			}
			txn := db.NewTransaction(true)
			for j := i; j < i+m && j < n; j++ {
				require.NoError(t, txn.Delete(data(j)))
			}
			require.NoError(t, txn.Commit(nil))
		}
		db.validate()
		for i := 0; i < n; i++ {
			if (i % 10000) == 0 {
				// Display some progress. Right now, it's not very fast with no caching.
				fmt.Printf("Testing i=%d\n", i)
			}
			k := data(i)
			txn := db.NewTransaction(false)
			_, err := txn.Get([]byte(k))
			require.Equal(t, ErrKeyNotFound, err, "should not have found k: %q", k)
			txn.Discard()
		}
	})
}

// Put a lot of data to move some data to disk.
// WARNING: This test might take a while but it should pass!
func TestExistsMore(t *testing.T) {
	runBadgerTest(t, nil, func(t *testing.T, db *DB) {
		//	n := 500000
		n := 10000
		m := 45
		for i := 0; i < n; i += m {
			if (i % 1000) == 0 {
				t.Logf("Putting i=%d\n", i)
			}
			txn := db.NewTransaction(true)
			for j := i; j < i+m && j < n; j++ {
				require.NoError(t, txn.Set([]byte(fmt.Sprintf("%09d", j)),
					[]byte(fmt.Sprintf("%09d", j))))
			}
			require.NoError(t, txn.Commit(nil))
		}
		db.validate()

		for i := 0; i < n; i++ {
			if (i % 1000) == 0 {
				fmt.Printf("Testing i=%d\n", i)
			}
			k := fmt.Sprintf("%09d", i)
			require.NoError(t, db.View(func(txn *Txn) error {
				_, err := txn.Get([]byte(k))
				require.NoError(t, err)
				return nil
			}))
		}
		require.NoError(t, db.View(func(txn *Txn) error {
			_, err := txn.Get([]byte("non-exists"))
			require.Error(t, err)
			return nil
		}))

		// "Delete" key.
		for i := 0; i < n; i += m {
			if (i % 1000) == 0 {
				fmt.Printf("Deleting i=%d\n", i)
			}
			txn := db.NewTransaction(true)
			for j := i; j < i+m && j < n; j++ {
				require.NoError(t, txn.Delete([]byte(fmt.Sprintf("%09d", j))))
			}
			require.NoError(t, txn.Commit(nil))
		}
		db.validate()
		for i := 0; i < n; i++ {
			if (i % 10000) == 0 {
				// Display some progress. Right now, it's not very fast with no caching.
				fmt.Printf("Testing i=%d\n", i)
			}
			k := fmt.Sprintf("%09d", i)

			require.NoError(t, db.View(func(txn *Txn) error {
				_, err := txn.Get([]byte(k))
				require.Error(t, err)
				return nil
			}))
		}
		fmt.Println("Done and closing")
	})
}

func TestIterate2Basic(t *testing.T) {
	runBadgerTest(t, nil, func(t *testing.T, db *DB) {

		bkey := func(i int) []byte {
			return []byte(fmt.Sprintf("%09d", i))
		}
		bval := func(i int) []byte {
			return []byte(fmt.Sprintf("%025d", i))
		}

		// n := 500000
		n := 10000
		for i := 0; i < n; i++ {
			if (i % 1000) == 0 {
				t.Logf("Put i=%d\n", i)
			}
			txnSet(t, db, bkey(i), bval(i), byte(i%127))
		}

		opt := IteratorOptions{}
		opt.PrefetchValues = true
		opt.PrefetchSize = 10

		txn := db.NewTransaction(false)
		it := txn.NewIterator(opt)
		{
			var count int
			rewind := true
			t.Log("Starting first basic iteration")
			for it.Rewind(); it.Valid(); it.Next() {
				item := it.Item()
				key := item.Key()
				if rewind && count == 5000 {
					// Rewind would skip /head/ key, and it.Next() would skip 0.
					count = 1
					it.Rewind()
					t.Log("Rewinding from 5000 to zero.")
					rewind = false
					continue
				}
				require.EqualValues(t, bkey(count), string(key))
				val := getItemValue(t, item)
				require.EqualValues(t, bval(count), string(val))
				require.Equal(t, byte(count%127), item.UserMeta())
				count++
			}
			require.EqualValues(t, n, count)
		}

		{
			t.Log("Starting second basic iteration")
			idx := 5030
			for it.Seek(bkey(idx)); it.Valid(); it.Next() {
				item := it.Item()
				require.EqualValues(t, bkey(idx), string(item.Key()))
				require.EqualValues(t, bval(idx), string(getItemValue(t, item)))
				idx++
			}
		}
		it.Close()
	})
}

func TestLoad(t *testing.T) {
	dir, err := ioutil.TempDir("", "badger")
	fmt.Printf("Writing to dir %s\n", dir)
	require.NoError(t, err)
	defer os.RemoveAll(dir)
	n := 10000
	{
		kv, _ := Open(getTestOptions(dir))
		for i := 0; i < n; i++ {
			if (i % 10000) == 0 {
				fmt.Printf("Putting i=%d\n", i)
			}
			k := []byte(fmt.Sprintf("%09d", i))
			txnSet(t, kv, k, k, 0x00)
		}
		kv.Close()
	}

	kv, err := Open(getTestOptions(dir))
	require.NoError(t, err)
	require.Equal(t, uint64(10001), kv.orc.readTs())
	for i := 0; i < n; i++ {
		if (i % 10000) == 0 {
			fmt.Printf("Testing i=%d\n", i)
		}
		k := fmt.Sprintf("%09d", i)
		require.NoError(t, kv.View(func(txn *Txn) error {
			item, err := txn.Get([]byte(k))
			require.NoError(t, err)
			require.EqualValues(t, k, string(getItemValue(t, item)))
			return nil
		}))

	}
	kv.Close()
	summary := kv.lc.getSummary()

	// Check that files are garbage collected.
	idMap := getIDMap(dir)
	for fileID := range idMap {
		// Check that name is in summary.filenames.
		require.True(t, summary.fileIDs[fileID], "%d", fileID)
	}
	require.EqualValues(t, len(idMap), len(summary.fileIDs))

	var fileIDs []uint64
	for k := range summary.fileIDs { // Map to array.
		fileIDs = append(fileIDs, k)
	}
	sort.Slice(fileIDs, func(i, j int) bool { return fileIDs[i] < fileIDs[j] })
	fmt.Printf("FileIDs: %v\n", fileIDs)
}

func TestIterateDeleted(t *testing.T) {
	runBadgerTest(t, nil, func(t *testing.T, db *DB) {
		txnSet(t, db, []byte("Key1"), []byte("Value1"), 0x00)
		txnSet(t, db, []byte("Key2"), []byte("Value2"), 0x00)

		iterOpt := DefaultIteratorOptions
		iterOpt.PrefetchValues = false
		txn := db.NewTransaction(false)
		idxIt := txn.NewIterator(iterOpt)
		defer idxIt.Close()

		count := 0
		txn2 := db.NewTransaction(true)
		prefix := []byte("Key")
		for idxIt.Seek(prefix); idxIt.ValidForPrefix(prefix); idxIt.Next() {
			key := idxIt.Item().Key()
			count++
			newKey := make([]byte, len(key))
			copy(newKey, key)
			require.NoError(t, txn2.Delete(newKey))
		}
		require.Equal(t, 2, count)
		require.NoError(t, txn2.Commit(nil))

		for _, prefetch := range [...]bool{true, false} {
			t.Run(fmt.Sprintf("Prefetch=%t", prefetch), func(t *testing.T) {
				txn := db.NewTransaction(false)
				iterOpt = DefaultIteratorOptions
				iterOpt.PrefetchValues = prefetch
				idxIt = txn.NewIterator(iterOpt)

				var estSize int64
				var idxKeys []string
				for idxIt.Seek(prefix); idxIt.Valid(); idxIt.Next() {
					item := idxIt.Item()
					key := item.Key()
					estSize += item.EstimatedSize()
					if !bytes.HasPrefix(key, prefix) {
						break
					}
					idxKeys = append(idxKeys, string(key))
					t.Logf("%+v\n", idxIt.Item())
				}
				require.Equal(t, 0, len(idxKeys))
				require.Equal(t, int64(0), estSize)
			})
		}
	})
}

func TestDeleteWithoutSyncWrite(t *testing.T) {
	dir, err := ioutil.TempDir("", "badger")
	require.NoError(t, err)
	defer os.RemoveAll(dir)
	opt := DefaultOptions
	opt.Dir = dir
	opt.ValueDir = dir
	kv, err := Open(opt)
	if err != nil {
		t.Error(err)
		t.Fail()
	}

	key := []byte("k1")
	// Set a value with size > value threshold so that its written to value log.
	txnSet(t, kv, key, []byte("ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789FOOBARZOGZOG"), 0x00)
	txnDelete(t, kv, key)
	kv.Close()

	// Reopen KV
	kv, err = Open(opt)
	if err != nil {
		t.Error(err)
		t.Fail()
	}
	defer kv.Close()

	require.NoError(t, kv.View(func(txn *Txn) error {
		_, err := txn.Get(key)
		require.Error(t, ErrKeyNotFound, err)
		return nil
	}))
}

func TestPidFile(t *testing.T) {
	runBadgerTest(t, nil, func(t *testing.T, db *DB) {
		// Reopen database
		_, err := Open(getTestOptions(db.opt.Dir))
		require.Error(t, err)
		require.Contains(t, err.Error(), "Another process is using this Badger database")
	})
}

func TestBigKeyValuePairs(t *testing.T) {
	runBadgerTest(t, nil, func(t *testing.T, db *DB) {
		bigK := make([]byte, maxKeySize+1)
		bigV := make([]byte, db.opt.ValueLogFileSize+1)
		small := make([]byte, 10)

		txn := db.NewTransaction(true)
		require.Regexp(t, regexp.MustCompile("Key.*exceeded"), txn.Set(bigK, small))
		require.Regexp(t, regexp.MustCompile("Value.*exceeded"), txn.Set(small, bigV))

		require.NoError(t, txn.Set(small, small))
		require.Regexp(t, regexp.MustCompile("Key.*exceeded"), txn.Set(bigK, bigV))

		require.NoError(t, db.View(func(txn *Txn) error {
			_, err := txn.Get(small)
			require.Equal(t, ErrKeyNotFound, err)
			return nil
		}))
	})
}

func TestIteratorPrefetchSize(t *testing.T) {
	runBadgerTest(t, nil, func(t *testing.T, db *DB) {

		bkey := func(i int) []byte {
			return []byte(fmt.Sprintf("%09d", i))
		}
		bval := func(i int) []byte {
			return []byte(fmt.Sprintf("%025d", i))
		}

		n := 100
		for i := 0; i < n; i++ {
			// if (i % 10) == 0 {
			// 	t.Logf("Put i=%d\n", i)
			// }
			txnSet(t, db, bkey(i), bval(i), byte(i%127))
		}

		getIteratorCount := func(prefetchSize int) int {
			opt := IteratorOptions{}
			opt.PrefetchValues = true
			opt.PrefetchSize = prefetchSize

			var count int
			txn := db.NewTransaction(false)
			it := txn.NewIterator(opt)
			{
				t.Log("Starting first basic iteration")
				for it.Rewind(); it.Valid(); it.Next() {
					count++
				}
				require.EqualValues(t, n, count)
			}
			return count
		}

		var sizes = []int{-10, 0, 1, 10}
		for _, size := range sizes {
			c := getIteratorCount(size)
			require.Equal(t, 100, c)
		}
	})
}

func TestSetIfAbsentAsync(t *testing.T) {
	dir, err := ioutil.TempDir("", "badger")
	require.NoError(t, err)
	defer os.RemoveAll(dir)
	kv, _ := Open(getTestOptions(dir))

	bkey := func(i int) []byte {
		return []byte(fmt.Sprintf("%09d", i))
	}

	f := func(err error) {}

	n := 1000
	for i := 0; i < n; i++ {
		// if (i % 10) == 0 {
		// 	t.Logf("Put i=%d\n", i)
		// }
		txn := kv.NewTransaction(true)
		_, err = txn.Get(bkey(i))
		require.Equal(t, ErrKeyNotFound, err)
		require.NoError(t, txn.SetWithMeta(bkey(i), nil, byte(i%127)))
		require.NoError(t, txn.Commit(f))
	}

	require.NoError(t, kv.Close())
	kv, err = Open(getTestOptions(dir))
	require.NoError(t, err)

	opt := DefaultIteratorOptions
	txn := kv.NewTransaction(false)
	var count int
	it := txn.NewIterator(opt)
	{
		t.Log("Starting first basic iteration")
		for it.Rewind(); it.Valid(); it.Next() {
			count++
		}
		require.EqualValues(t, n, count)
	}
	require.Equal(t, n, count)
	require.NoError(t, kv.Close())
}

func TestGetSetRace(t *testing.T) {
	runBadgerTest(t, nil, func(t *testing.T, db *DB) {

		data := make([]byte, 4096)
		_, err := rand.Read(data)
		require.NoError(t, err)

		var (
			numOp = 100
			wg    sync.WaitGroup
			keyCh = make(chan string)
		)

		// writer
		wg.Add(1)
		go func() {
			defer func() {
				wg.Done()
				close(keyCh)
			}()

			for i := 0; i < numOp; i++ {
				key := fmt.Sprintf("%d", i)
				txnSet(t, db, []byte(key), data, 0x00)
				keyCh <- key
			}
		}()

		// reader
		wg.Add(1)
		go func() {
			defer wg.Done()

			for key := range keyCh {
				require.NoError(t, db.View(func(txn *Txn) error {
					item, err := txn.Get([]byte(key))
					require.NoError(t, err)
					_, err = item.Value()
					require.NoError(t, err)
					return nil
				}))
			}
		}()

		wg.Wait()
	})
}

func TestPurgeVersionsBelow(t *testing.T) {
	runBadgerTest(t, nil, func(t *testing.T, db *DB) {
		// Write 4 versions of the same key
		for i := 0; i < 4; i++ {
			err := db.Update(func(txn *Txn) error {
				return txn.Set([]byte("answer"), []byte(fmt.Sprintf("%25d", i)))
			})
			require.NoError(t, err)
		}

		opts := DefaultIteratorOptions
		opts.AllVersions = true
		opts.PrefetchValues = false

		// Verify that there are 4 versions, and record 3rd version (2nd from top in iteration)
		var ts uint64
		db.View(func(txn *Txn) error {
			it := txn.NewIterator(opts)
			var count int
			for it.Rewind(); it.Valid(); it.Next() {
				count++
				item := it.Item()
				if count == 2 {
					ts = item.Version()
				}
				require.Equal(t, []byte("answer"), item.Key())
			}
			require.Equal(t, 4, count)
			return nil
		})

		// Delete all versions below the 3rd version
		err := db.PurgeVersionsBelow([]byte("answer"), ts)
		require.NoError(t, err)
		require.NotEmpty(t, db.vlog.lfDiscardStats.m)

		// Verify that there are only 2 versions left, and versions
		// below ts have been deleted.
		db.View(func(txn *Txn) error {
			it := txn.NewIterator(opts)
			var count int
			for it.Rewind(); it.Valid(); it.Next() {
				item := it.Item()
				if item.Version() < ts {
					require.True(t, item.meta&bitDelete > 0)
				} else {
					count++
				}
				require.Equal(t, []byte("answer"), item.Key())
			}
			require.Equal(t, 2, count)
			return nil
		})
	})
}

func TestPurgeVersionsBelow2(t *testing.T) {
	runBadgerTest(t, nil, func(t *testing.T, db *DB) {
		// Do a set and delete of same key
		err := db.Update(func(txn *Txn) error {
			return txn.Set([]byte("key"), []byte("value"))
		})
		require.NoError(t, err)

		err = db.Update(func(txn *Txn) error {
			return txn.Delete([]byte("key"))
		})
		require.NoError(t, err)

		opts := DefaultIteratorOptions
		opts.AllVersions = true
		opts.PrefetchValues = false
		// Verify that there are 2 versions and record highest version
		var ts uint64
		db.View(func(txn *Txn) error {
			it := txn.NewIterator(opts)
			var count int
			for it.Rewind(); it.Valid(); it.Next() {
				count++
				item := it.Item()
				require.Equal(t, []byte("key"), item.Key())
				if count == 1 {
					ts = item.Version()
					require.True(t, item.meta&bitDelete > 0)
					continue
				}
				val, err := item.Value()
				require.NoError(t, err)
				require.Equal(t, val, []byte("value"))
			}
			require.Equal(t, 2, count)
			return nil
		})

		// Delete all versions
		err = db.PurgeVersionsBelow([]byte("key"), ts)
		require.NoError(t, err)

		// Verify everything has been deleted
		db.View(func(txn *Txn) error {
			it := txn.NewIterator(opts)
			var count int
			for it.Rewind(); it.Valid(); it.Next() {
				item := it.Item()
				require.Equal(t, []byte("key"), item.Key())
				require.True(t, item.meta&bitDelete > 0)
				count++
			}
			require.Equal(t, 2, count)
			return nil
		})
	})
}

func TestPurgeOlderVersions(t *testing.T) {
	runBadgerTest(t, nil, func(t *testing.T, db *DB) {
		// Write two versions of a key
		err := db.Update(func(txn *Txn) error {
			return txn.Set([]byte("answer"), []byte("42"))
		})
		require.NoError(t, err)

		err = db.Update(func(txn *Txn) error {
			return txn.Set([]byte("answer"), []byte("43"))
		})
		require.NoError(t, err)

		opts := DefaultIteratorOptions
		opts.AllVersions = true
		opts.PrefetchValues = false

		// Verify that two versions are found during iteration
		err = db.View(func(txn *Txn) error {
			it := txn.NewIterator(opts)
			var count int
			for it.Rewind(); it.Valid(); it.Next() {
				count++
				item := it.Item()
				require.Equal(t, []byte("answer"), item.Key())
			}
			require.Equal(t, 2, count)
			return nil
		})
		require.NoError(t, err)

		// Invoke DeleteOlderVersions() to delete older version
		err = db.PurgeOlderVersions()
		require.NoError(t, err)

		// Verify that only one non-deleted version is found
		err = db.View(func(txn *Txn) error {
			it := txn.NewIterator(opts)
			var count int
			for it.Rewind(); it.Valid(); it.Next() {
				count++
				item := it.Item()
				require.Equal(t, []byte("answer"), item.Key())
				if count == 1 {
					val, err := item.Value()
					require.NoError(t, err)
					require.Equal(t, []byte("43"), val)
				} else {
					require.True(t, item.meta&bitDelete > 0)
				}
			}
			require.Equal(t, 2, count)
			return nil
		})
		require.NoError(t, err)
	})
}

func TestPurgeOlderVersions2(t *testing.T) {
	runBadgerTest(t, nil, func(t *testing.T, db *DB) {
		// Do a set and delete of same key
		err := db.Update(func(txn *Txn) error {
			return txn.Set([]byte("key"), []byte("value"))
		})
		require.NoError(t, err)

		err = db.Update(func(txn *Txn) error {
			return txn.Delete([]byte("key"))
		})
		require.NoError(t, err)

		opts := DefaultIteratorOptions
		opts.AllVersions = true
		opts.PrefetchValues = false
		// Verify that there are 2 versions
		db.View(func(txn *Txn) error {
			it := txn.NewIterator(opts)
			var count int
			for it.Rewind(); it.Valid(); it.Next() {
				count++
				item := it.Item()
				require.Equal(t, []byte("key"), item.Key())
				if count == 1 {
					require.True(t, item.meta&bitDelete > 0)
					continue
				}
				val, err := item.Value()
				require.NoError(t, err)
				require.Equal(t, val, []byte("value"))
			}
			require.Equal(t, 2, count)
			return nil
		})

		// Delete all versions
		err = db.PurgeOlderVersions()
		require.NoError(t, err)

		// Verify everything has been deleted
		db.View(func(txn *Txn) error {
			it := txn.NewIterator(opts)
			var count int
			for it.Rewind(); it.Valid(); it.Next() {
				item := it.Item()
				require.Equal(t, []byte("key"), item.Key())
				require.True(t, item.meta&bitDelete > 0)
				count++
			}
			require.Equal(t, 2, count)
			return nil
		})
	})
}

func TestExpiry(t *testing.T) {
	runBadgerTest(t, nil, func(t *testing.T, db *DB) {
		// Write two keys, one with a TTL
		err := db.Update(func(txn *Txn) error {
			return txn.Set([]byte("answer1"), []byte("42"))
		})
		require.NoError(t, err)

		err = db.Update(func(txn *Txn) error {
			return txn.SetWithTTL([]byte("answer2"), []byte("43"), 1*time.Second)
		})
		require.NoError(t, err)

		time.Sleep(2 * time.Second)

		// Verify that only unexpired key is found during iteration
		err = db.View(func(txn *Txn) error {
			_, err := txn.Get([]byte("answer1"))
			require.NoError(t, err)

			_, err = txn.Get([]byte("answer2"))
			require.Error(t, ErrKeyNotFound, err)
			return nil
		})
		require.NoError(t, err)

		// Verify that only one key is found during iteration
		opts := DefaultIteratorOptions
		opts.PrefetchValues = false
		err = db.View(func(txn *Txn) error {
			it := txn.NewIterator(opts)
			var count int
			for it.Rewind(); it.Valid(); it.Next() {
				count++
				item := it.Item()
				require.Equal(t, []byte("answer1"), item.Key())
			}
			require.Equal(t, 1, count)
			return nil
		})
		require.NoError(t, err)
	})
}

func randBytes(n int) []byte {
	recv := make([]byte, n)
	in, err := rand.Read(recv)
	if err != nil {
		log.Fatal(err)
	}
	return recv[:in]
}

var benchmarkData = []struct {
	key, value []byte
}{
	{randBytes(100), nil},
	{randBytes(1000), []byte("foo")},
	{[]byte("foo"), randBytes(1000)},
	{[]byte(""), randBytes(1000)},
	{nil, randBytes(1000000)},
	{randBytes(100000), nil},
	{randBytes(1000000), nil},
}

func TestLargeKeys(t *testing.T) {
	dir, err := ioutil.TempDir("", "badger")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	opts := new(Options)
	*opts = DefaultOptions
	opts.ValueLogFileSize = 1024 * 1024 * 1024
	opts.Dir = dir
	opts.ValueDir = dir

	db, err := Open(*opts)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 1000; i++ {
		tx := db.NewTransaction(true)
		for _, kv := range benchmarkData {
			k := make([]byte, len(kv.key))
			copy(k, kv.key)

			v := make([]byte, len(kv.value))
			copy(v, kv.value)
			if err := tx.Set(k, v); err != nil {
				// Skip over this record.
			}
		}
		if err := tx.Commit(nil); err != nil {
			t.Fatalf("#%d: batchSet err: %v", i, err)
		}
	}
}

func TestCreateDirs(t *testing.T) {
	dir, err := ioutil.TempDir("", "parent")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	opts := DefaultOptions
	dir = filepath.Join(dir, "badger")
	opts.Dir = dir
	opts.ValueDir = dir
	db, err := Open(opts)
	require.NoError(t, err)
	db.Close()
	_, err = os.Stat(dir)
	require.NoError(t, err)
}
func TestWriteDeadlock(t *testing.T) {
	dir, err := ioutil.TempDir("", "badger")
	fmt.Println(dir)
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	opt := DefaultOptions
	opt.Dir = dir
	opt.ValueDir = dir
	opt.ValueLogFileSize = 10 << 20
	db, err := Open(opt)
	require.NoError(t, err)

	print := func(count *int) {
		*count++
		if *count%100 == 0 {
			fmt.Printf("%05d\r", *count)
		}
	}

	var count int
	val := make([]byte, 10000)
	require.NoError(t, db.Update(func(txn *Txn) error {
		for i := 0; i < 1500; i++ {
			key := fmt.Sprintf("%d", i)
			rand.Read(val)
			require.NoError(t, txn.Set([]byte(key), val))
			print(&count)
		}
		return nil
	}))

	count = 0
	fmt.Println("\nWrites done. Iteration and updates starting...")
	err = db.Update(func(txn *Txn) error {
		opt := DefaultIteratorOptions
		opt.PrefetchValues = false
		it := txn.NewIterator(opt)
		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()

			// Using Value() would cause deadlock.
			// item.Value()
			out, err := item.ValueCopy(nil)
			require.NoError(t, err)
			require.Equal(t, len(val), len(out))

			key := y.Copy(item.Key())
			rand.Read(val)
			require.NoError(t, txn.Set(key, val))
			print(&count)
		}
		return nil
	})
	require.NoError(t, err)
}

func TestSequence(t *testing.T) {
	key0 := []byte("seq0")
	key1 := []byte("seq1")

	runBadgerTest(t, nil, func(t *testing.T, db *DB) {
		seq0, err := db.GetSequence(key0, 10)
		require.NoError(t, err)
		seq1, err := db.GetSequence(key1, 100)
		require.NoError(t, err)

		for i := uint64(0); i < uint64(105); i++ {
			num, err := seq0.Next()
			require.NoError(t, err)
			require.Equal(t, i, num)

			num, err = seq1.Next()
			require.NoError(t, err)
			require.Equal(t, i, num)
		}
		err = db.View(func(txn *Txn) error {
			item, err := txn.Get(key0)
			if err != nil {
				return err
			}
			val, err := item.Value()
			if err != nil {
				return err
			}
			num0 := binary.BigEndian.Uint64(val)
			require.Equal(t, uint64(110), num0)

			item, err = txn.Get(key1)
			if err != nil {
				return err
			}
			val, err = item.Value()
			if err != nil {
				return err
			}
			num1 := binary.BigEndian.Uint64(val)
			require.Equal(t, uint64(200), num1)
			return nil
		})
		require.NoError(t, err)
	})
}

func TestSequence_Release(t *testing.T) {
	runBadgerTest(t, nil, func(t *testing.T, db *DB) {
		// get sequence, use once and release
		key := []byte("key")
		seq, err := db.GetSequence(key, 1000)
		require.NoError(t, err)
		num, err := seq.Next()
		require.NoError(t, err)
		require.Equal(t, uint64(0), num)
		require.NoError(t, seq.Release())

		// we used up 0 and 1 should be stored now
		err = db.View(func(txn *Txn) error {
			item, err := txn.Get(key)
			if err != nil {
				return err
			}
			val, err := item.Value()
			if err != nil {
				return err
			}
			require.Equal(t, num+1, binary.BigEndian.Uint64(val))
			return nil
		})
		require.NoError(t, err)

		// using it again will lease 1+1000
		num, err = seq.Next()
		require.NoError(t, err)
		require.Equal(t, uint64(1), num)
		err = db.View(func(txn *Txn) error {
			item, err := txn.Get(key)
			if err != nil {
				return err
			}
			val, err := item.Value()
			if err != nil {
				return err
			}
			require.Equal(t, uint64(1001), binary.BigEndian.Uint64(val))
			return nil
		})
		require.NoError(t, err)
	})
}

func uint64ToBytes(i uint64) []byte {
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], i)
	return buf[:]
}

func bytesToUint64(b []byte) uint64 {
	return binary.BigEndian.Uint64(b)
}

// Merge function to add two uint64 numbers
func add(existing, new []byte) []byte {
	return uint64ToBytes(
		bytesToUint64(existing) +
			bytesToUint64(new))
}

func TestMergeOperatorGetBeforeAdd(t *testing.T) {
	key := []byte("merge")
	runBadgerTest(t, nil, func(t *testing.T, db *DB) {
		m := db.GetMergeOperator(key, add, 200*time.Millisecond)
		defer m.Stop()

		_, err := m.Get()
		require.Error(t, ErrKeyNotFound, err)
	})
}

func TestMergeOperatorBeforeAdd(t *testing.T) {
	key := []byte("merge")
	runBadgerTest(t, nil, func(t *testing.T, db *DB) {
		m := db.GetMergeOperator(key, add, 200*time.Millisecond)
		defer m.Stop()
		time.Sleep(time.Second)
	})
}

func TestMergeOperatorAddAndGet(t *testing.T) {
	key := []byte("merge")
	runBadgerTest(t, nil, func(t *testing.T, db *DB) {
		m := db.GetMergeOperator(key, add, 200*time.Millisecond)
		defer m.Stop()

		err := m.Add(uint64ToBytes(1))
		require.NoError(t, err)
		m.Add(uint64ToBytes(2))
		require.NoError(t, err)
		m.Add(uint64ToBytes(3))
		require.NoError(t, err)

		res, err := m.Get()
		require.NoError(t, err)
		require.Equal(t, uint64(6), bytesToUint64(res))
	})
}

func TestMergeOperatorCompactBeforeGet(t *testing.T) {
	key := []byte("merge")
	runBadgerTest(t, nil, func(t *testing.T, db *DB) {
		m := db.GetMergeOperator(key, add, 200*time.Millisecond)
		defer m.Stop()

		err := m.Add(uint64ToBytes(1))
		require.NoError(t, err)
		m.Add(uint64ToBytes(2))
		require.NoError(t, err)
		m.Add(uint64ToBytes(3))
		require.NoError(t, err)

		time.Sleep(250 * time.Millisecond) // wait for merge to happen

		res, err := m.Get()
		require.NoError(t, err)
		require.Equal(t, uint64(6), bytesToUint64(res))
	})
}

func TestMergeOperatorGetAfterStop(t *testing.T) {
	key := []byte("merge")
	runBadgerTest(t, nil, func(t *testing.T, db *DB) {
		m := db.GetMergeOperator(key, add, 1*time.Second)

		err := m.Add(uint64ToBytes(1))
		require.NoError(t, err)
		m.Add(uint64ToBytes(2))
		require.NoError(t, err)
		m.Add(uint64ToBytes(3))
		require.NoError(t, err)

		m.Stop()
		res, err := m.Get()
		require.NoError(t, err)
		require.Equal(t, uint64(6), bytesToUint64(res))
	})
}

func ExampleOpen() {
	dir, err := ioutil.TempDir("", "badger")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(dir)
	opts := DefaultOptions
	opts.Dir = dir
	opts.ValueDir = dir
	db, err := Open(opts)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	err = db.View(func(txn *Txn) error {
		_, err := txn.Get([]byte("key"))
		// We expect ErrKeyNotFound
		fmt.Println(err)
		return nil
	})

	if err != nil {
		log.Fatal(err)
	}

	txn := db.NewTransaction(true) // Read-write txn
	err = txn.Set([]byte("key"), []byte("value"))
	if err != nil {
		log.Fatal(err)
	}
	err = txn.Commit(nil)
	if err != nil {
		log.Fatal(err)
	}

	err = db.View(func(txn *Txn) error {
		item, err := txn.Get([]byte("key"))
		if err != nil {
			return err
		}
		val, err := item.Value()
		if err != nil {
			return err
		}
		fmt.Printf("%s\n", string(val))
		return nil
	})

	if err != nil {
		log.Fatal(err)
	}

	// Output:
	// Key not found
	// value
}

func ExampleTxn_NewIterator() {
	dir, err := ioutil.TempDir("", "badger")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(dir)

	opts := DefaultOptions
	opts.Dir = dir
	opts.ValueDir = dir

	db, err := Open(opts)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	bkey := func(i int) []byte {
		return []byte(fmt.Sprintf("%09d", i))
	}
	bval := func(i int) []byte {
		return []byte(fmt.Sprintf("%025d", i))
	}

	txn := db.NewTransaction(true)

	// Fill in 1000 items
	n := 1000
	for i := 0; i < n; i++ {
		err := txn.Set(bkey(i), bval(i))
		if err != nil {
			log.Fatal(err)
		}
	}

	err = txn.Commit(nil)
	if err != nil {
		log.Fatal(err)
	}

	opt := DefaultIteratorOptions
	opt.PrefetchSize = 10

	// Iterate over 1000 items
	var count int
	err = db.View(func(txn *Txn) error {
		it := txn.NewIterator(opt)
		for it.Rewind(); it.Valid(); it.Next() {
			count++
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Counted %d elements", count)
	// Output:
	// Counted 1000 elements
}
