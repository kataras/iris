module grpcexample

go 1.15

// replace github.com/kataras/iris/v12 => ../../../

require (
	github.com/golang/protobuf v1.4.2
	github.com/kataras/iris/v12 master
	google.golang.org/grpc v1.33.1
	google.golang.org/protobuf v1.25.0
)
