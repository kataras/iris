// Copyright 2017 Γεράσιμος Μαρόπουλος. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package unis

import (
	"strings"
)

// Divider should be implemented by all string dividers.
type Divider interface {
	// Divide takes a string "original" and splits it into two pieces.
	Divide(original string) (part1 string, part2 string)
}

// DividerFunc is the alias type of Divider, it implements the Divider also.
type DividerFunc func(original string) (string, string)

// Divide takes a string "original" and splits it into two pieces.
func (d DividerFunc) Divide(original string) (string, string) {
	return d(original)
}

// NewDivider returns a new divider which splits
// a string into two pieces, based on the "separator".
//
// On failure returns the original path as its first
// return value, and empty as it's second.
func NewDivider(separator string) DividerFunc {
	return func(original string) (string, string) {
		// > 0 we will never use that method to split the first char.
		if idx := strings.Index(original, separator); idx > 0 {
			part1 := original[0 : idx+1]
			part2 := original[idx+1:]
			return part1, part2
		}
		return original, ""
	}
}

// NewInvertOnFailureDivider accepts a Divider "divider"
// and returns a new one.
//
// It calls the previous "divider" if succed then it returns
// the result as it is, otherwise it inverts the order of the result.
//
// Rembmer: the "divider" by its nature, returns the original string
// and empty as second parameter if the divide action has being a failure.
func NewInvertOnFailureDivider(divider Divider) DividerFunc {
	return func(original string) (string, string) {
		part1, part2 := divider.Divide(original)
		if part2 == "" {
			return part2, part1
		}
		return part1, part2
	}
}

// Divide is an action which runs a new divider based on the "separator"
// and the "original" string.
func Divide(original string, separator string) (string, string) {
	return NewDivider(separator).Divide(original)
}
