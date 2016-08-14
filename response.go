package iris

import (
	"strings"

	"github.com/iris-contrib/errors"
	"github.com/valyala/fasthttp"
)

type (
	// notes for me:
	// edw an kai kalh idea alla den 9a borw na exw ta defaults mesa sto iris
	// kai 9a prepei na to metaferw auto sto context i sto utils kai pragmatika den leei
	// na kanoun import auto gia na kanoun to response engine, ara na prospa9isw kapws aliws mesw context.IContext mono
	// ektos an metaferw ta defaults mesa sto iris
	// alla an to kanw auto 9a prepei na vrw tropo na kanoun configuration ta defaults
	// kai to idio prepei na kanw kai sto template engine html tote...
	// diladi na kanw prwta to render tou real engine kai meta na parw
	// ta contents tou body kai na ta kanw gzip ? na kanw resetbody kai na ta ksanasteilw?
	// alla auto einai argh methodos kai gia ton poutso dn m aresei
	// kai ola auta gia na mh valw ena property parapanw? 9a valw ...
	// 9a einai writer,headers,object,options... anagastika.

	/* notes for me:
	   english 'final' thoughs' results now:

	   the response engine will be registered with its content type
	   for example: iris.UseResponse("application/json", the engine or func here, optionalOptions...)
	   if more than one response engines registered for the same content type
	   then all the content results will be sent to the client
	   there will be available a system something like middleware but for the response engines
	   this will be useful when one response engine's job is  only to add more content to the existing/parent response engine's results .
		 for example when you want to add a standard  json object like a last_fetch { "year":...,"month":...,"day":...} for all json responses.
	   The engine will not have access to context or something like that.
	   The default engines will registered when no other engine with the same content type
	   already registered, like I did with template engines, so if someone wants to make a 'middleware' for the default response engines
	   must register them explicit and after register his/her response engine too, for that reason
	   the default engines will not be located inside iris but inside iris-contrib (like the default,and others, template engine),
	   the reason is to be easier to the user/dev to remember what import path should use when he/she wants to edit something,
	   for templates it's 'iris-contrib/template',  for response engines will be 'iris-contrib/response'.
	   The body content will return as []byte, al/mong with an error if something bad happened.
	   Now you may ask why not set the header inside from response engine? because to do that we could have one of these four downsides:
	   1.to have access to context.IContext or *Context(if *Context then default engines should live here in iris repo)
	   and if we have context.IContext or *Context we will not be able to set a fast built'n gzip option,
	   because we would copy the contents to the gzip writer, and after copy these contents back to the response body's writer
	   but with an io.Writer as parameter we can simple change this writer to gzip writer and continue to the response engine after.
	   2. we could make something like ResponseWriter struct  {	io.Writer,Header *fasthttp.ResponseHeader}
	   inside iris repo(then the default response engines should exists in the iris repo and configuration will depends on the iris' configs )
	   or inside context/ folder inside iris repo, then the user/dev should import this path to
	   do his/her response engine, and I want simple things as usual, also we would make a pool for this response writer and create new if not available exist,
	    and this is performarnce downs witch I dissalow on Iris whne no need.
	   3. to have 4 parameters, the writer, the headers(again the user should import the fasthttp to do his/her response engine and I want simple things, as I told before),
	   the object and the optional parameters
	   4. one more function to implement like 'ContentType() string', but if we select this we lose the functionality for ResponseEngine created as simple function,
	   and the biggest issue will be that one response engine must explicit exists for one content type, the user/dev will not be available (to easly)
	   to set the content type for the engine.
	   these were the reasons I decide to set the content type by the frontend iris API itself and not taken by the response engine.

		 The Response will have two parameters (one required only) interface{], ...options}, and two return values([]byte,error)
		 The (first) parameter will be an interface{}, for json a json struct, for xml an xml struct, for binary data .([]byte) and so on
		 There will be available a second optional parameter, map of options, the "gzip" option will be built'n implemented by iris
		 so the response engines no need to manually add gzip support(same with template engines).
		 The Charset will be added to the headers automatically, for the previous example of json and the default charset which is UTF-8
		 the end "Content-Type" header content will be: "application/json; charset=UTF-8"
	   if the registered content type is not a $content/type then the text/plain will be sent to the client.

		 OR WAIT,	some engines maybe want to set the content type or other headers dynamically or render a response depends on cookies or some other existence headers
		 on that situtions it will be impossible with this implementation I explained before, so...
		 access to context.IContext  and return the []byte, in order to be able to add the built'n gzip support
		 the dev/user will have to make this import no no no we stick to the previous though, because
		 if the user wants to check all that he/she can just use a middleware with .Use/.UseFunc
		 this is not a middleware implementation, this is a custom content rendering, let's stick to that.

		 Ok I did that and I realized that template and response engines, final method structure (string,interface{},options...) is the same
		 so I make the ctx.Render/RenderWithStatus to work with both engines, so the developer can use any type of response engine and render it with ease.
		 Maybe at the future I could have one file 'render.go' which will contain the template engines and response engines, we will see, these all are unique so excuse me if something goes wrong xD

		 That's all. Hope some one (other than me) will understand the english here...
	*/

	// ResponseEngine is the interface which all response engines should implement to send responses
	// ResponseEngine(s) can be registered with,for example: iris.UseResponse(json.New(), "application/json")
	ResponseEngine interface {
		Response(interface{}, ...map[string]interface{}) ([]byte, error)
	}
	// ResponseEngineFunc is the alternative way to implement a ResponseEngine using a simple function
	ResponseEngineFunc func(interface{}, ...map[string]interface{}) ([]byte, error)

	// responseEngineMap is a wrapper with key (content type or name) values(engines) for the registered response engine
	// it contains all response engines for a specific contentType and two functions, render and toString
	// these will be used by the iris' context and iris' ResponseString, yes like TemplateToString
	// it's an internal struct, no need to be exported and return that on registration,
	// because the two top funcs will be easier to use by the user/dev for multiple engines
	responseEngineMap struct {
		values []ResponseEngine
		// this is used in order to the wrapper to be gettable by the responseEngines iteral,
		// if key is not a $content/type and contentType is not changed by the user/dev then the text/plain will be sent to the client
		key         string
		contentType string
	}
)

var (
	// markdown is custom type, used inside iris to initialize the defaults response engines if no other engine registered with these keys
	defaultResponseKeys = [...]string{contentText, contentXML, contentBinary, contentJSON, contentJSONP, contentMarkdown}
)

// Response returns  a response to the client(request's body content)
func (r ResponseEngineFunc) Response(obj interface{}, options ...map[string]interface{}) ([]byte, error) {
	return r(obj, options...)
}

var errNoResponseEngineFound = errors.New("No response engine found")

// on context: Send(contentType string, obj interface{}, ...options)

func (r *responseEngineMap) add(engine ResponseEngine) {
	r.values = append(r.values, engine)
}

// the gzip and charset options are built'n with iris
func (r *responseEngineMap) render(ctx *Context, obj interface{}, options ...map[string]interface{}) error {

	if r == nil {
		//render, but no response engine registered, this caused by context.RenderWithStatus, and responseEngines. getBy
		return errNoResponseEngineFound.Return()
	}

	var finalResult []byte

	for i, n := 0, len(r.values); i < n; i++ {
		result, err := r.values[i].Response(obj, options...)
		if err != nil { // fail on first the first error
			return err
		}
		finalResult = append(finalResult, result...)
	}

	gzipEnabled := ctx.framework.Config.Gzip
	charset := ctx.framework.Config.Charset
	if len(options) > 0 {
		gzipEnabled = getGzipOption(ctx, options[0]) // located to the template.go below the RenderOptions
		if chs := getCharsetOption(options[0]); chs != "" {
			charset = chs
		}
	}
	ctype := r.contentType

	if r.contentType != contentBinary { // set the charset only on non-binary data
		ctype += "; charset=" + charset
	}
	ctx.SetContentType(ctype)

	if gzipEnabled && ctx.clientAllowsGzip() {
		_, err := fasthttp.WriteGzip(ctx.RequestCtx.Response.BodyWriter(), finalResult)
		if err != nil {
			return err
		}
		ctx.RequestCtx.Response.Header.Add(varyHeader, acceptEncodingHeader)
		ctx.SetHeader(contentEncodingHeader, "gzip")
	} else {
		ctx.Response.SetBody(finalResult)
	}

	return nil
}

func (r *responseEngineMap) toString(obj interface{}, options ...map[string]interface{}) (string, error) {
	if r == nil {
		//render, but no response engine registered, this caused by context.RenderWithStatus, and responseEngines. getBy
		return "", errNoResponseEngineFound.Return()
	}
	var finalResult []byte
	for i, n := 0, len(r.values); i < n; i++ {
		result, err := r.values[i].Response(obj, options...)
		if err != nil {
			return "", err
		}
		finalResult = append(finalResult, result...)
	}
	return string(finalResult), nil
}

type responseEngines struct {
	engines []*responseEngineMap
}

// add accepts a simple response engine with its content type or key, key should not contains a dot('.').
// if key is a content type then it's the content type, but if it not, set the content type from the returned function,
// if it not called/changed then the default content type text/plain will be used.
// different content types for the same key will produce bugs, as it should!
// one key has one content type but many response engines ( one to many)
// note that the func should be used on the same call
func (r *responseEngines) add(engine ResponseEngine, forContentTypesOrKeys ...string) func(string) {
	if r.engines == nil {
		r.engines = make([]*responseEngineMap, 0)
	}

	var engineMap *responseEngineMap
	for _, key := range forContentTypesOrKeys {
		if strings.IndexByte(key, '.') != -1 { // the dot is not allowed as key
			continue // skip this engine
		}

		defaultCtypeAndKey := contentText
		if len(key) == 0 {
			//if empty key, then set it to text/plain
			key = defaultCtypeAndKey
		}

		engineMap = r.getBy(key)
		if engineMap == nil {

			ctype := defaultCtypeAndKey
			if strings.IndexByte(key, slashByte) != -1 { // pure check, but developer should know the content types at least.
				// we have 'valid' content type
				ctype = key
			}
			// the context.Markdown works without it but with .Render we will have problems without this:
			if key == contentMarkdown { // remember the text/markdown is just a custom internal iris content type, which in reallity renders html
				ctype = contentHTML
			}
			engineMap = &responseEngineMap{values: make([]ResponseEngine, 0), key: key, contentType: ctype}
			r.engines = append(r.engines, engineMap)
		}
		engineMap.add(engine)
	}

	return func(theContentType string) {
		// and this
		if theContentType == contentMarkdown {
			theContentType = contentHTML
		}

		engineMap.contentType = theContentType
	}

}

func (r *responseEngines) getBy(key string) *responseEngineMap {
	for i, n := 0, len(r.engines); i < n; i++ {
		if r.engines[i].key == key {
			return r.engines[i]
		}

	}
	return nil
}
