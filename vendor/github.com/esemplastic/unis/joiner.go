// Copyright 2017 Γεράσιμος Μαρόπουλος. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package unis

import "fmt"

// Joiner should be implemented by all string joiners.
type Joiner interface {
	// Join takes two pieces of strings
	// and returns a result of them, as one.
	Join(part1 string, part2 string) string
}

// JoinerFunc is the alias type of Joiner, it implements the Joiner also.
type JoinerFunc func(part1, part2 string) string

// Join takes two pieces of strings and returns a result of them, as one.
func (j JoinerFunc) Join(part1, part2 string) string {
	return j(part1, part2)
}

// NewJoiner returns a new joiner which joins
// two strings into one string, based on a "jointer".
func NewJoiner(jointer string) JoinerFunc {
	return func(part1, part2 string) string {
		return fmt.Sprintf("%s%s%s", part1, jointer, part2)
	}
}

// NewJoinerChain takes a Joiner and a chain of Processors and joins the
// Processors onto the output of the Joiner.
func NewJoinerChain(joiner Joiner, processors ...Processor) JoinerFunc {
	return func(part1, part2 string) (result string) {
		result = joiner.Join(part1, part2)
		for _, p := range processors {
			result = p.Process(result)
		}

		return
	}
}
