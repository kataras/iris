package rizla

type (
	// WatcherErrorListener the form the OnError event listener.
	WatcherErrorListener func(error)
	// WatcherChangeListener the form the OnChage event listener.
	// Receives the project which the change is valid (see Project.AllowReloadAfter)
	// and a second parameter which is the relative path to the changed file or directory.
	WatcherChangeListener func(p *Project, filename string)

	// Watcher a common interface which file system watchers should implement.
	Watcher interface {
		// OnChange registers an event listener which fires when a file change occurs.
		OnChange(WatcherChangeListener)
		// OnError registers an event listener which fires when a watcher error occurs.
		OnError(WatcherErrorListener)

		// Loop starts the watching and the loop.
		Loop()
		// Stop terminates the watcher.
		Stop()
	}
)

// WatcherFromFlag returns a new Watcher based on a string flah.
// This method keeps the watchers in the same spot.
//
// Note: this is why the internal watchers are not exported.
func WatcherFromFlag(flag string) Watcher {
	if flag == "-w" || flag == "-walk" || flag == "walk" {
		return newWalkWatcher()
	}
	// default: "signal"
	return newSignalWatcher()
}
