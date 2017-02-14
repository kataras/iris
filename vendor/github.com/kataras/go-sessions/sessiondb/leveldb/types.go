package leveldb

import (
	"sync"
	"time"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
)

const (
	_DefaultCleanTimeout = time.Hour
	_DefaultMaxAge       = time.Second * 31556926
)

// Interface is an interface
type Interface interface {
	// Load loads the values
	Load(string) map[string]interface{}
	// Update updates the store
	Update(string, map[string]interface{})
}

// impl is an implementation
type impl struct {
	DB          *leveldb.DB
	Cfg         Config
	Err         error
	doCloseUp   chan bool
	doCloseDone sync.WaitGroup
}

// Config the leveldb configuration used inside sessions
type Config struct {
	// Path to leveldb database
	Path string
	// Options LevelDB open database options
	Options *Options
	// ReadOptions LevelDB read database options
	ReadOptions *ReadOptions
	// WriteOptions LevelDB write database options
	WriteOptions *WriteOptions
	// CleanTimeout Waiting between starting database cleaning and compression. Default one hour
	CleanTimeout time.Duration
	// MaxAge how much long the LevelDB should keep the session. Default 1 year
	MaxAge time.Duration
}

// Options LevelDB open database options
type Options opt.Options

// ReadOptions LevelDB read database options
type ReadOptions opt.ReadOptions

// WriteOptions LevelDB write database options
type WriteOptions opt.WriteOptions
