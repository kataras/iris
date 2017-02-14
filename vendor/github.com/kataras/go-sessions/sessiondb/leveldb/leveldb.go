package leveldb

import (
	"bytes"
	"encoding/gob"
	"runtime"
	"time"

	"github.com/kataras/go-sessions/sessiondb/leveldb/record"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/util"
)

func init() { gob.Register(record.Record{}) }

// New returns a database interface
func New(cfg ...Config) Interface {
	var ldb = new(impl)
	ldb.doCloseUp = make(chan bool)
	for i := range cfg {
		ldb.Cfg.Path = cfg[i].Path
		ldb.Cfg.Options = cfg[i].Options
		ldb.Cfg.ReadOptions = cfg[i].ReadOptions
		ldb.Cfg.WriteOptions = cfg[i].WriteOptions
		ldb.Cfg.CleanTimeout = cfg[i].CleanTimeout
		ldb.Cfg.MaxAge = cfg[i].MaxAge
	}
	if ldb.Cfg.CleanTimeout == 0 {
		ldb.Cfg.CleanTimeout = _DefaultCleanTimeout
	}
	if ldb.Cfg.MaxAge == 0 {
		ldb.Cfg.MaxAge = _DefaultMaxAge
	}
	ldb.OpenDB()
	go func() {
		ldb.doCloseDone.Add(1)
		defer ldb.doCloseDone.Done()
		ldb.Cleaner()
	}()
	runtime.SetFinalizer(ldb, destructor)
	return ldb
}

func destructor(ldb *impl) {
	if ldb.DB == nil {
		return
	}
	ldb.doCloseUp <- true
	ldb.doCloseDone.Wait()

	ldb.Err = ldb.DB.CompactRange(util.Range{Limit: nil, Start: nil})
	if ldb.Err != nil {
		println("Error compact database LevelDB(" + ldb.Cfg.Path + "): " + ldb.Err.Error())
	}
	ldb.Err = ldb.DB.Close()
	if ldb.Err != nil {
		println("Error close database LevelDB(" + ldb.Cfg.Path + "): " + ldb.Err.Error())
	}
	ldb.DB = nil
}

// Cleaner Goroutine for clean and compact LevelDB database
func (ldb *impl) Cleaner() {
	var exit bool
	var iter iterator.Iterator
	var timer = time.NewTimer(ldb.Cfg.CleanTimeout)
	defer func() { _ = timer.Stop() }()
	for {
		timer.Reset(ldb.Cfg.CleanTimeout)
		select {
		case <-ldb.doCloseUp:
			exit = true
		case <-timer.C:
		}
		if ldb.Err = ldb.DB.CompactRange(util.Range{Limit: nil, Start: nil}); ldb.Err != nil {
			println("Error compact database LevelDB(" + ldb.Cfg.Path + "): " + ldb.Err.Error())
		}
		// Clean old data
		iter = ldb.DB.NewIterator(nil, (*opt.ReadOptions)(ldb.Cfg.ReadOptions))
		for iter.Next() {
			if err := iter.Error(); err != nil {
				continue
			}
			ldb.clean(iter.Key(), iter.Value())
		}
		iter.Release()
		if exit {
			break
		}
	}
}

// Cleaning old and error data
func (ldb *impl) clean(key, value []byte) {
	var err error
	var rec record.Record
	var kill bool
	if err = DeserializeBytes(value, &rec); err != nil {
		kill = true // Cleaning erroneous entries
	}
	if time.Since(rec.DeathTime) > 0 {
		kill = true // Cleaning deceased entries
	}
	if kill {
		err = ldb.DB.Delete(key, (*opt.WriteOptions)(ldb.Cfg.WriteOptions))
		if err != nil {
			println("Error delete key='" + string(key) + "' from database LevelDB(" + ldb.Cfg.Path + "): " + err.Error())
		}
	}
}

// OpenDB Open LevelDB database
func (ldb *impl) OpenDB() {
	ldb.DB, ldb.Err = leveldb.OpenFile(ldb.Cfg.Path, (*opt.Options)(ldb.Cfg.Options))
	if ldb.Err != nil {
		return
	}
	if ldb.Err = ldb.DB.CompactRange(util.Range{Limit: nil, Start: nil}); ldb.Err != nil {
		println("Error compact database LevelDB(" + ldb.Cfg.Path + "): " + ldb.Err.Error())
		return
	}
	return
}

// Config returns the configuration
func (ldb *impl) Config() *Config { return &ldb.Cfg }

// Load loads the values to the underline
func (ldb *impl) Load(id string) (ret map[string]interface{}) {
	var ok bool
	var val []byte
	var rec record.Record
	ret = make(map[string]interface{})
	ok, ldb.Err = ldb.DB.Has([]byte(id), (*opt.ReadOptions)(ldb.Cfg.ReadOptions))
	if !ok {
		return
	}
	val, ldb.Err = ldb.DB.Get([]byte(id), (*opt.ReadOptions)(ldb.Cfg.ReadOptions))
	if ldb.Err != nil {
		return
	}
	if ldb.Err = DeserializeBytes(val, &rec); ldb.Err != nil {
		return
	}
	ldb.Err = DeserializeBytes(rec.Data, &ret)
	return
}

// Update updates the store
func (ldb *impl) Update(id string, values map[string]interface{}) {
	var err error
	var rec record.Record
	if len(values) == 0 {
		go func(id string) {
			if err := ldb.DB.Delete([]byte(id), (*opt.WriteOptions)(ldb.Cfg.WriteOptions)); err != nil {
				println("Error delete key='" + id + "' from database LevelDB(" + ldb.Cfg.Path + "): " + err.Error())
			}
		}(id)
	} else {
		if rec.Data, err = SerializeBytes(values); err != nil {
			println("Error serialize value for key='" + id + "': " + err.Error())
			return
		}
		go func(id string, rec record.Record) {
			var err error
			var val []byte
			rec.DeathTime = time.Now().In(time.Local).Add(ldb.Cfg.MaxAge)
			val, err = SerializeBytes(rec)
			if err = ldb.DB.Put([]byte(id), val, (*opt.WriteOptions)(ldb.Cfg.WriteOptions)); err != nil {
				println("Error put key='" + id + "' to database LevelDB(" + ldb.Cfg.Path + "): " + err.Error())
			}
		}(id, rec)
	}
}

// SerializeBytes serializa bytes using gob encoder and returns them
func SerializeBytes(m interface{}) (ret []byte, err error) {
	var buf = new(bytes.Buffer)
	var enc = gob.NewEncoder(buf)
	err = enc.Encode(m)
	if err != nil {
		return
	}
	ret = buf.Bytes()
	return
}

// DeserializeBytes converts the bytes to an object using gob decoder
func DeserializeBytes(b []byte, m interface{}) (err error) {
	err = gob.NewDecoder(bytes.NewBuffer(b)).Decode(m)
	return
}
