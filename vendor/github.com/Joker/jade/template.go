// Jade.go - template engine. Package implements Jade-lang templates for generating Go html/template output.
package jade

import (
	"bytes"
	"io"
	"io/ioutil"
	"path/filepath"
)

/*
Parse parses the template definition string to construct a representation of the template for execution.

Trivial usage:

	package main

	import (
		"fmt"
		"github.com/Joker/jade"
	)

	func main() {
		tpl, err := jade.Parse("tpl_name", "doctype 5: html: body: p Hello world!")
		if err != nil {
			fmt.Printf("Parse error: %v", err)
			return
		}

		fmt.Printf( "Output:\n\n%s", tpl  )
	}

Output:

	<!DOCTYPE html><html><body><p>Hello world!</p></body></html>
*/
func Parse(name, text string) (string, error) {
	outTpl, err := New(name).Parse(text)
	if err != nil {
		return "", err
	}
	b := new(bytes.Buffer)
	outTpl.WriteIn(b)
	return b.String(), nil
}

// ParseFile parse the jade template file in given filename
func ParseFile(filename string) (string, error) {
	bs, err := ioutil.ReadFile(filename)
	if err != nil {
		return "", err
	}
	return Parse(filepath.Base(filename), string(bs))
}

func (t *Tree) WriteIn(b io.Writer) {
	t.Root.WriteIn(b)
}
