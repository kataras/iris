# gRPC Iris Example

## Generate TLS Keys

```sh
$ openssl genrsa -out server.key 2048
$ openssl req -new -x509 -sha256 -key server.key -out server.crt -days 3650
```

## Install the protoc Go plugin

```sh
$ go get -u github.com/golang/protobuf/protoc-gen-go
```

## Generate proto

```sh
$ protoc -I helloworld/ helloworld/helloworld.proto --go_out=plugins=grpc:helloworld
```
