// package filepersist provdies a mechanism for persisting data to a file at a
// permanent location
package filepersist

import (
	"fmt"
	"io"
	"os"

	"github.com/getlantern/golog"
)

var (
	log = golog.LoggerFor("filepersist")
)

// Save saves the given data to the file at filename. If an existing file at
// that filename already exists, this simply chmods the existing file to match
// the given fileMode and otherwise leaves it alone.
func Save(filename string, data []byte, fileMode os.FileMode) error {
	log.Tracef("Attempting to open %v for creating, but only if it doesn't already exist", filename)
	file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_EXCL, fileMode)
	if err != nil {
		if !os.IsExist(err) {
			return fmt.Errorf("Unexpected error opening %s: %s", filename, err)
		}

		log.Tracef("%s already exists, check to make sure contents is the same", filename)
		if dataMatches(filename, data) {
			log.Tracef("Data in %s matches expected, using existing", filename)
			chmod(filename, fileMode)
			// TODO - maybe don't swallow the error, but returning something
			// unique so the caller can decide whether or not to ignore it.
			return nil
		}

		log.Tracef("Data in %s doesn't match expected, truncating file", filename)
		file, err = openAndTruncate(filename, fileMode, true)
		if err != nil {
			return fmt.Errorf("Unable to truncate %s: %s", filename, err)
		}
	}

	log.Tracef("Created new file at %s, writing data", filename)
	_, err = file.Write(data)
	if err != nil {
		if err := os.Remove(filename); err != nil {
			log.Debugf("Unable to remove file: %v", err)
		}
		return fmt.Errorf("Unable to write to file at %s: %s", filename, err)
	}
	if err := file.Sync(); err != nil {
		log.Debugf("Unable to sync file: %v", err)
	}
	if err := file.Close(); err != nil {
		log.Debugf("Unable to close file: %v", err)
	}

	log.Trace("File saved")
	return nil
}

func openAndTruncate(filename string, fileMode os.FileMode, removeIfNecessary bool) (*os.File, error) {
	file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_TRUNC, fileMode)
	if err != nil && os.IsPermission(err) && removeIfNecessary {
		log.Tracef("Permission denied truncating file %v, try to remove", filename)
		err = os.Remove(filename)
		if err != nil {
			return nil, fmt.Errorf("Unable to remove file %v: %v", filename, err)
		}
		return openAndTruncate(filename, fileMode, false)
	}

	return file, err
}

// dataMatches compares the file at filename byte for byte with the given data
func dataMatches(filename string, data []byte) bool {
	file, err := os.OpenFile(filename, os.O_RDONLY, 0)
	if err != nil {
		log.Tracef("Unable to open existing file at %s for reading: %s", filename, err)
		return false
	}
	fileInfo, err := file.Stat()
	if err != nil {
		log.Tracef("Unable to stat file %s", filename)
		return false
	}
	if fileInfo.Size() != int64(len(data)) {
		return false
	}
	b := make([]byte, 65536)
	i := 0
	for {
		n, err := file.Read(b)
		if err != nil && err != io.EOF {
			log.Tracef("Error reading %s for comparison: %s", filename, err)
			return false
		}
		for j := 0; j < n; j++ {
			if b[j] != data[i] {
				return false
			}
			i = i + 1
		}
		if err == io.EOF {
			break
		}
	}
	return true
}

func chmod(filename string, fileMode os.FileMode) {
	fi, err := os.Stat(filename)
	if err != nil || fi.Mode() != fileMode {
		log.Tracef("Chmodding %v", filename)
		err = os.Chmod(filename, fileMode)
		if err != nil {
			log.Debugf("Warning - unable to chmod %v: %v", filename, err)
		}
	}
}
