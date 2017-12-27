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
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDumpLoad(t *testing.T) {
	dir, err := ioutil.TempDir("", "badger")
	require.NoError(t, err)
	defer os.RemoveAll(dir)
	db, err := Open(getTestOptions(dir))
	require.NoError(t, err)

	// Write some stuff
	entries := []struct {
		key      []byte
		val      []byte
		userMeta byte
		version  uint64
	}{
		{key: []byte("answer1"), val: []byte("42"), version: 1},
		{key: []byte("answer2"), val: []byte("43"), userMeta: 1, version: 1},
	}

	err = db.Update(func(txn *Txn) error {
		for _, e := range entries {
			err := txn.SetWithMeta(e.key, e.val, e.userMeta)
			if err != nil {
				return err
			}
		}
		return nil
	})
	require.NoError(t, err)

	bak, err := ioutil.TempFile(dir, "badgerbak")
	require.NoError(t, err)
	ts, err := db.Backup(bak, 0)
	t.Logf("New ts: %d\n", ts)
	require.NoError(t, err)
	require.NoError(t, bak.Close())
	require.NoError(t, db.Close())

	db, err = Open(getTestOptions(dir))
	require.NoError(t, err)
	defer db.Close()
	bak, err = os.Open(bak.Name())
	require.NoError(t, err)
	defer bak.Close()

	require.NoError(t, db.Load(bak))

	err = db.View(func(txn *Txn) error {
		opts := DefaultIteratorOptions
		opts.AllVersions = true
		it := txn.NewIterator(opts)
		var count int
		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			val, err := item.Value()
			if err != nil {
				return err
			}
			require.Equal(t, entries[count].key, item.Key())
			require.Equal(t, entries[count].val, val)
			require.Equal(t, entries[count].version, item.Version())
			require.Equal(t, entries[count].userMeta, item.UserMeta())
			count++
		}
		return nil
	})
	require.NoError(t, err)
}
