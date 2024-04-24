# Protocol Buffers

The `Context.Protobuf(proto.Message)` is the method which sends protos to the client. It accepts a [proto.Message](https://godoc.org/google.golang.org/protobuf/proto#Message) value.

> Note: Iris is using the newest version of the Go protocol buffers implementation. Read more about it at [The Go Blog: A new Go API for Protocol Buffers](https://blog.golang.org/protobuf-apiv2).


1. Install the protoc-gen-go tool.

```sh
$ go get -u google.golang.org/protobuf/cmd/protoc-gen-go@latest
```

2. Generate proto

```sh
$ protoc -I protos/ protos/hello.proto --go_out=.
```
