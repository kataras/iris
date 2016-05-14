package iris

import "github.com/kataras/iris/errors"

var (
	// Router, Party & Handler

	// ErrHandler returns na error with message: 'Passed argument is not func(*Context) neither an object which implements the iris.Handler with Serve(ctx *Context)
	// It seems to be a  +type Points to: +pointer.'
	ErrHandler = errors.New("Passed argument is not func(*Context) neither an object which implements the iris.Handler with Serve(ctx *Context)\n It seems to be a  %T Points to: %v.")
	// ErrHandleAnnotated returns an error with message: 'HandleAnnotated parse: +specific error(s)'
	ErrHandleAnnotated = errors.New("HandleAnnotated parse: %s")

	// Plugin

	// ErrPluginAlreadyExists returns an error with message: 'Cannot activate the same plugin again, plugin '+plugin name[+plugin description]' is already exists'
	ErrPluginAlreadyExists = errors.New("Cannot use the same plugin again, '%s[%s]' is already exists")
	// ErrPluginActivate returns an error with message: 'While trying to activate plugin '+plugin name'. Trace: +specific error'
	ErrPluginActivate = errors.New("While trying to activate plugin '%s'. Trace: %s")
	// ErrPluginRemoveNoPlugins returns an error with message: 'No plugins are registed yet, you cannot remove a plugin from an empty list!'
	ErrPluginRemoveNoPlugins = errors.New("No plugins are registed yet, you cannot remove a plugin from an empty list!")
	// ErrPluginRemoveEmptyName returns an error with message: 'Plugin with an empty name cannot be removed'
	ErrPluginRemoveEmptyName = errors.New("Plugin with an empty name cannot be removed")
	// ErrPluginRemoveNotFound returns an error with message: 'Cannot remove a plugin which doesn't exists'
	ErrPluginRemoveNotFound = errors.New("Cannot remove a plugin which doesn't exists")
	// Context other

	// ErrNoForm returns an error with message: 'Request has no any valid form'
	ErrNoForm = errors.New("Request has no any valid form")
	// ErrWriteJSON returns an error with message: 'Before JSON be written to the body, JSON Encoder returned an error. Trace: +specific error'
	ErrWriteJSON = errors.New("Before JSON be written to the body, JSON Encoder returned an error. Trace: %s")
	// ErrRenderMarshalled returns an error with message: 'Before +type Rendering, MarshalIndent retured an error. Trace: +specific error'
	ErrRenderMarshalled = errors.New("Before +type Rendering, MarshalIndent returned an error. Trace: %s")
	// ErrReadBody returns an error with message: 'While trying to read +type from the request body. Trace +specific error'
	ErrReadBody = errors.New("While trying to read %s from the request body. Trace %s")
	// ErrServeContent returns an error with message: 'While trying to serve content to the client. Trace +specific error'
	ErrServeContent = errors.New("While trying to serve content to the client. Trace %s")

	// ErrTemplateExecute returns an error with message:'Unable to execute a template. Trace: +specific error'
	ErrTemplateExecute = errors.New("Unable to execute a template. Trace: %s")

	// ErrFlashNotFound returns an error with message: 'Unable to get flash message. Trace: Cookie does not exists'
	ErrFlashNotFound = errors.New("Unable to get flash message. Trace: Cookie does not exists")
	// ErrSessionNil returns an error with message: 'Unable to set session, Config().Session.Provider is nil, please refer to the docs!'
	ErrSessionNil = errors.New("Unable to set session, Config().Session.Provider is nil, please refer to the docs!")
)
