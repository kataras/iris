//go:build windows

package fileserver

import (
	"os"
	"syscall"
)

// isHiddenByAttributes reports the FILE_ATTRIBUTE_HIDDEN bit, and whether the
// FileInfo carried real Win32 attributes at all.
//
// Sys() is nil for embed.FS entries and is this package's own *fileInfo for a
// cached file system, so the caller must have a fallback.
func isHiddenByAttributes(file os.FileInfo) (hidden, ok bool) {
	attrs, ok := file.Sys().(*syscall.Win32FileAttributeData)
	if !ok {
		return false, false
	}

	return attrs.FileAttributes&syscall.FILE_ATTRIBUTE_HIDDEN != 0, true
}
