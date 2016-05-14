package utils

import (
	"archive/zip"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	// ContentBINARY is the  string of "application/octet-stream response headers
	ContentBINARY = "application/octet-stream"
)

// DirectoryExists returns true if a directory(or file) exists, otherwise false
func DirectoryExists(dir string) bool {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return false
	}
	return true
}

// DownloadZip downloads a zip file returns the downloaded filename and an error.
//
// An indicator is always shown up to the terminal, so the user will know if (a plugin) try to download something
func DownloadZip(zipURL string, newDir string) (string, error) {
	var err error
	var size int64
	finish := make(chan bool)

	go func() {
		print("\n|")
		print("_")
		print("|")

		for {
			select {
			case v := <-finish:
				{
					if v {
						print("\010\010\010") //remove the loading chars
						close(finish)
						return
					}

				}
			default:
				print("\010\010-")
				time.Sleep(time.Second / 2)
				print("\010\\")
				time.Sleep(time.Second / 2)
				print("\010|")
				time.Sleep(time.Second / 2)
				print("\010/")
				time.Sleep(time.Second / 2)
				print("\010-")
				time.Sleep(time.Second / 2)
				print("|")
			}
		}

	}()

	os.MkdirAll(newDir, os.ModeDir)
	tokens := strings.Split(zipURL, "/")
	fileName := newDir + tokens[len(tokens)-1]
	if !strings.HasSuffix(fileName, ".zip") {
		return "", ErrNoZip.Format(fileName)
	}

	output, err := os.Create(fileName)
	if err != nil {
		return "", ErrFileCreate.Format(err.Error())
	}
	defer output.Close()
	response, err := http.Get(zipURL)
	if err != nil {
		return "", ErrFileDownload.Format(zipURL, err.Error())
	}
	defer response.Body.Close()

	size, err = io.Copy(output, response.Body)
	if err != nil {
		return "", ErrFileCopy.Format(err.Error())
	}
	finish <- true
	print("OK ", size, " bytes downloaded") //we keep that here so developer will always see in the terminal if a plugin downloads something
	return fileName, nil

}

// Unzip extracts a zipped file to the target location
//
// it removes the zipped file after successfully completion
// returns a string with the path of the created folder (if any) and an error (if any)
func Unzip(archive string, target string) (string, error) {
	reader, err := zip.OpenReader(archive)
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(target, 0755); err != nil {
		return "", ErrDirCreate.Format(target, err.Error())
	}
	createdFolder := ""
	for _, file := range reader.File {
		path := filepath.Join(target, file.Name)
		if file.FileInfo().IsDir() {
			os.MkdirAll(path, file.Mode())
			if createdFolder == "" {
				// this is the new directory that zip has
				createdFolder = path
			}
			continue
		}

		fileReader, err := file.Open()
		if err != nil {
			return "", ErrFileOpen.Format(err.Error())
		}
		defer fileReader.Close()

		targetFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			return "", ErrFileOpen.Format(err.Error())
		}
		defer targetFile.Close()

		if _, err := io.Copy(targetFile, fileReader); err != nil {
			return "", ErrFileCopy.Format(err.Error())
		}

	}

	reader.Close()
	return createdFolder, nil
}

// RemoveFile removes a file and returns an error, if any
func RemoveFile(filePath string) error {
	return ErrFileRemove.With(os.Remove(filePath))
}

// Install is just the flow of: downloadZip -> unzip -> removeFile(zippedFile)
// accepts 2 parameters
//
// first parameter is the remote url file zip
// second parameter is the target directory
// returns a string(installedDirectory) and an error
//
// (string) installedDirectory is the directory which the zip file had, this is the real installation path, you don't need to know what it's because these things maybe change to the future let's keep it to return the correct path.
// the installedDirectory is not empty when the installation is succed, the targetDirectory is not already exists and no error happens
// the installedDirectory is empty when the installation is already done by previous time or an error happens
func Install(remoteFileZip string, targetDirectory string) (installedDirectory string, err error) {
	var zipFile string

	zipFile, err = DownloadZip(remoteFileZip, targetDirectory)
	if err == nil {
		installedDirectory, err = Unzip(zipFile, targetDirectory)
		if err == nil {
			installedDirectory += string(os.PathSeparator)
			RemoveFile(zipFile)
		}
	}
	return
}

// CopyFile copy a file, accepts full path of the source and full path of destination, if file exists it's overrides it
// this function doesn't checks for permissions and all that, it returns an error if didn't worked
func CopyFile(source string, destination string) error {
	reader, err := os.Open(source)

	if err != nil {
		return ErrFileOpen.Format(err.Error())
	}

	defer reader.Close()

	writer, err := os.Create(destination)
	if err != nil {
		return ErrFileCreate.Format(err.Error())
	}

	defer writer.Close()

	_, err = io.Copy(writer, reader)
	if err != nil {
		return ErrFileCopy.Format(err.Error())
	}

	err = writer.Sync()
	if err != nil {
		return ErrFileCopy.Format(err.Error())
	}

	return nil
}

// TypeByExtension returns the MIME type associated with the file extension ext.
// The extension ext should begin with a leading dot, as in ".html".
// When ext has no associated type, TypeByExtension returns "".
//
// Extensions are looked up first case-sensitively, then case-insensitively.
//
// The built-in table is small but on unix it is augmented by the local
// system's mime.types file(s) if available under one or more of these
// names:
//
//   /etc/mime.types
//   /etc/apache2/mime.types
//   /etc/apache/mime.types
//
// On Windows, MIME types are extracted from the registry.
//
// Text types have the charset parameter set to "utf-8" by default.
func TypeByExtension(fullfilename string) (t string) {
	ext := filepath.Ext(fullfilename)
	//these should be found by the windows(registry) and unix(apache) but on windows some machines have problems on this part.
	if t = mime.TypeByExtension(ext); t == "" {
		// no use of map here because we will have to lock/unlock it, by hand is better, no problem:
		if ext == ".json" {
			t = "application/json"
		} else if ext == ".zip" {
			t = "application/zip"
		} else if ext == ".3gp" {
			t = "video/3gpp"
		} else if ext == ".7z" {
			t = "application/x-7z-compressed"
		} else if ext == ".ace" {
			t = "application/x-ace-compressed"
		} else if ext == ".aac" {
			t = "audio/x-aac"
		} else {
			t = ContentBINARY
		}
	}
	return
}

// GetParentDir returns the parent directory(string) of the passed targetDirectory (string)
func GetParentDir(targetDirectory string) string {
	lastSlashIndex := strings.LastIndexByte(targetDirectory, os.PathSeparator)
	//check if the slash is at the end , if yes then re- check without the last slash, we don't want /path/to/ , we want /path/to in order to get the /path/ which is the parent directory of the /path/to
	if lastSlashIndex == len(targetDirectory)-1 {
		lastSlashIndex = strings.LastIndexByte(targetDirectory[0:lastSlashIndex], os.PathSeparator)
	}

	parentDirectory := targetDirectory[0:lastSlashIndex]
	return parentDirectory
}

/*
	// 3-BSD License for package fsnotify/fsnotify
	// Copyright (c) 2012 The Go Authors. All rights reserved.
	// Copyright (c) 2012 fsnotify Authors. All rights reserved.
	"github.com/fsnotify/fsnotify"
	//
	"github.com/kataras/iris/errors"
	"github.com/kataras/iris/logger"

// WatchDirectoryChanges watches for directory changes and calls the 'evt' callback parameter
// unused after v2 but propably I will bring it back on v3

func WatchDirectoryChanges(rootPath string, evt func(filename string), logger ...*logger.Logger) {
	watcher, err := fsnotify.NewWatcher()

	if err != nil {
		if len(logger) > 0 {
			errors.Printf(logger[0], err)
		}
		return
	}

	go func() {
		var lastChange = time.Now()
		var i = 0
		for {
			select {
			case event := <-watcher.Events:
				if event.Op&fsnotify.Write == fsnotify.Write {
					//this is received two times, the last time is the real changed file, so
					i++
					if i%2 == 0 {
						if time.Now().After(lastChange.Add(time.Duration(1) * time.Second)) {
							lastChange = time.Now()
							evt(event.Name)
						}
					}

				}
			case err := <-watcher.Errors:
				if len(logger) > 0 {
					errors.Printf(logger[0], err)
				}
			}
		}
	}()

	err = watcher.Add(rootPath)
	if err != nil {
		if len(logger) > 0 {
			errors.Printf(logger[0], err)
		}
	}

}*/
