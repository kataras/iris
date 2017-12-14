# Hero Template Example

This folder contains the iris version of the original hero's example: https://github.com/shiyanhui/hero/tree/master/examples/app.

Iris is 100% compatible with `net/http` so you don't have to change anything else
except the handler input from the original example.

The only inline handler's changes were:

From:

```go
if _, err := w.Write(buffer.Bytes()); err != nil {
// and
template.UserListToWriter(userList, w)
```
To: 
```go
if _, err := ctx.Write(buffer.Bytes()); err != nil {
// and
template.UserListToWriter(userList, ctx)
```

So easy.

Read more at: https://github.com/shiyanhui/hero