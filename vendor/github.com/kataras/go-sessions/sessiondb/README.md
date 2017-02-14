## Repository information

This repository contains the built'n session databases for the [go-sessions](https://github.com/kataras/go-sessions).

## How to Register?



```go
//...
import (
	"github.com/kataras/go-sessions"
	"github.com/kataras/go-sessions/sessiondb/$FOLDER"
)
//...

db := $FOLDER.New($FOLDER.Config{})

//...
manager:= sessions.New(sessions.Config{})
manager.UseDatabase(db)

//...
```

> Note: You can use more than one database to save the session values, but the initial data will come from the first non-empty `Load`, look inside [code](https://github.com/kataras/go-sessions/blob/master/sessiondb/redis/database.go) for more information on how to create your own session database.
