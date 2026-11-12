/*
Package context implements the per-request Context of the Iris web framework.

Most code reaches it through the iris.Context alias rather than importing this
package directly, but the methods, the options and the request-scoped helpers
are all documented here.

# Lifecycle and Pooling

A [Context] is taken from a sync.Pool when the request arrives and returned to
it when the handler chain ends. Its fields are cleared on release, so a
reference kept past the handler's return points at whatever request came next.
Do not store one, do not close over one in a goroutine, and do not use one from
two goroutines at once.

When work must outlive the request, take a copy first:

	cloned := ctx.Clone()
	go func() {
		if cloned.IsCanceled() {
			return
		}
		// ...
	}()

[Context.OnConnectionClose] registers a callback for client disconnects and
hands it a snapshot taken at registration time, for the same reason.

# Reading Requests

Path parameters come from [Context.Params], query values from
[Context.Query], and bodies from the Read family: [Context.ReadJSON], ReadForm,
ReadQuery, ReadXML, ReadYAML, ReadMsgPack and [Context.ReadBody], which picks a
decoder from the request's content type.

Three options in [ContextOptions] shape all of them. PostMaxMemory bounds
multipart form parsing and defaults to 32MB. BodyMaxMemory caps the total bytes
any body reader will accept and answers [ErrRequestEntityTooLarge] past the
limit; it is unlimited until you set it. RequestValidator runs after every
successful decode.

# Writing Responses

[Context.JSON], JSONP, XML, YAML, Markdown, Protobuf, MsgPack and
[Context.Negotiate] write the response; [Context.SetHeader],
[Context.AddHeader] and the cookie helpers shape it. The package-level
WriteJSON and its siblings are variables, so an application can replace the
encoder without changing a handler.

An error raised while writing goes to ContextOptions.ErrorHandler rather than
being returned to the caller, unless DisableErrorHandler is set.

# Security Considerations

  - [SafeFilename] is the sanitizer behind [Context.UploadFormFiles]. It rejects
    traversal, escapes out of the destination directory through a symlink, and
    reserved Windows device names such as NUL and COM1, which are refused on
    every platform so that an upload set does not vary by host. Call it yourself
    before [Context.SaveFormFile], which writes the name you give it verbatim.
  - [IsValidJSONPCallback] gates [Context.JSONP]. A callback that is not a plain
    JavaScript identifier reference, or that runs past 128 bytes, fails with
    [ErrInvalidJSONPCallback] before anything is written.
  - [Context.RemoteAddr] filters proxy headers against the configured private
    subnets and trusts none of them until ContextOptions.RemoteAddrHeaders names
    one. [Context.Addr] returns the client's own unfiltered claim and must not
    reach an authorization or rate-limiting decision.
  - [Context.SetHeader] replaces, which is what a single-valued header such as
    Access-Control-Allow-Origin needs. [Context.AddHeader] appends, for a
    list-valued one such as Vary.
  - [MakeDisposition] encodes a Content-Disposition filename so that a
    non-ASCII name cannot break out of the header.
  - A [User] carries the credentials it authenticated with. Never write one to
    the response; see [Context.User].

# Customizing

[Context.Options] returns this request's [ContextOptions] for a middleware to
adjust; [Context.OptionsReadOnly] is the read path. To carry your own type
through handlers instead of *Context, wrap it with [NewContextWrapper] or
[NewContextWrapperWithPool].
*/
package context
