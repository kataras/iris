## Package information

This package is totally new, if you find any bugs please [post an issue here](https://github.com/kataras/iris/issues)


## Usage: Low-level

```go

package main

import (
	"time"

	"github.com/kataras/iris"
	"github.com/kataras/iris/sessions"

	_ "github.com/kataras/iris/sessions/providers/memory" // here we add the memory session store
)

var sess *sessions.Manager

func init() {
	sess = sessions.New("memory", "irissessionid", time.Duration(60)*time.Minute)
}

func main() {

	iris.Get("/set", func(c *iris.Context) {
		//get the session for this context
		session := sess.Start(c)

		//set session values
		session.Set("name", "kataras")

		//test if setted here
		c.Write("All ok session setted to: %s", session.Get("name"))
	})

	iris.Get("/get", func(c *iris.Context) {
		//get the session for this context
		session := sess.Start(c)

		var name string

		//get the session value
		if v := session.Get("name"); v != nil {
			name = v.(string)
		}
		// OR just name = session.GetString("name")

		c.Write("The name on the /set was: %s", name)
	})

	iris.Get("/clear", func(c *iris.Context) {
		//get the session for this context
		session := sess.Start(c)

		session.Delete("name")
	})

	iris.Get("/destroy", func(c *iris.Context) {
		//destroy, removes the entire session and cookie
		sess.Destroy(c)
	})

	println("Server is listening at :8080")
	iris.Listen("8080")

}



```


## Security: Prevent session hijacking

> This section  was originally written on a book


**cookie only and token**

Through this simple example of hijacking a session, you can see that it's very dangerous because it allows attackers to do whatever they want. So how can we prevent session hijacking?

The first step is to only set session ids in cookies, instead of in URL rewrites. Also, we should set the httponly cookie property to true. This restricts client side scripts that want access to the session id. Using these techniques, cookies cannot be accessed by XSS and it won't be as easy as we showed to get a session id from a cookie manager.

The second step is to add a token to every request. Similar to the way we dealt with repeat forms in previous sections, we add a hidden field that contains a token. When a request is sent to the server, we can verify this token to prove that the request is unique.

```go
h := md5.New()
salt:="astaxie%^7&8888"
io.WriteString(h,salt+time.Now().String())
token:=fmt.Sprintf("%x",h.Sum(nil))
if r.Form["token"]!=token{
    // ask to log in
}
session.Set("token",token)

```


**Session id timeout**

Another solution is to add a create time for every session, and to replace expired session ids with new ones. This can prevent session hijacking under certain circumstances.

createtime := session.Get("createtime")
if createtime == nil {
    session.Set("createtime", time.Now().Unix())
} else if (createtime.(int64) + 60) < (time.Now().Unix()) {
    sess.Destroy(c)
    session = sess.Start(c)
}

We set a value to save the create time and check if it's expired (I set 60 seconds here). This step can often thwart session hijacking attempts.

Combine the two solutions above and you will be able to prevent most session hijacking attempts from succeeding. On the one hand, session ids that are frequently reset will result in an attacker always getting expired and useless session ids; on the other hand, by setting the httponly property on cookies and ensuring that session ids can only be passed via cookies, all URL based attacks are mitigated.
