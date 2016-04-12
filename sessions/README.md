> This package is converted to work with Iris but it was originaly created by Gorila team, the original package is gorilla/sessions

The package has fully ported to work with fasthttp and iris. It's the first attempt for session with fasthttp, which I know of, so if you find any bugs please post an [issue here](https://github.com/kataras/iris/issues) 

The key features are:

* Simple API: use it as an easy way to set signed (and optionally
  encrypted) cookies.
* Built-in backends to store sessions in cookies or the filesystem.
* Flash messages: session values that last until read.
* Convenient way to switch session persistency (aka "remember me") and set
  other attributes.
* Mechanism to rotate authentication and encryption keys.
* Multiple sessions per request, even using different backends.
* Interfaces and infrastructure for custom session backends: sessions from
  different stores can be retrieved and batch-saved using a common API.


## Usage

```go

package main

import (
	"github.com/kataras/iris"
	"github.com/kataras/iris/sessions"
)

//there is no middleware, use the sessions anywhere you want
func main() {

	var store = sessions.NewCookieStore([]byte("myIrisSecretKey"))
	var mySessions = sessions.New("user_sessions", store)

	iris.Get("/set", func(c *iris.Context) {
		//get the session for this context
		session, err := mySessions.Get(c) // or .GetSession(c), it's the same 

		if err != nil {
			c.SendStatus(500, err.Error())
			return
		}
		//set session values
		session.Set("name", "kataras")

		//save them
		session.Save(c)

		//write anthing
		c.Write("All ok session setted to: %s", session.Get("name"))
	})

	iris.Get("/get", func(c *iris.Context) {
		//again get the session for this context
		session, err := mySessions.Get(c)

		if err != nil {
			c.SendStatus(500, err.Error())
			return
		}
		//get the session value
		name := session.GetString("name") // .Get or .GetInt

		c.Write("The name on the /set was: %s", name)
	})

	iris.Get("/clear", func(c *iris.Context) {
		session, err := mySessions.Get(c)
		if err != nil {
			c.SendStatus(500, err.Error())
			return
		}
		//Clear clears all
		//session.Clear()
		session.Delete("name")

	})

	// Use global sessions.Clear() to clear ALL sessions and stores if it's necessary
	//sessions.Clear()

	println("Iris is listening on :8080")
	iris.Listen("8080")

}


```

## Store Implementations

Other implementations of the `sessions.Store` interface:

* [github.com/starJammer/gorilla-sessions-arangodb](https://github.com/starJammer/gorilla-sessions-arangodb) - ArangoDB
* [github.com/yosssi/boltstore](https://github.com/yosssi/boltstore) - Bolt
* [github.com/srinathgs/couchbasestore](https://github.com/srinathgs/couchbasestore) - Couchbase
* [github.com/denizeren/dynamostore](https://github.com/denizeren/dynamostore) - Dynamodb on AWS
* [github.com/bradleypeabody/gorilla-sessions-memcache](https://github.com/bradleypeabody/gorilla-sessions-memcache) - Memcache
* [github.com/hnakamur/gaesessions](https://github.com/hnakamur/gaesessions) - Memcache on GAE
* [github.com/kidstuff/mongostore](https://github.com/kidstuff/mongostore) - MongoDB
* [github.com/srinathgs/mysqlstore](https://github.com/srinathgs/mysqlstore) - MySQL
* [github.com/antonlindstrom/pgstore](https://github.com/antonlindstrom/pgstore) - PostgreSQL
* [github.com/boj/redistore](https://github.com/boj/redistore) - Redis
* [github.com/boj/rethinkstore](https://github.com/boj/rethinkstore) - RethinkDB
* [github.com/boj/riakstore](https://github.com/boj/riakstore) - Riak
* [github.com/michaeljs1990/sqlitestore](https://github.com/michaeljs1990/sqlitestore) - SQLite
* [github.com/wader/gormstore](https://github.com/wader/gormstore) - GORM (MySQL, PostgreSQL, SQLite)


