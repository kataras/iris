package context_test

import (
	"os"
	"regexp"
	"slices"
	"strings"
	"testing"
)

var (
	// genericContextMethod matches a generic method declared on *Context, e.g.
	// "func (ctx *Context) BindJSON[Request any]() (Request, bool)".
	genericContextMethod = regexp.MustCompile(`func \(ctx \*Context\) [A-Za-z_][A-Za-z0-9_]*\[`)
	// writefDeclaration matches the printf-style method whose position in the
	// type is load bearing.
	writefDeclaration = regexp.MustCompile(`func \(ctx \*Context\) Writef\(`)
)

// Writef must be declared before every generic *Context method. The Go type
// checker orders a named type's methods by the sorted file names of the
// package, and go vet keys its "Writef is a printf wrapper" fact by that
// order; on go1.27.0 a generic method declared first shifts the key, so
// ctx.Writef("%d", "x") stops being reported to users of the framework and an
// unrelated method is reported in its place. before_generic_methods_writef.go
// exists to hold that position, and this test is what keeps a rename honest.
// Upstream: golang/go#81306, fixed in Go 1.27.2.
func TestWritefDeclaredBeforeGenericMethods(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}

	var fileNames []string
	for _, entry := range entries {
		name := entry.Name()
		// Test files are compiled into the test variant of the package only,
		// so they cannot move the methods of the package vet exports facts for.
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}

		fileNames = append(fileNames, name)
	}
	slices.Sort(fileNames)

	var (
		writefFile   string
		genericFiles []string
	)
	for _, name := range fileNames {
		source, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}

		if writefDeclaration.Match(source) {
			if writefFile != "" {
				t.Fatalf("Writef is declared in both %s and %s", writefFile, name)
			}

			writefFile = name
		}

		if genericContextMethod.Match(source) {
			genericFiles = append(genericFiles, name)
		}
	}

	if writefFile == "" {
		t.Fatal("no Writef declaration found in the package files")
	}

	if len(genericFiles) == 0 {
		t.Fatal("no generic *Context method found: this test can no longer prove anything")
	}

	if first := genericFiles[0]; writefFile >= first {
		t.Fatalf("Writef is declared in %s, which does not sort before %s, the first file "+
			"declaring a generic *Context method (all of them: %s). go vet then keys its printf "+
			"fact to the wrong method: ctx.Writef(\"%%d\", \"x\") stops being reported and an "+
			"unrelated *Context method is reported instead. Rename the file holding Writef so it "+
			"sorts first again, see its header comment", writefFile, first, strings.Join(genericFiles, ", "))
	}
}
