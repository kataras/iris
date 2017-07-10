// Package main shows how you can create a simple URL SHortener.
//
// $ go get github.com/boltdb/bolt/...
// $ go run main.go
// $ start http://localhost:8080
package main

import (
	"bytes"
	"html/template"
	"net/url"

	"github.com/kataras/iris"
	"github.com/kataras/iris/context"
	"github.com/kataras/iris/view"

	"github.com/boltdb/bolt"

	"github.com/satori/go.uuid"
)

func main() {
	// assign a variable to the DB so we can use its features later.
	db := NewDB("shortener.db")
	// Pass that db to our app, in order to be able to test the whole app with a different database later on.
	app := newApp(db)

	// release the "db" connection when server goes off.
	iris.RegisterOnInterrupt(db.Close)

	app.Run(iris.Addr(":8080"))
}

func newApp(db *DB) *iris.Application {
	app := iris.Default() // or app := iris.New()

	// create our factory, which is the manager for the object creation.
	// between our web app and the db.
	factory := NewFactory(DefaultGenerator, db)

	// serve the "./templates" directory's "*.html" files with the HTML std view engine.
	tmpl := view.HTML("./templates", ".html").Reload(true)
	// register any template func(s) here.
	//
	// Look ./templates/index.html#L16
	tmpl.AddFunc("IsPositive", func(n int) bool {
		if n > 0 {
			return true
		}
		return false
	})

	app.RegisterView(tmpl)

	// Serve static files (css)
	app.StaticWeb("/static", "./resources")

	indexHandler := func(ctx context.Context) {
		ctx.ViewData("URL_COUNT", db.Len())
		ctx.View("index.html")
	}
	app.Get("/", indexHandler)

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
	app.Get("/u/{shortkey}", func(ctx context.Context) {
		execShortURL(ctx, ctx.Params().Get("shortkey"))
	})

	app.Post("/shorten", func(ctx context.Context) {
		formValue := ctx.FormValue("url")
		if formValue == "" {
			ctx.ViewData("FORM_RESULT", "You need to a enter a URL")
			ctx.StatusCode(iris.StatusLengthRequired)
		} else {
			key, err := factory.Gen(formValue)
			if err != nil {
				ctx.ViewData("FORM_RESULT", "Invalid URL")
				ctx.StatusCode(iris.StatusBadRequest)
			} else {
				if err = db.Set(key, formValue); err != nil {
					ctx.ViewData("FORM_RESULT", "Internal error while saving the URL")
					app.Logger().Infof("while saving URL: " + err.Error())
					ctx.StatusCode(iris.StatusInternalServerError)
				} else {
					ctx.StatusCode(iris.StatusOK)
					shortenURL := "http://" + app.ConfigurationReadOnly().GetVHost() + "/u/" + key
					ctx.ViewData("FORM_RESULT",
						template.HTML("<pre><a target='_new' href='"+shortenURL+"'>"+shortenURL+" </a></pre>"))
				}

			}
		}

		indexHandler(ctx) // no redirect, we need the FORM_RESULT.
	})

	app.Post("/clear_cache", func(ctx context.Context) {
		db.Clear()
		ctx.Redirect("/")
	})

	return app
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
	Close()                             // release the store or ignore
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
	return d.db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(tableURLs)
		// Generate ID for the url
		// Note: we could use that instead of a random string key
		// but we want to simulate a real-world url shortener
		// so we skip that.
		// id, _ := b.NextSequence()
		if err != nil {
			return err
		}

		k := []byte(key)
		valueB := []byte(value)
		c := b.Cursor()

		found := false
		for k, v := c.First(); k != nil; k, v = c.Next() {
			if bytes.Equal(valueB, v) {
				found = true
				break
			}
		}
		// if value already exists don't re-put it.
		if found {
			return nil
		}

		return b.Put(k, []byte(value))
	})
}

// Clear clears all the database entries for the table urls.
func (d *DB) Clear() error {
	return d.db.Update(func(tx *bolt.Tx) error {
		return tx.DeleteBucket(tableURLs)
	})
}

// Get returns a url by its key.
//
// Returns an empty string if not found.
func (d *DB) Get(key string) (value string) {
	keyB := []byte(key)
	d.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(tableURLs)
		if b == nil {
			return nil
		}
		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			if bytes.Equal(keyB, k) {
				value = string(v)
				break
			}
		}

		return nil
	})

	return
}

// GetByValue returns all keys for a specific (original) url value.
func (d *DB) GetByValue(value string) (keys []string) {
	valueB := []byte(value)
	d.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(tableURLs)
		if b == nil {
			return nil
		}
		c := b.Cursor()
		// first for the bucket's table "urls"
		for k, v := c.First(); k != nil; k, v = c.Next() {
			if bytes.Equal(valueB, v) {
				keys = append(keys, string(k))
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
		if b == nil {
			return nil
		}

		b.ForEach(func([]byte, []byte) error {
			num++
			return nil
		})
		return nil
	})
	return
}

// Close shutdowns the data(base) connection.
func (d *DB) Close() {
	if err := d.db.Close(); err != nil {
		Panic(err)
	}
}

//  +------------------------------------------------------------+
//  |                                                            |
//  |                      Factory                               |
//  |                                                            |
//  +------------------------------------------------------------+

// Generator the type to generate keys(short urls)
type Generator func() string

// DefaultGenerator is the defautl url generator
var DefaultGenerator = func() string {
	return uuid.NewV4().String()
}

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

	key = f.generator()
	// Make sure that the key is unique
	for {
		if v := f.store.Get(key); v == "" {
			break
		}
		key = f.generator()
	}

	return key, nil
}
