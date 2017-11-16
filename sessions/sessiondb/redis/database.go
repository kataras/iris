package redis

import (
	"runtime"
	"time"

	"github.com/kataras/golog"
	"github.com/kataras/iris/sessions"
	"github.com/kataras/iris/sessions/sessiondb/redis/service"
)

// Database the redis back-end session database for the sessions.
type Database struct {
	redis *service.Service
}

// New returns a new redis database.
func New(cfg ...service.Config) *Database {
	db := &Database{redis: service.New(cfg...)}
	runtime.SetFinalizer(db, closeDB)
	return db
}

// Config returns the configuration for the redis server bridge, you can change them.
func (db *Database) Config() *service.Config {
	return db.redis.Config
}

// Async is DEPRECATED
// if it was true then it could use different to update the back-end storage, now it does nothing.
func (db *Database) Async(useGoRoutines bool) *Database {
	return db
}

// Load loads the values to the underline.
func (db *Database) Load(sid string) (storeDB sessions.RemoteStore) {
	// values := make(map[string]interface{})

	if !db.redis.Connected { //yes, check every first time's session for valid redis connection
		db.redis.Connect()
		_, err := db.redis.PingPong()
		if err != nil {
			golog.Errorf("redis database error on connect: %v", err)
			return
		}
	}

	// fetch the values from this session id and copy-> store them
	storeMaybe, err := db.redis.Get(sid)
	// exists
	if err == nil {
		storeB, ok := storeMaybe.([]byte)
		if !ok {
			golog.Errorf("something wrong, store should be stored as []byte but stored as %#v", storeMaybe)
			return
		}

		storeDB, err = sessions.DecodeRemoteStore(storeB) // decode the whole value, as a remote store
		if err != nil {
			golog.Errorf(`error while trying to load session values(%s) from redis:
			the retrieved value is not a sessions.RemoteStore type, please report that as bug, that should never occur: %v`,
				sid, err)
		}
	}

	return
}

// Sync syncs the database.
func (db *Database) Sync(p sessions.SyncPayload) {
	db.sync(p)
}

func (db *Database) sync(p sessions.SyncPayload) {
	if p.Action == sessions.ActionDestroy {
		db.redis.Delete(p.SessionID)
		return
	}
	storeB, err := p.Store.Serialize()
	if err != nil {
		golog.Error("error while encoding the remote session store")
		return
	}

	// not expire if zero
	seconds := 0

	if lifetime := p.Store.Lifetime; !lifetime.IsZero() {
		seconds = int(lifetime.Sub(time.Now()).Seconds())
	}

	db.redis.Set(p.SessionID, storeB, seconds)
}

// Close shutdowns the redis connection.
func (db *Database) Close() error {
	return closeDB(db)
}

func closeDB(db *Database) error {
	return db.redis.CloseConnection()
}
