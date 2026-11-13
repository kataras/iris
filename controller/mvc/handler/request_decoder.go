package handler

import (
	"fmt"
	"reflect"
	"slices"
	"unsafe"
	"uuid"

	"github.com/kataras/iris/v14/context"
	"github.com/kataras/iris/v14/context/encoding"
	"github.com/kataras/iris/v14/core/pkg/reflex"
	"github.com/kataras/iris/v14/rest/errors"

	"google.golang.org/protobuf/proto"
)

type (
	DirectRequestDecoder interface {
		DecodeRequest(*context.Context) error
	}

	// RequestDecoder is an interface which contains a function which takes a context and binds a generic type Request.
	// It is used to call a service function with a generic type Request.
	// It is used for functions which do not bind a request payload.
	// It is used for XHandler functions.
	// Developers can design their own RequestDecoder and use them with the XHandler functions.
	//
	// See the NewPathParamDecoder, NewQueryDecoder and NewValueDecoder package-level helpers too.
	RequestDecoder[RequestOrParam any] interface {
		DecodeRequest(ctx *context.Context, req *RequestOrParam) error
	}

	// RequestDecoderFunc is a function which takes a context and returns a generic type Request.
	// Implements the RequestDecoder interface.
	RequestDecoderFunc[RequestOrParam any] func(ctx *context.Context, req *RequestOrParam) error
)

func extractDirectRequestDecoder[RequestOrParam any]() RequestDecoder[RequestOrParam] {
	if _, implementsRequestHandler := any(new(RequestOrParam)).(DirectRequestDecoder); implementsRequestHandler {
		// type emptyInterface struct {
		// 	typ unsafe.Pointer
		// 	ptr unsafe.Pointer
		// }
		return RequestDecoderFunc[RequestOrParam](func(ctx *context.Context, req *RequestOrParam) error {
			// handler := *(*RequestDecoder)(unsafe.Pointer(&payload))
			// err := handler.DecodeRequest(ctx)
			// These don't work, so dereference the pointer to the handler using the standard way, we already checked if this request type implements the DirectRequestDecoder at built-time.
			// iface := (*emptyInterface)(unsafe.Pointer(&req))
			// decoder := *(*DirectRequestDecoder)(iface.typ)
			decoder := any(req).(DirectRequestDecoder)
			return decoder.DecodeRequest(ctx)
		})
	}

	return nil
}

// Compile-time check for the RequestDecoder interface.
var _ RequestDecoder[any] = RequestDecoderFunc[any](nil)

// DecodeRequest calls the RequestDecoderFunc.
func (fn RequestDecoderFunc[Request]) DecodeRequest(ctx *context.Context, req *Request) error {
	return fn(ctx, req)
}

// PathParamDecoder is a RequestDecoder which reads a path parameter from the context and returns it as a generic type Request.
type PathParamDecoder[Param ParamType] struct {
	paramName string
}

var _ RequestDecoder[any] = (*PathParamDecoder[any])(nil)

// NewPathParamDecoder returns a RequestDecoder which reads a path parameter from the context and returns it as a generic type Request.
// If paramName is empty then it reads the last path parameter from the context.
// It is used for XHandler functions.
func NewPathParamDecoder[Param ParamType](paramName string) *PathParamDecoder[Param] {
	return &PathParamDecoder[Param]{paramName: paramName}
}

// DecodeRequest reads a path parameter from the context and returns it as a generic type Request.
// It implements the RequestDecoder interface.
func (p *PathParamDecoder[Param]) DecodeRequest(ctx *context.Context, req *Param) error {
	return decodePathParam(ctx, req, p.paramName)
}

// String returns the name of the path parameter.
func (p *PathParamDecoder[Param]) String() string {
	return p.paramName
}

// PathParam reads a path parameter from the context and returns it as a generic type Param.
// If paramName is empty then it reads the last path parameter from the context.
func decodePathParam[Param ParamType](ctx *context.Context, param *Param, paramName string) error {
	var paramValue any

	if paramName != "" {
		// Honour the requested name. Falling back to the last entry regardless of
		// the name silently returns the wrong parameter on a nested route such as
		// "/stores/{store_id}/books/{book_id}".
		value, ok := ctx.Params().Lookup(paramName)
		if !ok {
			return errors.InvalidArgument
		}
		paramValue = value
	} else {
		paramEntry, ok := ctx.Params().GetLastEntry()
		if !ok {
			return errors.InvalidArgument
		}
		paramValue = paramEntry.Value
	}

	value, ok := ConvertPathParamValue[Param](paramValue)
	if !ok {
		return errors.InvalidArgument
	}

	*param = value
	return nil
}

// NewValueDecoder returns a RequestDecoder which returns a generic type Request.
// It is used for XHandler functions.
func NewValueDecoder[RequestOrParam any](value RequestOrParam) RequestDecoder[RequestOrParam] {
	return RequestDecoderFunc[RequestOrParam](func(ctx *context.Context, t *RequestOrParam) error {
		*t = value
		return nil
	})
}

// NewZeroValueDecoder returns a RequestDecoder which returns a zero value of a generic type Request.
func NewZeroValueDecoder[RequestOrParam any]() RequestDecoder[RequestOrParam] {
	return RequestDecoderFunc[RequestOrParam](func(ctx *context.Context, t *RequestOrParam) error {
		return nil
	})
}

type userDecoder struct{}

var _ RequestDecoder[context.User] = (*userDecoder)(nil)

// NewUserDecoder returns a RequestDecoder which reads the user from the context and returns it as a generic type Request.
func NewUserDecoder() RequestDecoder[context.User] {
	return &userDecoder{}
}

func (p *userDecoder) DecodeRequest(ctx *context.Context, u *context.User) error {
	user := ctx.User()
	userID, _ := user.GetID()
	if userID == "" {
		return errors.Unauthenticated
	}

	*u = user
	return nil
}

// pathParamsDecoder is a RequestDecoder which reads one or more path parameters from the context and returns it as a generic type Request.
type pathParamsDecoder[RequestOrParam any] struct{}

var _ RequestDecoder[any] = (*pathParamsDecoder[any])(nil)

// NewPathParamsDecoder returns a RequestDecoder which reads one or more path parameters from
// the context and returns it as a generic type Request.
func NewPathParamsDecoder[RequestOrParam any]() RequestDecoder[RequestOrParam] {
	return &pathParamsDecoder[RequestOrParam]{}
}

func (p *pathParamsDecoder[RequestOrParam]) DecodeRequest(ctx *context.Context, req *RequestOrParam) error {
	return decodePathParams(ctx, req)
}

func decodePathParams[Params any](ctx *context.Context, t *Params) error {
	return ctx.ReadParams(t)
}

// queryDecoder is a RequestDecoder which reads a URL query from the context and returns it as a generic type Request.
// It is used for XHandler functions.
type queryDecoder[RequestOrParam any] struct{}

var _ RequestDecoder[any] = (*queryDecoder[any])(nil)

// NewQueryDecoder returns a RequestDecoder which reads a URL query from the context and returns it as a generic type Request.
func NewQueryDecoder[RequestOrParam any]() RequestDecoder[RequestOrParam] {
	return &queryDecoder[RequestOrParam]{}
}

func (p *queryDecoder[Request]) DecodeRequest(ctx *context.Context, q *Request) error {
	return ctx.ReadQuery(q)
}

// urlDecoder is a RequestDecoder which reads a URL value from the context and returns it as a generic type Request.
// It binds the struct fields by the following field tags: "param" (dynamic path parameter) and "url" (url query parameter).
type urlDecoder[RequestOrParam any] struct{}

var _ RequestDecoder[any] = (*urlDecoder[any])(nil)

// NewURLDecoder returns a RequestDecoder which reads URL values from the context and returns it as a generic type Request.
func NewURLDecoder[RequestOrParam any]() RequestDecoder[RequestOrParam] {
	return &urlDecoder[RequestOrParam]{}
}

func (p *urlDecoder[RequestOrParam]) DecodeRequest(ctx *context.Context, payload *RequestOrParam) error {
	return ctx.ReadURL(payload)
}

// headerDecoder is a RequestDecoder which reads headers values from the context and returns it as a generic type Request.
type headerDecoder[RequestOrParam any] struct{}

var _ RequestDecoder[any] = (*headerDecoder[any])(nil)

// NewHeaderDecoder returns a RequestDecoder which reads headers values from the context and returns it as a generic type Request.
func NewHeaderDecoder[RequestOrParam any]() RequestDecoder[RequestOrParam] {
	return &headerDecoder[RequestOrParam]{}
}

func (p *headerDecoder[Request]) DecodeRequest(ctx *context.Context, payload *Request) error {
	return ctx.ReadHeader(payload)
}

// formDecoder is a RequestDecoder which reads the form data from the context and returns it as a generic type Request.
type formDecoder[RequestOrParam any] struct{}

var _ RequestDecoder[any] = (*formDecoder[any])(nil)

// NewFormDecoder returns a RequestDecoder which reads the form data from the context and returns it as a generic type Request.
func NewFormDecoder[RequestOrParam any]() RequestDecoder[RequestOrParam] {
	return &formDecoder[RequestOrParam]{}
}

func (p *formDecoder[RequestOrParam]) DecodeRequest(ctx *context.Context, payload *RequestOrParam) error {
	return ctx.ReadForm(payload)
}

type jsonDecoder[RequestOrParam any] struct{}

var _ RequestDecoder[any] = (*jsonDecoder[any])(nil)

// NewJSONDecoder returns a RequestDecoder which reads a JSON payload from the context and returns it as a generic type Request.
func NewJSONDecoder[RequestOrParam any]() RequestDecoder[RequestOrParam] {
	return &jsonDecoder[RequestOrParam]{}
}

func (p *jsonDecoder[RequestOrParam]) DecodeRequest(ctx *context.Context, payload *RequestOrParam) error {
	return ctx.ReadJSON(payload)
}

// Transformer returns a RequestDecoder which reads a generic type From from the context,
// transforms it to a generic type To and returns it.
func Transformer[From any, To RequestType](fn func(From) (To, error)) RequestDecoder[To] {
	bodyDecoder := DefaultRequestDecoder[From]()

	return RequestDecoderFunc[To](func(ctx *context.Context, to *To) error {
		var from From
		err := bodyDecoder.DecodeRequest(ctx, &from)
		if err != nil {
			return err
		}

		toValue, err := fn(from)
		if err != nil {
			return err
		}
		*to = toValue
		return nil
	})
}

/*
var requestTypeUser requestTypeReader = "user" // it's a field tag for the current authorizated user.

// UserUnmarshaler is an interface which unmarshals the current user from the context.
// Remember that reading the body from this interface is not allowed,
// because data should be available for the upcoming Request data type itself.
// If you should consume the request's body please make sure you first replace it with a bufio.NewReader
// or similar in order for the request body's data to be available:
// r := ctx.Request()
// reader := bufio.NewReader(r.Body)
// r.Body = ioutil.NopCloser(reader)
// OR just:
// ctx.RecordRequestBody(true).
type UserUnmarshaler interface {
	// UnmarshalUser unmarshals the current user from the context to the given destination.
	UnmarshalUser(ctx *context.Context) error
}

// defaultUserUnmarshaler reads the user from the context and returns it as a generic type T.
// It does not consume the request body, only reads from the context's values.
// It returns errors.Unauthenticated if the user is not found or not of the expected type T.
func defaultUserUnmarshaler[T any](ctx *context.Context, dst *T) error {
	if value := ctx.Values().Get(context.UserContextKey); value != nil { // don't use ctx.User() because it only returns on a compatible context.User interface.
		if user, ok := value.(T); ok {
			*dst = user
			return nil
		}
	}

	return errors.Unauthenticated
}

func getBuiltinUserUnmarshaler[T any](field *T) UserUnmarshaler {
	return *(*UserUnmarshaler)(unsafe.Pointer(field))
}
*/

type (
	requestTypeReader string
)

var (
	requestTypeParam         requestTypeReader = encoding.DefaultPathParamTag
	requestTypeContextValues requestTypeReader = encoding.DefaultContextValueTag
	requestTypeHeader        requestTypeReader = encoding.DefaultHeaderTag
	requestTypeQuery         requestTypeReader = encoding.DefaultQueryTag
	requestTypeForm          requestTypeReader = encoding.DefaultFormTag
	requestTypeXML           requestTypeReader = "xml"
	requestTypeYAML          requestTypeReader = "yaml"
	requestTypeProtobuf      requestTypeReader = "protobuf"
	requestTypeMsgPack       requestTypeReader = "msgpack"
	requestTypeJSON          requestTypeReader = "json"
)

func (r requestTypeReader) IsBody() bool {
	return r == requestTypeForm || r == requestTypeXML || r == requestTypeYAML || r == requestTypeProtobuf || r == requestTypeMsgPack || r == requestTypeJSON
}

func defaultRequestDecoder[RequestOrParam any](readerTypes ...requestTypeReader) RequestDecoder[RequestOrParam] {
	if directRequestDecoder := extractDirectRequestDecoder[RequestOrParam](); directRequestDecoder != nil {
		// It overrides all the below for the maximum performance on custom param or request self-struct handlers/decoders.
		// fmt.Printf("%T implements the decode request interface\n", directRequestDecoder)
		return directRequestDecoder
	}

	var reqValueType RequestOrParam
	switch reqValue := any(reqValueType).(type) {
	case string, int, int64, uint64, uuid.UUID:
		return NewPathParamDecoder[RequestOrParam]("")
	default:
		typ := reflect.TypeOf(reqValueType)
		if typ == nil {
			// reflect.TypeOf on a nil interface value returns nil, and the switch
			// below would nil-deref. That happens when Request is instantiated with
			// an interface type, which this decoder cannot construct.
			panic(fmt.Sprintf("DefaultRequestDecoder: the Request type parameter must be a concrete type, "+
				"got the interface %s", reflect.TypeFor[RequestOrParam]().String()))
		}

		// implementsRequestHandler := typ.Implements(reflect.TypeOf((*RequestDecoder[Request])(nil)).Elem())
		switch typ.Kind() {
		case reflect.Struct:
			fields := reflex.LookupFields(typ, "")
			if len(fields) == 0 {
				return nil
			}

			var (
				hasPathParams, hasContextValues, hasHeaders, hasQuery, hasRequiredQuery, hasForm, hasXML, hasYAML, hasProtobuf, hasMsgPack bool
				pathParamsDecoder                                                                                                          *encoding.EntriesDecoder[RequestOrParam, any]
				contextValuesDecoder                                                                                                       *encoding.EntriesDecoder[RequestOrParam, any]
				queryDecoder                                                                                                               *encoding.KeyValuesDecoder[RequestOrParam]
				headersDecoder                                                                                                             *encoding.KeyValuesDecoder[RequestOrParam]
				formDecoder                                                                                                                *encoding.KeyValuesDecoder[RequestOrParam]
			)

			// See iris-contrib/schema/schema.go for these tags.
			if slices.Contains(readerTypes, requestTypeParam) {
				hasPathParams = reflex.StructFieldsCountByTag(fields, encoding.DefaultPathParamTag) > 0
				if hasPathParams {
					if dec, err := encoding.NewPathParamsDecoder[RequestOrParam](encoding.PathParamsDecoderOptions{}); err == nil {
						pathParamsDecoder = dec
					} // no error? just let ctx.ReadParams(v) as a faillback.
				}
			}

			if slices.Contains(readerTypes, requestTypeContextValues) {
				hasContextValues = reflex.StructFieldsCountByTag(fields, encoding.DefaultContextValueTag) > 0
				if hasContextValues {
					if dec, err := encoding.NewContextValuesDecoder[RequestOrParam](encoding.ContextValuesDecoderOptions{}); err == nil {
						contextValuesDecoder = dec
					} // else no error? just let ctx.ReadValues(v) as a faillback.
				}
			}

			if slices.Contains(readerTypes, requestTypeHeader) {
				hasHeaders = reflex.StructFieldsCountByTag(fields, encoding.DefaultHeaderTag) > 0
				if hasHeaders {
					if dec, err := encoding.NewHeaderDecoder[RequestOrParam](encoding.HeaderDecoderOptions{}); err == nil {
						headersDecoder = dec
					} // else no error? just let ctx.ReadHeader(v) as a faillback.
				}
			}

			if slices.Contains(readerTypes, requestTypeQuery) {
				hasQuery = reflex.StructFieldsCountByTag(fields, encoding.DefaultQueryTag) > 0
				if hasQuery {
					if dec, err := encoding.NewQueryDecoder[RequestOrParam](encoding.QueryDecoderOptions{}); err == nil {
						queryDecoder = dec
						hasRequiredQuery = dec.HasRequired()
					} // else no error? just let ctx.ReadURL(v) as a faillback.
				}
			}

			if slices.Contains(readerTypes, requestTypeForm) {
				hasForm = reflex.StructFieldsCountByTag(fields, encoding.DefaultFormTag) > 0
				if hasForm {
					if dec, err := encoding.NewFormDecoder[RequestOrParam](encoding.FormDecoderOptions{}); err == nil {
						formDecoder = dec
					} // else no error? just let ctx.ReadForm(v) as a faillback.
				}
			}

			if slices.Contains(readerTypes, requestTypeXML) {
				hasXML = true
			}

			if slices.Contains(readerTypes, requestTypeYAML) {
				hasYAML = true
			}

			if slices.Contains(readerTypes, requestTypeProtobuf) {
				hasProtobuf = reflex.StructFieldsCountByTag(fields, "protobuf") > 0
				// check if the request type is actually a proto.Message.
				if _, ok := reqValue.(proto.Message); !ok {
					hasProtobuf = false
				}
			}

			if slices.Contains(readerTypes, requestTypeMsgPack) {
				// tagging is not required for msgpack, just like JSON we just check the content type requested.
				// hasMsgPack = reflex.StructFieldsCountByTag(fields, "msgpack") > 0
				hasMsgPack = true
			}

			/*
				// User param.
				var (
					hasUser                          bool
					userField                        reflect.StructField
					payloadHasBuiltinUserUnmarshaler bool
				)
				if slices.Contains(readerTypes, requestTypeUser) {
					userField, hasUser = reflex.FirstStructFieldByTag(fields, "user")
					if hasUser {
						payloadHasBuiltinUserUnmarshaler = userField.Type.Implements(reflect.TypeOf((*UserUnmarshaler)(nil)).Elem())
						// if _, ok := reqValue.(UserUnmarshaler); ok {
						// 	payloadHasBuiltinUserUnmarshaler = true
						// }
					}
				}
			*/

			return RequestDecoderFunc[RequestOrParam](func(ctx *context.Context, payload *RequestOrParam) error {

				// var nonBody bool // boolean to indicate that the model was readen by non-body values.
				if hasPathParams {
					pathParams := *ctx.Params()
					if len(pathParams) > 0 {
						if pathParamsDecoder != nil {
							if err := pathParamsDecoder.DecodeEntries(pathParams, payload); err != nil {
								return fmt.Errorf("read params: %w", err)
							}
						} else {
							if err := ctx.ReadParams(payload); err != nil {
								return fmt.Errorf("read params: %w", err)
							}
						}
						// These "nonBody = true" can be replaced by the readerTypesShouldReadBody function helper
						// nonBody = true
					}
				}

				if hasContextValues {
					contextValues := ctx.Values()
					if contextValues.Len() > 0 {
						if contextValuesDecoder != nil {
							if err := contextValuesDecoder.DecodeEntries(*ctx.Values(), payload); err != nil {
								return fmt.Errorf("read context values: %w", err)
							}
						}
						if err := ctx.ReadValues(payload); err != nil {
							return fmt.Errorf("read context values: %w", err)
						}
					}

					// nonBody = true
				}

				if hasQuery {
					if hasRequiredQuery && ctx.Request().URL.RawQuery == "" {
						return fmt.Errorf("read query: %w", &encoding.ValidationError{Field: "query", Value: "", Reason: "required"})
					}

					if queryDecoder != nil {
						if err := queryDecoder.DecodeKeyValues(ctx.Request().URL.Query() /* use this instead of entries, as this is used once per life-cycle. */, payload); err != nil {
							return fmt.Errorf("read query: %w", err)
						}
					} else {
						if err := ctx.ReadQuery(payload); err != nil {
							return fmt.Errorf("read query: %w", err)
						}
					}

					// nonBody = true
				}

				if hasHeaders {
					if headersDecoder != nil {
						if err := headersDecoder.DecodeKeyValues(ctx.Request().Header, payload); err != nil {
							return fmt.Errorf("read headers: %w", err)
						}
					} else {
						if err := ctx.ReadHeader(payload); err != nil {
							return fmt.Errorf("read headers: %w", err)
						}
					}

					// nonBody = true
				}

				/* Good idea but I want to completely remove the need of reflection at serve-time.
				if hasUser {
					userFieldValue := reflect.ValueOf(payload).FieldByIndex(userField.Index)
					userFieldActualValue := userFieldValue.Interface()
					if payloadHasBuiltinUserUnmarshaler {
						if err := getBuiltinUserUnmarshaler(&userFieldActualValue).UnmarshalUser(ctx); err != nil {
							return payload, fmt.Errorf("read user: UnmarshalUser: %w", err)
						}
					} else {
						if err := defaultUserUnmarshaler(ctx, &userFieldActualValue); err != nil {
							return payload, fmt.Errorf("read user: %w", err)
						}
					}
					userFieldValue.Set(reflect.ValueOf(userFieldValue))
				}*/

				// If read from path parameters, quety or headers, then return the payload and don't consume the request body,
				// in order for the [Param,Request] to work as expected, unless the Param payload has no specific param, header or/and url tags.
				//
				// No need of hasJSON because we want to encode the response even if no json tag exists.
				askedForParamDecoder := !slices.Contains(readerTypes, requestTypeJSON) // param does not read json.
				enableBodyReader := hasForm || hasXML || hasYAML || hasProtobuf || hasMsgPack || slices.Contains(readerTypes, requestTypeJSON)
				if askedForParamDecoder && !enableBodyReader {
					// The decoders above do not run the request validator, the ctx.Read methods do.
					return ctx.OptionsReadOnly().Validate(payload)
				}

				// I don't think is useful to check this, just let it.
				requestMediaType := ctx.MediaType()
				if hasForm && ctx.IsMediaType(context.ContentFormHeaderValue, context.ContentFormMultipartHeaderValue) {
					if formDecoder != nil {
						if err := formDecoder.DecodeKeyValues(ctx.Form().ToMap(), payload); err != nil {
							return fmt.Errorf("read form: %w", err)
						}

						if err := ctx.OptionsReadOnly().Validate(payload); err != nil {
							return fmt.Errorf("read form: %w", err)
						}
					} else {
						if err := ctx.ReadForm(payload); err != nil {
							return fmt.Errorf("read form: %w", err)
						}
					}
				} else {
					if hasXML && ctx.IsMediaType(context.ContentXMLHeaderValue, context.ContentXMLUnreadableHeaderValue) {
						if err := ctx.ReadXML(payload); err != nil {
							return fmt.Errorf("read xml: %w", err)
						}
					} else if hasYAML && ctx.IsMediaType(context.ContentYAMLHeaderValue, context.ContentYAMLTextHeaderValue) {
						if err := ctx.ReadYAML(payload); err != nil {
							return fmt.Errorf("read yaml: %w", err)
						}
					} else if hasProtobuf {
						// CONSTRAINT: the first two words of *payload must already be a
						// {itab, data} interface header, i.e. the Request type parameter
						// must be an interface (proto.Message), not a struct whose
						// promoted methods happen to satisfy proto.Message.
						//
						// This branch is only reached when typ.Kind() is Struct, so a
						// struct embedding a generated *pb.Foo - which does satisfy
						// proto.Message through promotion - reads the struct's own first
						// 16 bytes as an interface header. Note also that this executes
						// BEFORE the media-type check just below, so it runs on every
						// non-form/xml/yaml request to such a route.
						payloadPtrAsProtoMessage := *(*proto.Message)(unsafe.Pointer(payload))

						if requestMediaType == context.ContentProtobufHeaderValue { // protobuf.
							if err := ctx.ReadProtobuf(payloadPtrAsProtoMessage); err != nil {
								return fmt.Errorf("read protobuf: %w", err)
							}
						} else if requestMediaType == context.ContentJSONHeaderValue { // protobuf with json encoding.
							if err := ctx.ReadProtobufJSON(payloadPtrAsProtoMessage); err != nil {
								return fmt.Errorf("read protobuf (json): %w", err)
							}
						}
					} else if hasMsgPack && ctx.IsMediaType(context.ContentMsgPackHeaderValue, context.ContentMsgPack2HeaderValue) {
						if err := ctx.ReadMsgPack(payload); err != nil {
							return fmt.Errorf("read msgpack: %w", err)
						}
					} else if requestMediaType == context.ContentJSONHeaderValue {
						if err := ctx.ReadJSON(payload); err != nil {
							return fmt.Errorf("read json: %w", err)
						}
					} else { // try everything else...
						if !hasPathParams && !hasQuery && !hasHeaders {
							if err := ctx.ReadBody(payload); err != nil {
								return fmt.Errorf("read body: %w", err)
							}
						} else {
							// Bound from path parameters, the query or headers only.
							return ctx.OptionsReadOnly().Validate(payload)
						}
					}
				}

				return nil
			})
		case reflect.Slice, reflect.Map, reflect.Array:
			// support for slices and map, we don't perform any special action here, just json reading.
			// TODO: can be improved by checking the slice/map element value type and its fields and then decide what to do.
			return RequestDecoderFunc[RequestOrParam](func(ctx *context.Context, payload *RequestOrParam) error {
				if err := ctx.ReadJSON(payload); err != nil {
					return fmt.Errorf("read json: %w", err)
				}

				return nil
			})
		default:
			// return nil
			panic(fmt.Sprintf("DefaultRequestDecoder: unsupported request type: %T. "+
				"The Request type parameter must be a struct, a pointer to a struct, "+
				"or one of the supported path parameter types (string, int, int64, uint64, uuid.UUID)", reqValue))
		}
	}
}

// ExecuteRequestHandlers calls the DecodeRequest method of each handler in the order they are passed.
func ExecuteRequestHandlers[Request RequestType](ctx *context.Context, handlers ...DirectRequestDecoder) error {
	for _, handler := range handlers {
		if err := handler.DecodeRequest(ctx); err != nil {
			return err
		}
	}

	return nil
}

/*
func unsafeIsBufBodyReaden(r *http.Request) bool {
	type bufReader struct {
		buf []byte // r.Body is a *bufio.Reader
	}

	brPointer := (*bufReader)(unsafe.Pointer(r.Body))
	return len(brPointer.buf) == 0
}
*/

// DefaultRequestDecoder returns a RequestDecoder which reads a
// Single path parameter (string, int, int64, uint64 or uuid.UUID type) or
// URL path parameters and/or (param field tag)
// Context values (context field tag)
// Headers (header field tag)
// URL query and/or (url field tag)
// Form data and/or (form field tag)
// JSON payload (json field tag)
// from the context and returns it as a generic type Request.
func DefaultRequestDecoder[Request any]() RequestDecoder[Request] {
	return defaultRequestDecoder[Request](
		requestTypeParam,
		requestTypeContextValues,
		requestTypeHeader,
		requestTypeQuery,
		requestTypeForm,
		requestTypeXML,
		requestTypeYAML,
		requestTypeProtobuf,
		requestTypeMsgPack,
		requestTypeJSON,
	) // enable all readers.
}

// DefaultParamDecoder returns a RequestDecoder which reads a
// Single path parameter (string, int, int64, uint64 or uuid.UUID type) or
// URL path parameters and/or (param field tag)
// Context values (context field tag)
// Headers (header field tag)
// URL query and/or (url field tag)
// from the context and returns it as a generic type Request.
func DefaultParamDecoder[Param any]() RequestDecoder[Param] {
	return defaultRequestDecoder[Param](requestTypeParam,
		requestTypeContextValues,
		requestTypeHeader,
		requestTypeQuery,
	) // enable only path parameters, headers, query and context values readers.
}
