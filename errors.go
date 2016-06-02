package iris

import "github.com/kataras/iris/errors"

var (
	// Router, Party & Handler

	// ErrHandler returns na error with message: 'Passed argument is not func(*Context) neither an object which implements the iris.Handler with Serve(ctx *Context)
	// It seems to be a  +type Points to: +pointer.'
	ErrHandler = errors.New("Passed argument is not func(*Context) neither an object which implements the iris.Handler with Serve(ctx *Context)\n It seems to be a  %T Points to: %v.")
	// ErrHandleAnnotated returns an error with message: 'HandleAnnotated parse: +specific error(s)'
	ErrHandleAnnotated = errors.New("HandleAnnotated parse: %s")
	// ErrControllerContextNotFound returns an error with message: 'Context *iris.Context could not be found, the Controller won't be registed.'
	ErrControllerContextNotFound = errors.New("Context *iris.Context could not be found, the Controller won't be registed.")
	// ErrDirectoryFileNotFound returns an error with message: 'Directory or file %s couldn't found. Trace: +error trace'
	ErrDirectoryFileNotFound = errors.New("Directory or file %s couldn't found. Trace: %s")
	// ErrRenderRouteNotFound returns an error with message 'Route with name +route_name not found', used inside 'url' template func
	ErrRenderRouteNotFound = errors.New("Route with name %s not found")

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

	// ErrServeContent returns an error with message: 'While trying to serve content to the client. Trace +specific error'
	ErrServeContent = errors.New("While trying to serve content to the client. Trace %s")

	// ErrTemplateExecute returns an error with message:'Unable to execute a template. Trace: +specific error'
	ErrTemplateExecute = errors.New("Unable to execute a template. Trace: %s")

	// ErrFlashNotFound returns an error with message: 'Unable to get flash message. Trace: Cookie does not exists'
	ErrFlashNotFound = errors.New("Unable to get flash message. Trace: Cookie does not exists")
	// ErrSessionNil returns an error with message: 'Unable to set session, Config().Session.Provider is nil, please refer to the docs!'
	ErrSessionNil = errors.New("Unable to set session, Config().Session.Provider is nil, please refer to the docs!")
)
