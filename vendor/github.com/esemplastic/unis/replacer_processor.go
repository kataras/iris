// Copyright 2017 Γεράσιμος Μαρόπουλος. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package unis

import (
	"strings"
)

// NewReplacer accepts a map of old and new string values.
// The "old" will be replaced with the "new" one.
//
// Same as for loop with a strings.Replace("original", old, new, -1).
func NewReplacer(replacements map[string]string) ProcessorFunc {
	replacementsClone := make(map[string]string, len(replacements))

	for old, new := range replacements {
		replacementsClone[old] = new
	}

	return func(original string) (result string) {
		result = original
		for old, new := range replacementsClone {
			result = strings.Replace(result, old, new, -1)
		}

		return
	}
}
