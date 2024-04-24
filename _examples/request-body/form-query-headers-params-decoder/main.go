// package main contains an example on how to register a custom decoder
// for a custom type when using the `ReadQuery/ReadParams/ReadHeaders/ReadForm` methods.
//
// Let's take for example the mongo-driver/primite.ObjectID:
//
// ObjectID is type ObjectID [12]byte.
// You have to register a converter for that custom type.
// ReadJSON works because the ObjectID has a MarshalJSON method
// which the encoding/json pakcage uses as a converter.
// See here: https://godoc.org/go.mongodb.org/mongo-driver/bson/primitive#ObjectID.MarshalJSON.
//
// To register a converter import the github.com/iris-contrib/schema and call
// schema.Query.RegisterConverter(value interface{}, converterFunc Converter).
//
// The Converter is just a type of func(string) reflect.Value.
//
// There is another way, but requires introducing a custom type which will wrap the mongo's ObjectID
// and implements the encoding.TextUnmarshaler e.g.
// func(id *ID) UnmarshalText(text []byte) error){ ...}.
package main

import (
	"reflect"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/iris-contrib/schema"
	"github.com/kataras/iris/v12"
)

type MyType struct {
	ID primitive.ObjectID `url:"id"`
}

func main() {
	// Register on initialization.
	schema.Query.RegisterConverter(primitive.ObjectID{}, func(value string) reflect.Value {
		id, err := primitive.ObjectIDFromHex(value)
		if err != nil {
			return reflect.Value{}
		}
		return reflect.ValueOf(id)
	})

	app := iris.New()

	app.Get("/", func(ctx iris.Context) {
		var t MyType
		err := ctx.ReadQuery(&t)
		if err != nil && !iris.IsErrPath(err) {
			ctx.StopWithError(iris.StatusInternalServerError, err)
			return
		}

		ctx.Writef("MyType.ID: %q", t.ID.Hex())
	})

	// http://localhost:8080?id=507f1f77bcf86cd799439011
	app.Listen(":8080")
}
