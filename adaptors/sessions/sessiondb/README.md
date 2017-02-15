## Session databases

Find more databases at [github.com/kataras/go-sessions/sessiondb](https://github.com/kataras/go-sessions/tree/master/sessiondb).

This folder contains only the redis database because the rest (two so far, 'file' and 'leveldb') were created by the Community.
So go [there](https://github.com/kataras/go-sessions/tree/master/sessiondb) and find more about them. `Database` is just an
interface so you're able to `UseDatabase(anyCompatibleDatabase)`. A Database should implement two functions, `Load` and `Update`.

**Database interface**

```go
type Database interface {
	Load(string) map[string]interface{}
	Update(string, map[string]interface{})
}
```

```go
import (
  "...myDatabase"
)
s := New(...)
s.UseDatabase(myDatabase) // <---

app := iris.New()
app.Adapt(s)

app.Listen(":8080")
```
