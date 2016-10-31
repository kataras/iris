package iris

import (
	"sync"

	"log"

	"gopkg.in/kataras/go-errors.v0"
	"gopkg.in/kataras/go-fs.v0"
)

var (
	// errPluginAlreadyExists returns an error with message: 'Cannot activate the same plugin again, plugin '+plugin name[+plugin description]' is already exists'
	errPluginAlreadyExists = errors.New("Cannot use the same plugin again, '%s[%s]' is already exists")
	// errPluginActivate returns an error with message: 'While trying to activate plugin '+plugin name'. Trace: +specific error'
	errPluginActivate = errors.New("While trying to activate plugin '%s'. Trace: %s")
	// errPluginRemoveNoPlugins returns an error with message: 'No plugins are registed yet, you cannot remove a plugin from an empty list!'
	errPluginRemoveNoPlugins = errors.New("No plugins are registed yet, you cannot remove a plugin from an empty list!")
	// errPluginRemoveEmptyName returns an error with message: 'Plugin with an empty name cannot be removed'
	errPluginRemoveEmptyName = errors.New("Plugin with an empty name cannot be removed")
	// errPluginRemoveNotFound returns an error with message: 'Cannot remove a plugin which doesn't exists'
	errPluginRemoveNotFound = errors.New("Cannot remove a plugin which doesn't exists")
)

type (
	// Plugin just an empty base for plugins
	// A Plugin can be added with: .Add(PreListenFunc(func(*Framework))) and so on... or
	// .Add(myPlugin{},myPlugin2{}) which myPlugin is  a struct with any of the methods below or
	// .PostListen(func(*Framework)) and so on...
	Plugin interface {
	}

	// pluginGetName implements the GetName() string method
	pluginGetName interface {
		// GetName has to returns the name of the plugin, a name is unique
		// name has to be not dependent from other methods of the plugin,
		// because it is being called even before the Activate
		GetName() string
	}

	// pluginGetDescription implements the GetDescription() string method
	pluginGetDescription interface {
		// GetDescription has to returns the description of what the plugins is used for
		GetDescription() string
	}

	// pluginActivate implements the Activate(pluginContainer) error method
	pluginActivate interface {
		// Activate called BEFORE the plugin being added to the plugins list,
		// if Activate returns none nil error then the plugin is not being added to the list
		// it is being called only one time
		//
		// PluginContainer parameter used to add other plugins if that's necessary by the plugin
		Activate(PluginContainer) error
	}
	// pluginPreLookup implements the PreRoute(Route) method
	pluginPreLookup interface {
		// PreLookup called before register a route
		PreLookup(Route)
	}
	// PreLookupFunc implements the simple function listener for the PreLookup(Route)
	PreLookupFunc func(Route)
	// pluginPreBuild implements the PreBuild(*Framework) method
	pluginPreBuild interface {
		// PreBuild it's being called once time, BEFORE the Server is started and before PreListen
		// is used to do work before all other things are ready
		// use this event if you want to add routes to your iris station
		// or make any changes to the iris main configuration
		// receiver is the station
		PreBuild(*Framework)
	}
	// PreBuildFunc implements the simple function listener for the PreBuild(*Framework)
	PreBuildFunc func(*Framework)
	// pluginPreListen implements the PreListen(*Framework) method
	pluginPreListen interface {
		// PreListen it's being called only one time, BEFORE the Server is started (if .Listen called)
		// is used to do work at the time all other things are ready to go
		// receiver is the station
		PreListen(*Framework)
	}
	// PreListenFunc implements the simple function listener for the PreListen(*Framework)
	PreListenFunc func(*Framework)
	// pluginPostListen implements the PostListen(*Framework) method
	pluginPostListen interface {
		// PostListen it's being called only one time, AFTER the Server is started (if .Listen called)
		// parameter is the station
		PostListen(*Framework)
	}
	// PostListenFunc implements the simple function listener for the PostListen(*Framework)
	PostListenFunc func(*Framework)
	// pluginPreClose implements the PreClose(*Framework) method
	pluginPreClose interface {
		// PreClose it's being called only one time, BEFORE the Iris .Close method
		// any plugin cleanup/clear memory happens here
		//
		// The plugin is deactivated after this state
		PreClose(*Framework)
	}
	// PreCloseFunc implements the simple function listener for the PreClose(*Framework)
	PreCloseFunc func(*Framework)

	// pluginPreDownload It's for the future, not being used, I need to create
	// and return an ActivatedPlugin type which will have it's methods, and pass it on .Activate
	// but now we return the whole pluginContainer, which I can't determinate which plugin tries to
	// download something, so we will leave it here for the future.
	pluginPreDownload interface {
		// PreDownload it's being called every time a plugin tries to download something
		//
		// first parameter is the plugin
		// second parameter is the download url
		// must return a boolean, if false then the plugin is not permmited to download this file
		PreDownload(plugin Plugin, downloadURL string) // bool
	}

	// PreDownloadFunc implements the simple function listener for the PreDownload(plugin,string)
	PreDownloadFunc func(Plugin, string)

	// PluginContainer is the interface which the pluginContainer should implements
	PluginContainer interface {
		Add(...Plugin) error
		Remove(string) error
		Len() int
		GetName(Plugin) string
		GetDescription(Plugin) string
		GetByName(string) Plugin
		Printf(string, ...interface{})
		Fired(string) int
		PreLookup(PreLookupFunc)
		DoPreLookup(Route)
		PreLookupFired() bool
		PreBuild(PreBuildFunc)
		DoPreBuild(*Framework)
		PreBuildFired() bool
		PreListen(PreListenFunc)
		DoPreListen(*Framework)
		DoPreListenParallel(*Framework)
		PreListenFired() bool
		PostListen(PostListenFunc)
		DoPostListen(*Framework)
		PostListenFired() bool
		PreClose(PreCloseFunc)
		DoPreClose(*Framework)
		PreCloseFired() bool
		PreDownload(PreDownloadFunc)
		DoPreDownload(Plugin, string)
		PreDownloadFired() bool
		//
		GetAll() []Plugin
		// GetDownloader is the only one module that is used and fire listeners at the same time in this file
		GetDownloader() PluginDownloadManager
	} //Note: custom event callbacks, never used internaly by Iris, but if you need them use this: github.com/kataras/go-events
	// PluginDownloadManager is the interface which the DownloadManager should implements
	PluginDownloadManager interface {
		DirectoryExists(string) bool
		DownloadZip(string, string) (string, error)
		Unzip(string, string) (string, error)
		Remove(string) error
		// install is just the flow of: downloadZip -> unzip -> removeFile(zippedFile)
		// accepts 2 parameters
		//
		// first parameter is the remote url file zip
		// second parameter is the target directory
		// returns a string(installedDirectory) and an error
		//
		// (string) installedDirectory is the directory which the zip file had, this is the real installation path, you don't need to know what it's because these things maybe change to the future let's keep it to return the correct path.
		// the installedDirectory is not empty when the installation is succed, the targetDirectory is not already exists and no error happens
		// the installedDirectory is empty when the installation is already done by previous time or an error happens
		Install(remoteFileZip string, targetDirectory string) (string, error)
	}

	// pluginDownloadManager is just a struch which exports the util's downloadZip, directoryExists, unzip methods, used by the plugins via the pluginContainer
	pluginDownloadManager struct {
	}
)

// convert the functions to plugin

// PreLookup called before register a route
func (fn PreLookupFunc) PreLookup(r Route) {
	fn(r)
}

// PreBuild it's being called once time, BEFORE the Server is started and before PreListen
// is used to do work before all other things are ready
// use this event if you want to add routes to your iris station
// or make any changes to the iris main configuration
// receiver is the station
func (fn PreBuildFunc) PreBuild(station *Framework) {
	fn(station)
}

// PreListen it's being called only one time, BEFORE the Server is started (if .Listen called)
// is used to do work at the time all other things are ready to go
//  parameter is the station
func (fn PreListenFunc) PreListen(station *Framework) {
	fn(station)
}

// PostListen it's being called only one time, AFTER the Server is started (if .Listen called)
// parameter is the station
func (fn PostListenFunc) PostListen(station *Framework) {
	fn(station)
}

// PreClose it's being called only one time, BEFORE the Iris .Close method
// any plugin cleanup/clear memory happens here
//
// The plugin is deactivated after this state
func (fn PreCloseFunc) PreClose(station *Framework) {
	fn(station)
}

// PreDownload it's being called every time a plugin tries to download something
//
// first parameter is the plugin
// second parameter is the download url
// must return a boolean, if false then the plugin is not permmited to download this file
func (fn PreDownloadFunc) PreDownload(pl Plugin, downloadURL string) {
	fn(pl, downloadURL)
}

//

var _ PluginDownloadManager = &pluginDownloadManager{}
var _ PluginContainer = &pluginContainer{}

// DirectoryExists returns true if a given local directory exists
func (d *pluginDownloadManager) DirectoryExists(dir string) bool {
	return fs.DirectoryExists(dir)
}

// DownloadZip downlodas a zip to the given local path location
func (d *pluginDownloadManager) DownloadZip(zipURL string, targetDir string) (string, error) {
	return fs.DownloadZip(zipURL, targetDir, true)
}

// Unzip unzips a zip to the given local path location
func (d *pluginDownloadManager) Unzip(archive string, target string) (string, error) {
	return fs.DownloadZip(archive, target, true)
}

// Remove deletes/removes/rm a file
func (d *pluginDownloadManager) Remove(filePath string) error {
	return fs.RemoveFile(filePath)
}

// Install is just the flow of the: DownloadZip->Unzip->Remove the zip
func (d *pluginDownloadManager) Install(remoteFileZip string, targetDirectory string) (string, error) {
	return fs.Install(remoteFileZip, targetDirectory, true)
}

// pluginContainer is the base container of all Iris, registed plugins
type pluginContainer struct {
	activatedPlugins []Plugin
	customEvents     map[string][]func()
	downloader       *pluginDownloadManager
	logger           *log.Logger
	mu               *sync.Mutex
	fired            map[string]int // event/plugin type name and the times fired
}

// newPluginContainer receives a logger and returns a new PluginContainer
func newPluginContainer(l *log.Logger) PluginContainer {
	return &pluginContainer{logger: l, fired: make(map[string]int, 0), mu: &sync.Mutex{}}
}

// Add activates the plugins and if succeed then adds it to the activated plugins list
func (p *pluginContainer) Add(plugins ...Plugin) error {
	for _, plugin := range plugins {

		if p.activatedPlugins == nil {
			p.activatedPlugins = make([]Plugin, 0)
		}

		// Check if it's a plugin first, has Activate GetName

		// Check if the plugin already exists
		pName := p.GetName(plugin)
		if pName != "" && p.GetByName(pName) != nil {
			return errPluginAlreadyExists.Format(pName, p.GetDescription(plugin))
		}
		// Activate the plugin, if no error then add it to the plugins
		if pluginObj, ok := plugin.(pluginActivate); ok {

			tempPluginContainer := *p
			err := pluginObj.Activate(&tempPluginContainer)
			if err != nil {
				return errPluginActivate.Format(pName, err.Error())
			}

			tempActivatedPluginsLen := len(tempPluginContainer.activatedPlugins)
			if tempActivatedPluginsLen != len(p.activatedPlugins)+tempActivatedPluginsLen+1 { // see test: plugin_test.go TestPluginActivate && TestPluginActivationError
				p.activatedPlugins = tempPluginContainer.activatedPlugins
			}

		}

		// All ok, add it to the plugins list
		p.activatedPlugins = append(p.activatedPlugins, plugin)
	}
	return nil
}

// Remove removes a plugin by it's name, if pluginName is empty "" or no plugin found with this name, then nothing is removed and a specific error is returned.
// This doesn't calls the PreClose method
func (p *pluginContainer) Remove(pluginName string) error {
	if p.activatedPlugins == nil {
		return errPluginRemoveNoPlugins
	}

	if pluginName == "" {
		//return error: cannot delete an unamed plugin
		return errPluginRemoveEmptyName
	}

	indexToRemove := -1
	for i := range p.activatedPlugins {
		if p.GetName(p.activatedPlugins[i]) == pluginName { // Note: if GetName is not implemented then the name is "" which is != with the plugiName, we checked this before.
			indexToRemove = i
		}
	}
	if indexToRemove == -1 { //if index stills -1 then no plugin was found with this name, just return an error. it is not a critical error.
		return errPluginRemoveNotFound
	}

	p.activatedPlugins = append(p.activatedPlugins[:indexToRemove], p.activatedPlugins[indexToRemove+1:]...)

	return nil
}

// Len returns the number of activate plugins
func (p *pluginContainer) Len() int {
	return len(p.activatedPlugins)
}

// GetName returns the name of a plugin, if no GetName() implemented it returns an empty string ""
func (p *pluginContainer) GetName(plugin Plugin) string {
	if pluginObj, ok := plugin.(pluginGetName); ok {
		return pluginObj.GetName()
	}
	return ""
}

// GetDescription returns the name of a plugin, if no GetDescription() implemented it returns an empty string ""
func (p *pluginContainer) GetDescription(plugin Plugin) string {
	if pluginObj, ok := plugin.(pluginGetDescription); ok {
		return pluginObj.GetDescription()
	}
	return ""
}

// GetByName returns a plugin instance by it's name
func (p *pluginContainer) GetByName(pluginName string) Plugin {
	if p.activatedPlugins == nil {
		return nil
	}

	for i := range p.activatedPlugins {
		if pluginObj, ok := p.activatedPlugins[i].(pluginGetName); ok {
			if pluginObj.GetName() == pluginName {
				return pluginObj
			}
		}
	}

	return nil
}

// GetAll returns all activated plugins
func (p *pluginContainer) GetAll() []Plugin {
	return p.activatedPlugins
}

// GetDownloader returns the download manager
func (p *pluginContainer) GetDownloader() PluginDownloadManager {
	// create it if and only if it used somewhere
	if p.downloader == nil {
		p.downloader = &pluginDownloadManager{}
	}
	return p.downloader
}

// Printf sends plain text to any registed logger (future), some plugins maybe want use this method
// maybe at the future I change it, instead of sync even-driven to async channels...
func (p *pluginContainer) Printf(format string, a ...interface{}) {
	if p.logger != nil {
		p.logger.Printf(format, a...) //for now just this.
	}

}

// fire adds a fired event on the (statically type named) map and returns the new times
func (p *pluginContainer) fire(name string) int {
	p.mu.Lock()
	var times int
	// maybe unnecessary but for clarity reasons
	if t, found := p.fired[name]; found {
		times = t
	}
	times++
	p.fired[name] = times
	p.mu.Unlock()
	return times
}

// Fired receives an event name/plugin type and returns the times which this event is fired and how many plugins are fired this event,
// if zero then it's not fired at all
func (p *pluginContainer) Fired(name string) (times int) {
	if t, found := p.fired[name]; found {
		times = t
	}
	return
}

// PreLookup adds a PreLookup plugin-function to the plugin flow container
func (p *pluginContainer) PreLookup(fn PreLookupFunc) {
	p.Add(fn)
}

// DoPreLookup raise all plugins which has the PreLookup method
func (p *pluginContainer) DoPreLookup(r Route) {
	for i := range p.activatedPlugins {
		// check if this method exists on our plugin obj, these are optionaly and call it
		if pluginObj, ok := p.activatedPlugins[i].(pluginPreLookup); ok {
			// fire will add times to the number of events fired this event
			p.fire("prelookup")
			pluginObj.PreLookup(r)
		}
	}
}

// PreLookupFired returns true if PreLookup event/ plugin type is fired at least one time
func (p *pluginContainer) PreLookupFired() bool {
	return p.Fired("prelookup") > 0
}

// PreBuild adds a PreBuild plugin-function to the plugin flow container
func (p *pluginContainer) PreBuild(fn PreBuildFunc) {
	p.Add(fn)
}

// DoPreBuild raise all plugins that have the PreBuild method
func (p *pluginContainer) DoPreBuild(station *Framework) {
	for i := range p.activatedPlugins {
		// check if this method exists on our plugin obj, these are optionaly and call it
		if pluginObj, ok := p.activatedPlugins[i].(pluginPreBuild); ok {
			pluginObj.PreBuild(station)
			p.fire("prebuild")
		}
	}
}

// PreBuildFired returns true if PreBuild event/ plugin type is fired at least one time
func (p *pluginContainer) PreBuildFired() bool {
	return p.Fired("prebuild") > 0
}

// PreListen adds a PreListen plugin-function to the plugin flow container
func (p *pluginContainer) PreListen(fn PreListenFunc) {
	p.Add(fn)
}

// DoPreListen raise all plugins which has the PreListen method
func (p *pluginContainer) DoPreListen(station *Framework) {
	for i := range p.activatedPlugins {
		// check if this method exists on our plugin obj, these are optionaly and call it
		if pluginObj, ok := p.activatedPlugins[i].(pluginPreListen); ok {
			pluginObj.PreListen(station)
			p.fire("prelisten")
		}
	}
}

// DoPreListenParallel raise all PreListen plugins 'at the same time'
func (p *pluginContainer) DoPreListenParallel(station *Framework) {
	var wg sync.WaitGroup

	for _, plugin := range p.activatedPlugins {
		wg.Add(1)
		// check if this method exists on our plugin obj, these are optionaly and call it
		go func(plugin Plugin) {
			if pluginObj, ok := plugin.(pluginPreListen); ok {
				pluginObj.PreListen(station)
				p.fire("prelisten")
			}

			wg.Done()

		}(plugin)
	}

	wg.Wait()

}

// PreListenFired returns true if PreListen or PreListenParallel event/ plugin type is fired at least one time
func (p *pluginContainer) PreListenFired() bool {
	return p.Fired("prelisten") > 0
}

// PostListen adds a PostListen plugin-function to the plugin flow container
func (p *pluginContainer) PostListen(fn PostListenFunc) {
	p.Add(fn)
}

// DoPostListen raise all plugins which has the DoPostListen method
func (p *pluginContainer) DoPostListen(station *Framework) {
	for i := range p.activatedPlugins {
		// check if this method exists on our plugin obj, these are optionaly and call it
		if pluginObj, ok := p.activatedPlugins[i].(pluginPostListen); ok {
			pluginObj.PostListen(station)
			p.fire("postlisten")
		}
	}
}

// PostListenFired returns true if PostListen event/ plugin type is fired at least one time
func (p *pluginContainer) PostListenFired() bool {
	return p.Fired("postlisten") > 0
}

// PreClose adds a PreClose plugin-function to the plugin flow container
func (p *pluginContainer) PreClose(fn PreCloseFunc) {
	p.Add(fn)
}

// DoPreClose raise all plugins which has the DoPreClose method
func (p *pluginContainer) DoPreClose(station *Framework) {
	for i := range p.activatedPlugins {
		// check if this method exists on our plugin obj, these are optionaly and call it
		if pluginObj, ok := p.activatedPlugins[i].(pluginPreClose); ok {
			pluginObj.PreClose(station)
			p.fire("preclose")
		}
	}
}

// PreCloseFired returns true if PreCLose event/ plugin type is fired at least one time
func (p *pluginContainer) PreCloseFired() bool {
	return p.Fired("preclose") > 0
}

// PreDownload adds a PreDownload plugin-function to the plugin flow container
func (p *pluginContainer) PreDownload(fn PreDownloadFunc) {
	p.Add(fn)
}

// DoPreDownload raise all plugins which has the DoPreDownload method
func (p *pluginContainer) DoPreDownload(pluginTryToDownload Plugin, downloadURL string) {
	for i := range p.activatedPlugins {
		// check if this method exists on our plugin obj, these are optionaly and call it
		if pluginObj, ok := p.activatedPlugins[i].(pluginPreDownload); ok {
			pluginObj.PreDownload(pluginTryToDownload, downloadURL)
			p.fire("predownload")
		}
	}
}

// PreDownloadFired returns true if PreDownload event/ plugin type is fired at least one time
func (p *pluginContainer) PreDownloadFired() bool {
	return p.Fired("predownload") > 0
}
