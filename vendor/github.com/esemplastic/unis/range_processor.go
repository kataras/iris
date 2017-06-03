// Copyright 2017 Γεράσιμος Μαρόπουλος. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package unis

// NewRange accepts "begin" and "end" indexes.
// Returns a new processor which tries to
// return the "original[begin:end]".
func NewRange(begin, end int) ProcessorFunc {
	if begin <= -1 || end <= -1 {
		return OriginProcessor
	}

	return func(str string) string {
		l := len(str)
		if begin < l && end < l {
			str = str[begin:end]
		}

		return str
	}
}

// NewRangeBegin almost same as NewRange but it
// accepts only a "begin" index, that means that
// it assumes that the "end" index is the last of the "original" string.
//
// Returns the "original[begin:]".
func NewRangeBegin(begin int) ProcessorFunc {
	if begin <= -1 {
		return OriginProcessor
	}
	return func(str string) string {
		l := len(str)
		if begin < l {
			str = str[begin:]
		}

		return str
	}
}

// NewRangeEnd almost same as NewRange but it
// accepts only an "end" index, that means that
// it assumes that the "start" index is 0 of the "original".
//
// Returns the "original[0:end]".
func NewRangeEnd(end int) ProcessorFunc {
	// end should be > 0
	if end <= 0 {
		return OriginProcessor
	}
	return func(str string) string {
		l := len(str)
		if end < l {
			str = str[0:end]
		}
		return str
	}
}
