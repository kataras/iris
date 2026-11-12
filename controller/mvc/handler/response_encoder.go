package handler

import (
	"net/http"
	"unsafe"
	"uuid"

	"github.com/kataras/iris/v14/context"
	"github.com/kataras/iris/v14/rest/errors"

	"google.golang.org/protobuf/proto"
)

// ResponseHandler returns a context.Handler that sends a response to the client.
func ResponseHandler[Response ResponseType](response Response, options ...Option) context.Handler {
	opts := makeOptions(options)
	encoder := getResponseEncoder[Response]()
	return func(ctx *context.Context) {
		if !opts.serveHTTP(ctx) {
			return
		}

		err := encoder(ctx, response)
		if errors.HandleError(ctx, err) {
			return
		}
	}
}

// Renderer is the interface that wraps the Render method. It is used to render a response to the client.
// If a Response value implements it, this will be called instead of the default JSON encoding.
type Renderer = context.Renderer

// At REST router & controller we know upfront the response type, so
// for even better performance with REST router & controller casting to find type is not necessary.
//
// CONSTRAINT on the protobuf fast path below:
//
//	Response must be the INTERFACE type (proto.Message),
//	not a concrete pointer such as *pb.Foo.
//
// That branch builds the interface with an unsafe reinterpretation of
// &responseValue. That is only correct when Response is itself an interface, so
// the variable really is a two-word {itab, data} header. Instantiated with a
// concrete `*pb.Foo`, the variable is a single word and the read takes 16 bytes
// from an 8-byte location, treating adjacent memory as an itab.
//
// Note that a typed nil satisfies the interface, so `any((*pb.Foo)(nil))` DOES
// match `.(proto.Message)` in the type switch - the mismatch is not caught here.
//
// The `bool` and `uuid.UUID` casts are a different case and are sound: the type
// switch matched the exact concrete type, so the layouts agree by construction.
// Do not "fix" them into interface assertions.

// Compile-time proof that uuid.UUID is a plain 16-byte array, which the
// unsafe reinterpretation in the uuid.UUID encoder branch below relies on.
var _ [16]byte = uuid.UUID{}

func getResponseEncoder[Response ResponseType]() func(*context.Context, Response) error { // and finally, with this trick we reduce the allocs/op by ~20.
	var resp Response

	switch v := any(resp).(type) {
	case bool:
		// A boolean response is an update or delete outcome, rendered with an
		// empty body: true is a 204 No Content (something changed), false a
		// 304 Not Modified (the call succeeded, nothing changed).
		return func(ctx *context.Context, responseValue Response) error {
			booleanValue := *(*bool)(unsafe.Pointer(&responseValue))
			if booleanValue {
				ctx.StatusCode(http.StatusNoContent)
			} else {
				ctx.StatusCode(http.StatusNotModified)
			}

			return nil
		}
	case string, int, int64, uint64:
		type idPayload struct {
			ID any `json:"id"`
		}
		return func(ctx *context.Context, responseValue Response) error {
			return ctx.JSON(idPayload{ID: responseValue})
		}
	case uuid.UUID:
		return func(ctx *context.Context, responseValue Response) error {
			uuidValue := *(*uuid.UUID)(unsafe.Pointer(&responseValue))
			return ctx.JSON(context.IDPayload[string]{ID: uuidValue.String()})
		}
	case context.Component: // support a Component as a response value.
		return func(ctx *context.Context, responseValue Response) error {
			// renderer := *(*context.Component)(unsafe.Pointer(&responseValue))
			renderer := any(responseValue).(context.Component)
			return renderer.Render(ctx.Request().Context(), ctx.ResponseWriter())
		}
	case context.ResponseRenderer:
		return func(ctx *context.Context, responseValue Response) error {
			// renderer := *(*context.ResponseRenderer)(unsafe.Pointer(&responseValue))
			renderer := any(responseValue).(context.ResponseRenderer)
			return ctx.RenderResponse(ctx.GetStatusCode(), renderer)
		}
	case Renderer:
		// _, isView := any(resp).(*ViewBuilder)
		return func(ctx *context.Context, responseValue Response) error {
			// var renderer Renderer
			// if isView { // specific case for ViewBuilder, in order to catch it without runtime any cast.
			// renderer = (*ViewBuilder)(unsafe.Pointer(&responseValue))
			// Don't ^ this may bind the statusCode internal field to the pointer address of responseValue instead of casting it.
			// Other idea:
			// type emptyInterface struct {
			//	typ unsafe.Pointer
			//	ptr unsafe.Pointer
			// }
			// iface := (*emptyInterface)(unsafe.Pointer(&v)
			// } else {
			renderer := any(responseValue).(Renderer)
			return renderer.Render(ctx)
		}
	default:
		// TODO:
		// Here maybe use some negoation to find the best way to render the response.
		// mediaType, charset, encoding, _ := ctx.Negotiation().Build()
		// if mediaType == "" {
		// 	mediaType = ctx.MediaType() // try to get the media type by the request's content type itself.
		// }
		// if charset == "" {
		// 	charset = ctx.Charset()
		// }
		// if encoding != "" {
		// 	ctx.CompressWriter(true)
		// }

		if _, isProto := v.(proto.Message); isProto {
			return func(ctx *context.Context, responseValue Response) error {
				return ctx.Protobuf(*(*proto.Message)(unsafe.Pointer(&responseValue)))
			}
		}

		// Types implementing encoding/json/v2's MarshalerTo already stream
		// through ctx.JSON's encoder without an intermediate []byte — the
		// old easyjson fast path (and its unsafe interface reinterpretation)
		// has no reason to exist anymore.
		return func(ctx *context.Context, responseValue Response) error {
			return ctx.JSON(responseValue)
		}
	}
}
