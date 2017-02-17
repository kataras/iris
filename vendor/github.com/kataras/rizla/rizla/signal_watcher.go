package rizla

import (
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
)

type signalWatcher struct {
	underline *fsnotify.Watcher
	// to stop the for loop
	stopChan           chan bool
	hasStoppedManually bool
	errListeners       []WatcherErrorListener
	changeListeners    []WatcherChangeListener
}

var _ Watcher = &signalWatcher{}

// newSignalWatcher returns a new fsnotify wrapper
// which watching the operating system's file system's signals.
func newSignalWatcher() Watcher {
	watcher, werr := fsnotify.NewWatcher()
	if werr != nil {
		// yes we should panic on these things.
		panic(werr)
	}

	return &signalWatcher{
		underline: watcher,
		stopChan:  make(chan bool, 1),
	}
}

func (w *signalWatcher) OnError(evt WatcherErrorListener) {
	w.errListeners = append(w.errListeners, evt)
}

func (w *signalWatcher) OnChange(evt WatcherChangeListener) {
	w.changeListeners = append(w.changeListeners, evt)
}

func (w *signalWatcher) Stop() {
	w.stopChan <- true
	w.hasStoppedManually = true
	w.underline.Close()
}

func (w *signalWatcher) Loop() {
	// fsnotify needs to know the folder one by one, it doesn't cares about root's subdir yet.
	// so:
	for _, p := range projects {
		// add to the watcher first in order to watch changes and re-builds if the first build has fallen

		// add its root folder first
		if err := w.underline.Add(p.dir); err != nil {
			p.Err.Dangerf("\n" + err.Error() + "\n")
		}

		visitFn := func(path string, f os.FileInfo, err error) error {
			if f.IsDir() {
				// check if this subdir is allowed
				if p.Watcher(path) {
					if err := w.underline.Add(path); err != nil {
						p.Err.Dangerf("\n" + err.Error() + "\n")
					}
				} else {
					return filepath.SkipDir
				}

			}
			return nil
		}

		if err := filepath.Walk(p.dir, visitFn); err != nil {
			// actually it never panics here but keep it for note
			panic(err)
		}
	}

	defer func() {
		w.underline.Close()
		for _, p := range projects {
			killProcess(p.proc)
		}
		if !w.hasStoppedManually {
			for i := range w.errListeners {
				w.errListeners[i](errUnexpected)
			}
		}
	}()

	w.stopChan <- false

	// run the watcher
	for {
		select {
		case stop := <-w.stopChan:
			if stop {
				w.hasStoppedManually = true
				return
			}

		case event := <-w.underline.Events:
			// ignore CHMOD events
			if event.Op&fsnotify.Chmod == fsnotify.Chmod {
				continue
			}

			filename := event.Name
			for _, p := range projects {
				p.i++
				// fix two-times reload on windows
				if isWindows && p.i%2 != 0 {
					continue
				}

				if !p.DisableRuntimeDir { // if add folders to watch at runtime is enabled
					// if a folder created after the first Adds, add them here at runtime.
					if isDirectory(filename) && p.Watcher(filename) {
						if err := w.underline.Add(filename); err != nil {
							p.Err.Dangerf("\n" + err.Error() + "\n")
						}
					}
				}

				for i := range w.changeListeners {
					w.changeListeners[i](p, filename)
				}

			}
		case err := <-w.underline.Errors:
			if !w.hasStoppedManually {
				for i := range w.errListeners {
					w.errListeners[i](err)
				}
			}
		}
	}
}
