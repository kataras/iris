//go:build !windows

package fileserver

import "os"

// isHiddenByAttributes always reports "no attributes available" outside Windows;
// the caller falls back to the dot-prefix convention, which is the only notion
// of a hidden file on these platforms.
func isHiddenByAttributes(os.FileInfo) (hidden, ok bool) {
	return false, false
}
