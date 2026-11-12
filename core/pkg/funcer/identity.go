package funcer

import "strings"

// NameSource records how a FuncInfo's Name was produced. It exists because,
// since Go 1.27, two distinct function literals with identical code may share
// a single code pointer — so a name derived from the program counter is not
// a safe identity for anonymous functions anymore. Explicitly assigned and
// resolver-matched names remain safe: they are chosen by a human, not by the
// compiler's symbol generator.
type NameSource uint8

const (
	// NameDerived means the Name came straight from the runtime symbol of the
	// function's program counter, e.g. "mypackage.MyHandler" or
	// "mypackage.New.func1".
	NameDerived NameSource = iota
	// NameResolved means a SetName (iris.SetHandlerName) mapping matched and
	// replaced the derived name with a friendly one, e.g. "iris.recover".
	NameResolved
	// NameExplicit means the Name was assigned directly through WithFuncName,
	// FuncInfo.CopyFrom/CopyTo or SetNameFileAndLineByMethod.
	NameExplicit
)

// TrustedName reports whether "name" is safe to use as a function identity:
// anything except a compiler-generated anonymous symbol. Names produced by
// the compiler for function literals (a trailing ".funcN" chain, "glob..funcN"
// package-level literals and "-fm" method-value wrappers) are NOT trusted,
// because since Go 1.27 two distinct literals may share the code pointer the
// name was derived from.
func TrustedName(name string) bool {
	return name != "" && !isAnonymousFuncName(name)
}

// isAnonymousFuncName reports whether "name" is a compiler-generated symbol
// for a function literal or method-value wrapper.
func isAnonymousFuncName(name string) bool {
	if strings.HasSuffix(name, "-fm") {
		return true // method-value wrapper, e.g. "(*Limiter).serveHTTP-fm".
	}

	// Walk dot-separated segments from the right: numeric segments are nested
	// literal counters (".func2.1"); the first non-numeric segment decides.
	for {
		rest, seg, ok := strings.CutLast(name, ".")
		if !ok {
			return false
		}

		name = rest

		if isDigits(seg) {
			continue // e.g. the trailing "1" of "New.func2.1".
		}

		return strings.HasPrefix(seg, "func") && isDigits(seg[len("func"):])
	}
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}

	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}

	return true
}
