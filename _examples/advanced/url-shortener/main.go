// Package main shows how you can create a simple URL SHortener using only Iris and BoltDB.
//
// $ go get github.com/boltdb/bolt/...
// $ go run main.go
// $ start http://localhost:8080
package main

import (
	"bytes"
	"html/template"
	"math/rand"
	"net/url"
	"time"

	"github.com/boltdb/bolt"

	"github.com/cdren/iris"
	"github.com/cdren/iris/context"
	"github.com/cdren/iris/view"
)

func main() {
	app := iris.New()

	// assign a variable to the DB so we can use its features later
	db := NewDB("shortener.db")
	factory := NewFactory(DefaultGenerator, db)

	// serve the "./templates" directory's "*.html" files with the HTML std view engine.
	tmpl := view.HTML("./templates", ".html").Reload(true)
	// template funcs
	//
	// look ./templates/index.html#L16
	tmpl.AddFunc("isPositive", func(n int) bool {
		if n > 0 {
			return true
		}
		return false
	})

	app.AttachView(tmpl)

	// Serve static files (css)
	app.StaticWeb("/static", "./resources")

	app.Get("/", func(ctx context.Context) {
		ctx.ViewData("url_count", db.Len())
		ctx.View("index.html")
	})

	// find and execute a short url by its key
	// used on http://localhost:8080/u/dsaoj41u321dsa
	execShortURL := func(ctx context.Context, key string) {
		if key == "" {
			ctx.StatusCode(iris.StatusBadRequest)
			return
		}

		value := db.Get(key)
		if value == "" {
			ctx.StatusCode(iris.StatusNotFound)
			ctx.Writef("Short URL for key: '%s' not found", key)
			return
		}

		ctx.Redirect(value, iris.StatusTemporaryRedirect)
	}
	app.Get("/u/:shortkey", func(ctx context.Context) {
		execShortURL(ctx, ctx.Params().Get("shortkey"))
	})

	app.Post("/shorten", func(ctx context.Context) {
		formValue := ctx.FormValue("url")
		if formValue == "" {
			ctx.ViewData("form_result", "You need to a enter a URL")
		} else {
			key, err := factory.Gen(formValue)
			if err != nil {
				ctx.ViewData("form_result", "Invalid URL")
			} else {
				if err = db.Set(key, formValue); err != nil {
					ctx.ViewData("form_result", "Internal error while saving the URL")
					app.Log("while saving URL: " + err.Error())
				} else {
					ctx.StatusCode(iris.StatusOK)
					shortenURL := "http://" + app.ConfigurationReadOnly().GetVHost() + "/u/" + key
					ctx.ViewData("form_result",
						template.HTML("<pre><a target='_new' href='"+shortenURL+"'>"+shortenURL+" </a></pre>"))
				}

			}
		}
		ctx.ViewData("url_count", db.Len())
		ctx.View("index.html")
	})

	app.Run(iris.Addr(":8080"))

	db.Close()
}

//  +------------------------------------------------------------+
//  |                                                            |
//  |                      Store                                 |
//  |                                                            |
//  +------------------------------------------------------------+

// Panic panics, change it if you don't want to panic on critical INITIALIZE-ONLY-ERRORS
var Panic = func(v interface{}) {
	panic(v)
}

// Store is the store interface for urls.
// Note: no Del functionality.
type Store interface {
	Set(key string, value string) error // error if something went wrong
	Get(key string) string              // empty value if not found
	Len() int                           // should return the number of all the records/tables/buckets
	Close() error                       // release the store or ignore
}

var (
	tableURLs = []byte("urls")
)

// DB representation of a Store.
// Only one table/bucket which contains the urls, so it's not a fully Database,
// it works only with single bucket because that all we need.
type DB struct {
	db *bolt.DB
}

var _ Store = &DB{}

// openDatabase open a new database connection
// and returns its instance.
func openDatabase(stumb string) *bolt.DB {
	// Open the data(base) file in the current working directory.
	// It will be created if it doesn't exist.
	db, err := bolt.Open(stumb, 0600, nil)
	if err != nil {
		Panic(err)
	}

	// create the buckets here
	var tables = [...][]byte{
		tableURLs,
	}

	db.Update(func(tx *bolt.Tx) (err error) {
		for _, table := range tables {
			_, err = tx.CreateBucketIfNotExists(table)
			if err != nil {
				Panic(err)
			}
		}

		return
	})

	return db
}

// NewDB returns a new DB instance, its connection is opened.
// DB implements the Store.
func NewDB(stumb string) *DB {
	return &DB{
		db: openDatabase(stumb),
	}
}

// Set sets a shorten url and its key
// Note: Caller is responsible to generate a key.
func (d *DB) Set(key string, value string) error {
	d.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(tableURLs)
		// Generate ID for the url
		// Note: we could use that instead of a random string key
		// but we want to simulate a real-world url shortener
		// so we skip that.
		// id, _ := b.NextSequence()
		return b.Put([]byte(key), []byte(value))
	})
	return nil
}

// Get returns a url by its key.
//
// Returns an empty string if not found.
func (d *DB) Get(key string) (value string) {
	keyb := []byte(key)
	d.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(tableURLs)
		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			if bytes.Equal(keyb, k) {
				value = string(v)
				break
			}
		}

		return nil
	})

	return
}

// Len returns all the "shorted" urls length
func (d *DB) Len() (num int) {
	d.db.View(func(tx *bolt.Tx) error {

		// Assume bucket exists and has keys
		b := tx.Bucket(tableURLs)

		b.ForEach(func([]byte, []byte) error {
			num++
			return nil
		})
		return nil
	})
	return
}

// Close the data(base) connection
func (d *DB) Close() error {
	return d.db.Close()
}

//  +------------------------------------------------------------+
//  |                                                            |
//  |                      Factory                               |
//  |                                                            |
//  +------------------------------------------------------------+

// Generator the type to generate keys(short urls) based on 'n'
type Generator func(n int) string

// DefaultGenerator is the defautl url generator (the simple randomString)
var DefaultGenerator = randomString

// Factory is responsible to generate keys(short urls)
type Factory struct {
	store     Store
	generator Generator
}

// NewFactory receives a generator and a store and returns a new url Factory.
func NewFactory(generator Generator, store Store) *Factory {
	return &Factory{
		store:     store,
		generator: generator,
	}
}

// Gen generates the key.
func (f *Factory) Gen(uri string) (key string, err error) {
	// we don't return the parsed url because #hash are converted to uri-compatible
	// and we don't want to encode/decode all the time, there is no need for that,
	// we save the url as the user expects if the uri validation passed.
	_, err = url.ParseRequestURI(uri)
	if err != nil {
		return "", err
	}
	key = f.generator(len(uri))
	// Make sure that the key is unique
	for {
		if v := f.store.Get(key); v == "" {
			break
		}
		key = f.generator((len(uri) / 2) + 1)
	}

	return key, nil
}

const (
	letterBytes   = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

func randomString(n int) string {
	src := rand.NewSource(time.Now().UnixNano())
	b := make([]byte, n)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return string(b)
}
