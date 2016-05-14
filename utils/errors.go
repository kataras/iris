package utils

import (
	"github.com/kataras/iris/errors"
)

var (
	// ErrNoZip returns an error with message: 'While creating file '+filename'. It's not a zip'
	ErrNoZip = errors.New("While installing file '%s'. It's not a zip")
	// ErrFileOpen returns an error with message: 'While opening a file. Trace: +specific error'
	ErrFileOpen = errors.New("While opening a file. Trace: %s")
	// ErrFileCreate returns an error with message: 'While creating a file. Trace: +specific error'
	ErrFileCreate = errors.New("While creating a file. Trace: %s")
	// ErrFileRemove returns an error with message: 'While removing a file. Trace: +specific error'
	ErrFileRemove = errors.New("While removing a file. Trace: %s")
	// ErrFileCopy returns an error with message: 'While copying files. Trace: +specific error'
	ErrFileCopy = errors.New("While copying files. Trace: %s")
	// ErrFileDownload returns an error with message: 'While downloading from +specific url. Trace: +specific error'
	ErrFileDownload = errors.New("While downloading from %s. Trace: %s")
	// ErrDirCreate returns an error with message: 'Unable to create directory on '+root dir'. Trace: +specific error
	ErrDirCreate = errors.New("Unable to create directory on '%s'. Trace: %s")
)
