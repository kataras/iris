# Domain Models

There should be the domain/business-level models.

Example:

```go
import "github.com/kataras/iris/_examples/mvc/overview/datamodels"

type Movie struct {
    datamodels.Movie
}

func (m Movie) Validate() (Movie, error) {
    /* do some checks and return an error if that Movie is not valid */
}
```

However, we will use the "datamodels" as the only one models package because
Movie structure we don't need any extra functionality or validation inside it.