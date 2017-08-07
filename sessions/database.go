package sessions

import (
	"bytes"
	"encoding/gob"
	"io"
	"sync"

	"github.com/kataras/iris/core/memstore"
)

func init() {
	gob.Register(RemoteStore{})
}

// Database is the interface which all session databases should implement
// By design it doesn't support any type of cookie session like other frameworks.
// I want to protect you, believe me.
// The scope of the database is to store somewhere the sessions in order to
// keep them after restarting the server, nothing more.
//
// Synchronization are made automatically, you can register more than one session database
// but the first non-empty Load return data will be used as the session values.
//
//
// Note: Expiration on Load is up to the database, meaning that:
// the database can decide how to retrieve and parse the expiration datetime
//
// I'll try to explain you the flow:
//
// .Start -> if session database attached then load from that storage and save to the memory, otherwise load from memory. The load from database is done once on the initialize of each session.
// .Get (important) -> load from memory,
//                     if database attached then it already loaded the values
//                     from database on the .Start action, so it will
//                     retrieve the data from the memory (fast)
// .Set -> set to the memory, if database attached then update the storage
// .Delete -> clear from memory, if database attached then update the storage
// .Destroy -> destroy from memory and client cookie,
//             if database attached then update the storage with empty values,
//             empty values means delete the storage with that specific session id.
// Using everything else except memory is slower than memory but database is
// fetched once at each session and its updated on every Set, Delete,
// Destroy at call-time.
// All other external sessions managers out there work different than Iris one as far as I know,
// you may find them more suited to your application, it depends.
type Database interface {
	Load(sid string) RemoteStore
	Sync(p SyncPayload)
}

// New Idea, it should work faster for the most databases needs
// the only minus is that the databases is coupled with this package, they
// should import the kataras/iris/sessions package, but we don't use any
// database by-default so that's ok here.

// Action reports the specific action that the memory store
// sends to the database.
type Action uint32

const (
	// ActionCreate occurs when add a key-value pair
	// on the database session entry for the first time.
	ActionCreate Action = iota
	// ActionInsert occurs when add a key-value pair
	// on the database session entry.
	ActionInsert
	// ActionUpdate occurs when modify an existing key-value pair
	// on the database session entry.
	ActionUpdate
	// ActionDelete occurs when delete a specific value from
	// a specific key from the database session entry.
	ActionDelete
	// ActionClear occurs when clear all values but keep the database session entry.
	ActionClear
	// ActionDestroy occurs when destroy,
	// destroy is the action when clear all and remove the session entry from the database.
	ActionDestroy
)

// SyncPayload reports the state of the session inside a database sync action.
type SyncPayload struct {
	SessionID string

	Action Action
	// on insert it contains the new key and the value
	// on update it contains the existing key and the new value
	// on delete it contains the key (the value is nil)
	// on clear it contains nothing (empty key, value is nil)
	// on destroy it contains nothing (empty key, value is nil)
	Value memstore.Entry
	// Store contains the whole memory store, this store
	// contains the current, updated from memory calls,
	// session data (keys and values). This way
	// the database has access to the whole session's data
	// every time.
	Store RemoteStore
}

var spPool = sync.Pool{New: func() interface{} { return SyncPayload{} }}

func acquireSyncPayload(session *Session, action Action) SyncPayload {
	p := spPool.Get().(SyncPayload)
	p.SessionID = session.sid

	// clone the life time, except the timer.
	// lifetime := LifeTime{
	// 	Time:             session.lifetime.Time,
	// 	OriginalDuration: session.lifetime.OriginalDuration,
	// }

	// lifetime := acquireLifetime(session.lifetime.OriginalDuration, nil)

	p.Store = RemoteStore{
		Values:   session.values,
		Lifetime: session.lifetime,
	}

	p.Action = action
	return p
}

func releaseSyncPayload(p SyncPayload) {
	p.Value.Key = ""
	p.Value.ValueRaw = nil

	// releaseLifetime(p.Store.Lifetime)
	spPool.Put(p)
}

func syncDatabases(databases []Database, payload SyncPayload) {
	for i, n := 0, len(databases); i < n; i++ {
		databases[i].Sync(payload)
	}
	releaseSyncPayload(payload)
}

// RemoteStore is a helper which is a wrapper
// for the store, it can be used as the session "table" which will be
// saved to the session database.
type RemoteStore struct {
	// Values contains the whole memory store, this store
	// contains the current, updated from memory calls,
	// session data (keys and values). This way
	// the database has access to the whole session's data
	// every time.
	Values memstore.Store
	// on insert it contains the expiration datetime
	// on update it contains the new expiration datetime(if updated or the old one)
	// on delete it will be zero
	// on clear it will be zero
	// on destroy it will be zero
	Lifetime LifeTime
}

// Serialize returns the byte representation of this RemoteStore.
func (s RemoteStore) Serialize() ([]byte, error) {
	w := new(bytes.Buffer)
	err := encode(s, w)
	return w.Bytes(), err
}

// encode accepts a store and writes
// as series of bytes to the "w" writer.
func encode(s RemoteStore, w io.Writer) error {
	enc := gob.NewEncoder(w)
	err := enc.Encode(s)
	return err
}

// DecodeRemoteStore accepts a series of bytes and returns
// the store.
func DecodeRemoteStore(b []byte) (store RemoteStore, err error) {
	dec := gob.NewDecoder(bytes.NewBuffer(b))
	err = dec.Decode(&store)
	return
}
