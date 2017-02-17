## 0.0.6 -> 0.0.7

Rizla uses the operating system's signals to fire a change because it is the fastest way and it consumes the minimal CPU.
But as the [feature request](https://github.com/kataras/rizla/issues/6) explains, some IDEs overrides the Operating System's signals, so I needed to change the things a bit in order to allow
looping-every-while and compare the file(s) with its modtime for new changes while in the same time keep the default as it's.

- **NEW**: Add a common interface for file system watchers in order to accoblish select between two methods of scanning for file changes.
    - file system's signals (default)
    - `filepath.Walk` (using the `-walk` flag)

### When to enable `-walk`?
When the default method doesn't works for your IDE's save method.

### How to enable `-walk`?
- If you're command line user: `rizla -walk main.go` , just append the `-walk` flag.
- If you use rizla behind your source code then use the `rizla.RunWith(rizla.WatcherFromFlag("-flag"))` instead of `rizla.Run()`.


## 0.0.4 -> 0.0.5 and 0.0.6

- **Fix**: Reload more than one time on Mac

## 0.0.3 -> 0.0.4

- **Added**: global `DefaultDisableProgramRerunOutput` and per-project `DisableProgramRerunOutput` option, to disable the program's output when reloads.
- **Fix** two-times reload on windows
- **Fix** fail to re-run when previous build error, issue [here](https://github.com/kataras/rizla/issues/1)

## 0.0.2 -> 0.0.3
- Change: `rizla.Out & rizla.Err` moved to the Project iteral (`project.Out` & `project.Err`), each project can have its own output now, and are type of *Printer
- Change/NEW: `project.OnChange` removed, new  `project.OnReload(func(string))` & `project.OnReloaded(func(string))` , with these you can change the output message when a file has changed and also when project reloaded, see the [project.go](https://github.com/kataras/rizla/blob/master/project.go) for more.

## 0.0.1 -> 0.0.2

- A lot of underline code improvements & fixes
- New: `project.Watcher(string) bool`
- New: `project.OnChange(string)`
- New: Allow watching new directories in runtime
- New: Rizla accepts all fs os events as change events
- Fix: Not watching third-level subdirectories, now watch everything except ` ".git", "node_modules", "vendor"` (you can change this behavior with the `project.Watcher`)
- Maybe I'm missing something, just upgrade with `go get -u github.com/kataras/rizla` and have fun :)
