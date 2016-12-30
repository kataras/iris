## 0.0.4 -> 0.0.5

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
