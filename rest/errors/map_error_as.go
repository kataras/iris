package errors

import "strings"

// The typed mapping, MapErrorAs, is a generic method and lives in
// zz_generic_methods.go. That file's header says why it has to be there.

// MapErrorContains registers a substring mapping: an error whose Error() text
// contains substr is answered with e and the given message, which replaces the
// original text. The match is case-sensitive.
//
// This exists for the SQL and vendor errors which arrive as text only, a
// constraint violation reported by a driver that exposes no code for it being
// the usual case. A typed mapping is preferable wherever one is possible:
// prefer MapErrorAs, MapErrorTypes or MapErrors, because a driver is free to
// reword its text in the next release.
//
// An empty message keeps the original error text. An empty substr registers
// nothing at all, rather than a mapping which matches every error: a missing
// constant is a mistake, and the safe reading of it is "map nothing".
//
// The message replaces the text on the way out, so the driver's own words are
// gone before any collector sees them. When they are the only diagnostic
// there is, log the error first, as pgerrors does with errors.LogError, and
// map it after.
//
// Usage:
//
//	opts.MapErrorContains(errors.InvalidArgument,
//		"violates check constraint", "invalid value")
func (opts *Options) MapErrorContains(e ErrorCodeName, substr, message string) *Options {
	if opts.codeMap == nil {
		opts.codeMap = make(ErrorCodeMap)
	}
	opts.codeMap.MapErrorContains(e, substr, message)

	return opts
}

// MapErrorContains registers a substring mapping.
// See Options.MapErrorContains for the full documentation.
func (m ErrorCodeMap) MapErrorContains(e ErrorCodeName, substr, message string) ErrorCodeMap {
	if substr == "" {
		return m
	}

	return m.MapErrorFunc(e, func(err error) error {
		if !strings.Contains(err.Error(), substr) {
			return nil
		}

		if message == "" {
			return err
		}

		return New(message)
	})
}
