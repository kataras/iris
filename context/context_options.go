package context

import (
	"maps"
	"slices"
	"strings"
	"sync"

	"github.com/kataras/iris/v14/context/encoding"
	"github.com/kataras/iris/v14/core/pkg/netutil"
)

type (
	// ContextOptions contains the request and response options for the current context.
	ContextOptions struct {
		//
		// Request options.
		//

		// URLParamSeparator sets the separator used to split the multi-value entries of
		// Context.Query, Context.Form and Context.PostForm (e.g. Query().GetSlice).
		// If not set, parameters with comma-separated values are treated as a single value.
		// Defaults to Comma (",").
		URLParamSeparator string `ini:"url_param_separator" json:"url_param_separator,omitempty" yaml:"URLParamSeparator" toml:"URLParamSeparator"`
		// FireEmptyFormError, when true, causes `context.ReadForm/ReadQuery/ReadBody` to return `iris.ErrEmptyForm` on empty form data.
		// Defaults to false.
		FireEmptyFormError bool `ini:"fire_empty_form_error" json:"fire_empty_form_error,omitzero" yaml:"FireEmptyFormError" toml:"FireEmptyFormError"`
		// TimeFormat specifies the time format for datetime parsing.
		// Defaults to "Mon, 02 Jan 2006 15:04:05 GMT". Used in Context.CheckIfModifiedSince.
		TimeFormat string `ini:"time_format" json:"time_format,omitempty" yaml:"TimeFormat" toml:"TimeFormat"`
		// PostMaxMemory defines the maximum size of post data a client can send, separate from the overall request body limit.
		// Defaults to 32MB (32 << 20).
		PostMaxMemory int64 `ini:"post_max_memory" json:"post_max_memory,omitzero" yaml:"PostMaxMemory" toml:"PostMaxMemory"`
		// BodyMaxMemory limits the total number of bytes read from a request body by
		// Body, ReadJSON, ReadJSONStream and ReadMultipartRelated (and therefore by
		// the readers built on them, such as ReadXML/ReadYAML/ReadBody and
		// ctx.BindJSON). Reading past it fails with
		// ErrRequestEntityTooLarge, which the rest/errors package maps to 413.
		//
		// It is separate from PostMaxMemory, which only bounds multipart form parsing.
		// A value of 0 (the default) means no limit, preserving prior behavior; set it
		// to opt into protection against large-body and decompression-bomb denial of
		// service. For per-route control use the bodylimit middleware instead.
		BodyMaxMemory int64 `ini:"body_max_memory" json:"body_max_memory,omitzero" yaml:"BodyMaxMemory" toml:"BodyMaxMemory"`
		// RemoteAddrHeaders specifies the request header names that can be used to determine the client's IP address.
		// By default, no "X-" headers are considered safe. Use behind proxies or with trusted services like Cloudflare.
		// Defaults to empty slice.
		RemoteAddrHeaders []string `ini:"remote_addr_headers" json:"remote_addr_headers,omitempty" yaml:"RemoteAddrHeaders" toml:"RemoteAddrHeaders"`
		// RemoteAddrHeadersForce overrides the default behavior of `Context.RemoteAddr()` to use the first entry
		// from a request header as the client's IP, even if it's within the `RemoteAddrPrivateSubnets`.
		// Defaults to false.
		RemoteAddrHeadersForce bool `ini:"remote_addr_headers_force" json:"remote_addr_headers_force,omitzero" yaml:"RemoteAddrHeadersForce" toml:"RemoteAddrHeadersForce"`
		// RemoteAddrPrivateSubnets lists the IP ranges considered as private subnets.
		// These ranges are checked against IPs obtained from `RemoteAddrHeaders` or `Context.Request.RemoteAddr`.
		// Defaults to specified ranges covering common private subnets.
		RemoteAddrPrivateSubnets []netutil.IPRange `ini:"remote_addr_private_subnets" json:"remote_addr_private_subnets,omitempty" yaml:"RemoteAddrPrivateSubnets" toml:"RemoteAddrPrivateSubnets"`
		// SSLProxyHeaders specifies headers that indicate a valid HTTPS request.
		// Refer to `Context.IsSSL()` for usage.
		// Defaults to empty map.
		SSLProxyHeaders map[string]string `ini:"ssl_proxy_headers" json:"ssl_proxy_headers,omitempty" yaml:"SSLProxyHeaders" toml:"SSLProxyHeaders"`
		// HostProxyHeaders lists headers that may contain a proxied hostname value.
		// Refer to `Context.Host()` for usage.
		// Defaults to empty map.
		HostProxyHeaders map[string]bool `ini:"host_proxy_headers" json:"host_proxy_headers,omitempty" yaml:"HostProxyHeaders" toml:"HostProxyHeaders"`
		// Validator is the request validator for the current context.
		// It's been called after the request body was decoded to a given struct value on
		// ReadValues, ReadHeader, UnmarshalBody, ReadJSON, ReadMsgPack, ReadForm, ReadParams, ReadURL and ReadQuery.
		// Defaults to nil.
		RequestValidator Validator `ini:"-" json:"-" yaml:"-" toml:"-"`
		//
		// Response options.
		//

		// Charset sets the character encoding for rendering, used in templates and other responses.
		// Defaults to "utf-8".
		Charset string `ini:"charset" json:"charset,omitempty" yaml:"Charset" toml:"Charset"`
		// ErrorHandler is the handler for errors that occur during writes through
		// Render, RenderComponent, JSON, JSONP, XML, Markdown, YAML, Protobuf and MsgPack methods.
		ErrorHandler ErrorHandler `ini:"-" json:"-" yaml:"-" toml:"-"`
		// DisableErrorHandler, when true, disables the error handler completely.
		DisableErrorHandler bool `ini:"disable_error_handler" json:"disable_error_handler,omitzero" yaml:"DisableErrorHandler" toml:"DisableErrorHandler"`

		// Objects holds other object options given by the end-developer.
		// The key is the name of the object and the value is the pointer to the object.
		Objects map[string]any `ini:"-" json:"-" yaml:"-" toml:"-"`
	}
)

const (
	// DefaultTimeFormat is the default time format.
	DefaultTimeFormat = "Mon, 02 Jan 2006 15:04:05 GMT"
)

// DefaultContextOptions returns the default values of ContextOptions instance.
func DefaultContextOptions() ContextOptions {
	return ContextOptions{ // new address each time, or use the reflex.UnsafeClone.
		// Request.
		URLParamSeparator: encoding.DefaultQueryEntrySep,
		TimeFormat:        DefaultTimeFormat,
		PostMaxMemory:     32 << 20, // 32MB.
		RemoteAddrPrivateSubnets: []netutil.IPRange{
			{
				Start: "10.0.0.0",
				End:   "10.255.255.255",
			},
			{
				Start: "100.64.0.0",
				End:   "100.127.255.255",
			},
			{
				Start: "172.16.0.0",
				End:   "172.31.255.255",
			},
			{
				Start: "192.0.0.0",
				End:   "192.0.0.255",
			},
			{
				Start: "192.168.0.0",
				End:   "192.168.255.255",
			},
			{
				Start: "198.18.0.0",
				End:   "198.19.255.255",
			},
		},
		// Response.
		Charset:      DefaultCharset,
		ErrorHandler: DefaultErrorHandler,
	}
}

// FillZero fills every unset (zero) field of o from "from".
// A non-nil empty slice or map is an explicit "none" and is kept as-is;
// boolean fields default to false and are never touched. A scalar whose
// zero value is meaningful cannot be distinguished from unset here; force
// a zero value per request through a middleware instead:
//
//	app.Use(func(ctx *Context) { ctx.Options().URLParamSeparator = ""; ctx.Next() })
func (o *ContextOptions) FillZero(from ContextOptions) {
	if o.URLParamSeparator == "" {
		o.URLParamSeparator = from.URLParamSeparator
	}
	if o.TimeFormat == "" {
		o.TimeFormat = from.TimeFormat
	}
	if o.PostMaxMemory == 0 {
		o.PostMaxMemory = from.PostMaxMemory
	}
	if o.BodyMaxMemory == 0 {
		o.BodyMaxMemory = from.BodyMaxMemory
	}
	if o.RemoteAddrHeaders == nil {
		o.RemoteAddrHeaders = from.RemoteAddrHeaders
	}
	if o.RemoteAddrPrivateSubnets == nil {
		o.RemoteAddrPrivateSubnets = from.RemoteAddrPrivateSubnets
	}
	if o.SSLProxyHeaders == nil {
		o.SSLProxyHeaders = from.SSLProxyHeaders
	}
	if o.HostProxyHeaders == nil {
		o.HostProxyHeaders = from.HostProxyHeaders
	}
	if o.RequestValidator == nil {
		o.RequestValidator = from.RequestValidator
	}
	if o.Charset == "" {
		o.Charset = from.Charset
	}
	if o.ErrorHandler == nil {
		o.ErrorHandler = from.ErrorHandler
	}
	if o.Objects == nil {
		o.Objects = from.Objects
	}
}

// Equal reports whether the ContextOptions and the other ContextOptions are equal.
func (o *ContextOptions) Equal(other *ContextOptions) bool {
	if o == other { // simple pointer check first.
		return true
	}

	return o.URLParamSeparator == other.URLParamSeparator &&
		o.FireEmptyFormError == other.FireEmptyFormError &&
		o.TimeFormat == other.TimeFormat &&
		o.PostMaxMemory == other.PostMaxMemory &&
		o.BodyMaxMemory == other.BodyMaxMemory &&
		slices.Equal(o.RemoteAddrHeaders, other.RemoteAddrHeaders) &&
		o.RemoteAddrHeadersForce == other.RemoteAddrHeadersForce &&
		slices.Equal(o.RemoteAddrPrivateSubnets, other.RemoteAddrPrivateSubnets) &&
		maps.Equal(o.SSLProxyHeaders, other.SSLProxyHeaders) &&
		maps.Equal(o.HostProxyHeaders, other.HostProxyHeaders) &&
		o.RequestValidator == other.RequestValidator &&
		o.Charset == other.Charset &&
		HandlerName(o.ErrorHandler) == HandlerName(other.ErrorHandler) &&
		o.DisableErrorHandler == other.DisableErrorHandler &&
		maps.Equal(o.Objects, other.Objects)
}

// CopyTo writes a deep copy of the ContextOptions into "to", replacing its contents.
// See Clone for a copy that is returned instead.
func (o *ContextOptions) CopyTo(to *ContextOptions) {
	newOptions := &ContextOptions{
		URLParamSeparator:        o.URLParamSeparator,
		FireEmptyFormError:       o.FireEmptyFormError,
		TimeFormat:               o.TimeFormat,
		PostMaxMemory:            o.PostMaxMemory,
		BodyMaxMemory:            o.BodyMaxMemory,
		RemoteAddrHeaders:        slices.Clone(o.RemoteAddrHeaders),
		RemoteAddrHeadersForce:   o.RemoteAddrHeadersForce,
		RemoteAddrPrivateSubnets: slices.Clone(o.RemoteAddrPrivateSubnets),
		SSLProxyHeaders:          maps.Clone(o.SSLProxyHeaders),
		HostProxyHeaders:         maps.Clone(o.HostProxyHeaders),
		RequestValidator:         o.RequestValidator,
		Charset:                  o.Charset,
		ErrorHandler:             o.ErrorHandler,
		DisableErrorHandler:      o.DisableErrorHandler,
		Objects:                  maps.Clone(o.Objects),
	}
	*to = *newOptions
}

// CopyFrom copies the values from the "from" ContextOptions to the current ContextOptions.
func (o *ContextOptions) CopyFrom(from *ContextOptions) {
	from.CopyTo(o)
}

// Clone returns a deep copy of the ContextOptions.
func (o *ContextOptions) Clone() *ContextOptions {
	if o == nil {
		return nil
	}

	var dest ContextOptions
	o.CopyTo(&dest)
	return &dest
}

// OptionsReadOnly returns the context (request and response) options, read-only,
// the caller should not modify the contents because it may be alter the original instance of default options,
// Use the `Options` method for read and write access instead.
// This method is mostly used internally by the Framework.
func (ctx *Context) OptionsReadOnly() *ContextOptions { // for internal use.
	return acquireOptions(ctx, false, true)
}

// Options returns the context (request and response) options, can be edited.
func (ctx *Context) Options() *ContextOptions {
	return acquireOptions(ctx, true, false)
}

//
// Request Options.
//

// Validate validates the given struct value using the request validator for the current context.
func (opts *ContextOptions) Validate(ptr any) error {
	if opts == nil || opts.RequestValidator == nil {
		return nil
	}

	// if err != nil {
	// 	if !strings.HasPrefix(err.Error(), "validator: ") {
	// 		return err
	// 	}
	// }

	return opts.RequestValidator.Struct(ptr)
}

// HandleError handles the error for the current context.
func (opts *ContextOptions) HandleError(ctx *Context, err error) {
	if err == nil {
		return
	}

	errHandler := DefaultErrorHandler
	if opts != nil {
		if opts.DisableErrorHandler {
			return
		}

		if opts.ErrorHandler != nil {
			errHandler = opts.ErrorHandler
		}
	}

	if errHandler != nil {
		// This check is required, because DefaultErrorHandler can be set to nil globally in order to disable the error handling,
		// and let the caller decide how to handle that error. Other option to disable rendering the error: DisableErrorHandler field.
		//
		// This can be modified globally, e.g. when importing the errors package, the framework will automatically bind this to the errors.HandleError.
		errHandler(ctx, err)
	}
}

// ContextOptionObject is the interface for the context options's holded objects.
// type ContextOptionObject interface {
// 	ContextKey() string
// }
// // Set sets a new object to the context options's holded objects.
// func (opts *ContextOptions) Set(obj ContextOptionObject) {
// 	opts.set(obj.ContextKey(), obj)
// }

// Set sets a new object to the context options's holded objects.
func (opts *ContextOptions) Set(objectKey string, objectValue any) {
	if opts.Objects == nil {
		opts.Objects = make(map[string]any)
	}

	// opts.Objects[objectKey] = uintptr(unsafe.Pointer(&objectValue))
	opts.Objects[objectKey] = objectValue
}

// Get returns the object from the context options's holded objects based on the given key.
func (opts *ContextOptions) Get(objectKey string) any {
	if opts.Objects == nil {
		return nil
	}

	return opts.Objects[objectKey]
}

// MimeToContentType returns the media type and charset as content-type based on the given mime type,
// helps with reducing time to resolve a simple content type.
func (opts *ContextOptions) MimeToContentType(mime string) string {
	if opts == nil || (opts.Charset == "" || opts.Charset == DefaultCharset) {
		return defaultMimeTypes.getContentType(mime) // one-map fast access.
	}

	// search for the mime type.
	if mimes, ok := defaultCharsetMimeTypes[opts.Charset]; ok {
		return mimes.getContentType(mime)
	}

	return mime
}

var (
	defaultCharsetMimeTypes map[string]*mimeTypes
	defaultMimeTypes        = makeMimeTypes(DefaultCharset)
)

func init() {
	commonCharsets := [...]string{
		"utf-8",      // It's the default one and specified on defaultMimeTypes for faster access too.
		"iso-8859-1", // latin-1
		"windows-1252",
		"utf-16",
		"utf-32",
		"iso-8859-2",  // latin-2
		"iso-8859-5",  // cyrillic
		"iso-8859-15", // latin-9
	}
	defaultCharsetMimeTypes = make(map[string]*mimeTypes, len(commonCharsets))
	// defaultCharsetMimeTypes[DefaultCharset] = defaultMimeTypes // add the default here just in case.
	for _, charset := range commonCharsets {
		defaultCharsetMimeTypes[charset] = makeMimeTypes(charset)
		// build the upper-cases too.
		defaultCharsetMimeTypes[strings.ToUpper(charset)] = makeMimeTypes(charset)
	}
}

type mimeTypes struct {
	charset string
	mimes   map[string]string
}

func (m *mimeTypes) getContentType(mime string) string {
	if contentType, ok := m.mimes[mime]; ok {
		return contentType
	}

	return mime
}

func makeMimeTypes(charset string) *mimeTypes { // key = mime type, value = content-type.
	mimes := map[string]string{
		"text/html":       "text/html; charset=" + charset,
		"text/css":        "text/css; charset=" + charset,
		"text/plain":      "text/plain; charset=" + charset,
		"text/xml":        "text/xml; charset=" + charset,
		"text/javascript": "text/javascript; charset=" + charset,
		// The "application/json" media type does not require a charset parameter when sending a response from the server to the client.
		// According to RFC 8259, JSON text exchanged between systems that are not part of a closed ecosystem must be encoded using UTF-8.
		// Since UTF-8 is the only encoding standard for JSON, specifying "charset=utf-8" is redundant.
		// Therefore, it is sufficient to use "application/json" without specifying the charset in the Content-Type header of the response.
		"application/xml":                   "application/xml; charset=" + charset,
		"application/problem+xml":           "application/problem+xml; charset=" + charset,
		"application/x-yaml":                "application/x-yaml; charset=" + charset,
		"application/xhtml+xml":             "application/xhtml+xml; charset=" + charset,
		"application/atom+xml":              "application/atom+xml; charset=" + charset,
		"application/rss+xml":               "application/rss+xml; charset=" + charset,
		"application/soap+xml":              "application/soap+xml; charset=" + charset,
		"application/svg+xml":               "application/svg+xml; charset=" + charset,
		"application/x-www-form-urlencoded": "application/x-www-form-urlencoded; charset=" + charset,
	}

	return &mimeTypes{charset: charset, mimes: mimes}
}

var optionsPool = &sync.Pool{
	New: func() any {
		return new(ContextOptions)
	},
}

func acquireOptions(ctx *Context, createIfMissing bool /* if true then returnDefaults no affect */, returnDefaults bool) *ContextOptions {
	// if v := ctx.values.Get(optionsKey); v != nil {
	// 	return v.(*ContextOptions)
	// }
	if v := ctx.options; v != nil {
		return v
	}

	if createIfMissing {
		opts := optionsPool.Get().(*ContextOptions)
		getDefaultOptions(ctx).CopyTo(opts)
		// reflex.UnsafeCopy(ptr, unsafe.Pointer(o.defaults), o.defaultsSizeOfValue)
		// ctx.values.Set(optionsKey, opts) // return this, otherwise defaults.
		ctx.options = opts
		return opts
	}

	if returnDefaults {
		return getDefaultOptions(ctx)
	}

	return nil
}

func getDefaultOptions(ctx *Context) *ContextOptions {
	if ctx.currentRoute != nil {
		return ctx.currentRoute.Router().ContextOptions()
	}
	return ctx.app.ContextOptions() // sometimes the current route is nil, e.g. on pre-routing middleware and on canHandleSubdomain on ServeMux. This indeed returns the root Router's ones.
}

//	func releaseOptions(ctx *Context) {
//		if v := ctx.values.Pop(optionsKey); v != nil { // careful here.
//			optionsPool.Put(v)
//		}
//	}
func releaseOptions(v *ContextOptions) {
	optionsPool.Put(v)
}
