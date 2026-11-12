package errors_test

import (
	"os"
	"regexp"
	"slices"
	"strings"
	"testing"
)

var (
	// mapperMethod matches a method declared on *Options or on ErrorCodeMap,
	// the two types that carry generic methods, e.g.
	// "func (opts *Options) MapErrors(" or "func (m ErrorCodeMap) Filter(".
	mapperMethod = regexp.MustCompile(`(?m)^func \([A-Za-z_][A-Za-z0-9_]* \*?(?:Options|ErrorCodeMap)\) [A-Za-z_][A-Za-z0-9_]*[\[(]`)
	// genericMapperMethod matches the generic ones among them, e.g.
	// "func (opts *Options) MapErrorAs[E error](".
	genericMapperMethod = regexp.MustCompile(`(?m)^func \([A-Za-z_][A-Za-z0-9_]* \*?(?:Options|ErrorCodeMap)\) [A-Za-z_][A-Za-z0-9_]*\[`)
)

// The generic methods of *Options and ErrorCodeMap must be declared after
// every other method of those two types. The Go type checker orders a named
// type's methods by the sorted file names of the package, and go vet and
// staticcheck key their facts about a method by that order; on this toolchain
// a generic method declared before another method shifts the key for every
// package that imports this one, so a fact lands on the wrong method.
// zz_generic_methods.go exists to hold the last position, and this test is
// what keeps a rename, or a generic method added somewhere else, honest.
// Upstream: golang/go#81306. The same guard in the context package is
// TestWritefDeclaredBeforeGenericMethods.
func TestOptionsGenericMethodsDeclaredLast(t *testing.T) {
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
		methodFiles  []string // files declaring any method of the two types.
		genericFiles []string // files declaring a generic one.
	)
	for _, name := range fileNames {
		source, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}

		all := mapperMethod.FindAll(source, -1)
		if len(all) == 0 {
			continue
		}
		methodFiles = append(methodFiles, name)

		generic := genericMapperMethod.FindAll(source, -1)
		if len(generic) == 0 {
			continue
		}
		genericFiles = append(genericFiles, name)

		if len(generic) != len(all) {
			t.Fatalf("%s declares generic and ordinary methods of *Options or ErrorCodeMap together. "+
				"An ordinary method declared after a generic one is the case that mis-keys go vet's "+
				"facts (golang/go#81306): keep the generic methods in zz_generic_methods.go and "+
				"everything else in another file", name)
		}
	}

	if len(genericFiles) == 0 {
		t.Fatal("no generic method of *Options or ErrorCodeMap found: this test can no longer prove anything")
	}

	if len(genericFiles) != 1 {
		t.Fatalf("generic methods of *Options or ErrorCodeMap are declared in more than one file (%s): "+
			"keep them all in zz_generic_methods.go", strings.Join(genericFiles, ", "))
	}

	if genericFile, last := genericFiles[0], methodFiles[len(methodFiles)-1]; genericFile != last {
		t.Fatalf("the generic methods of *Options and ErrorCodeMap are declared in %s, which sorts "+
			"before %s, a file declaring other methods of those types. go vet then keys its facts "+
			"to the wrong method for every package that imports this one (golang/go#81306). Rename "+
			"the file holding the generic methods so it sorts last again, see its header comment",
			genericFile, last)
	}
}
