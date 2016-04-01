// Copyright (c) 2016, Gerasimos Maropoulos
// All rights reserved.
//
// Redistribution and use in source and binary forms, with or without modification,
// are permitted provided that the following conditions are met:
//
// 1. Redistributions of source code must retain the above copyright notice,
//    this list of conditions and the following disclaimer.
//
// 2. Redistributions in binary form must reproduce the above copyright notice,
//	  this list of conditions and the following disclaimer
//    in the documentation and/or other materials provided with the distribution.
//
// 3. Neither the name of the copyright holder nor the names of its contributors may be used to endorse
//    or promote products derived from this software without specific prior written permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
// ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
// WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
// DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER AND CONTRIBUTOR, GERASIMOS MAROPOULOS BE LIABLE FOR ANY
// DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
// (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
// LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
// ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
// SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
package iris

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

//THESE ARE FROM Go Authors
var htmlReplacer = strings.NewReplacer(
	"&", "&amp;",
	"<", "&lt;",
	">", "&gt;",
	// "&#34;" is shorter than "&quot;".
	`"`, "&#34;",
	// "&#39;" is shorter than "&apos;" and apos was not in HTML until HTML5.
	"'", "&#39;",
)

func htmlEscape(s string) string {
	return htmlReplacer.Replace(s)
}

func findLower(a, b int) int {
	if a <= b {
		return a
	}
	return b
}

// check if middleware passsed to a route has cors
// this is a poor implementation which only works with the iris/middleware/cors middleware
// it's bad and anti-pattern to check if a kind of  middleware has passed but I don't have any other options right now
// because I don't want to check in the router if method == req.Method || method == "OPTIONS" this will be low at least 900-2000 nanoseconds
// I made a func CorsMethodMatch, which is setted to the router.methodMatch if and only if the user passed the middleware cors on any of the routes
// only then the  second check of || method == "OPTIONS" will be evalutated. This is the way iris is working and have the best performance, maybe poor code I don't like to do but I Have to do.
// look at .Plant here, and on station.forceOptimusPrime
func hasCors(route IRoute) bool {
	for _, h := range route.GetMiddleware() {
		if _, ok := h.(interface {
			IsMethodAllowed(method string) bool
		}); ok {
			return true
		}
	}

	return false
}

// these are experimentals, will be used inside plugins to extend their power.

// directoryExists returns true if a directory(or file) exists, otherwise false
func directoryExists(dir string) bool {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return false
	}
	return true
}

// downloadZip downloads a zip file returns the downloaded filename and an error.
//
// An indicator is always shown up to the terminal, so the user will know if (a plugin) try to download something
func downloadZip(zipURL string, newDir string) (string, error) {
	var err error
	var size int64
	finish := false

	go func() {
		i := 0
		print("\n|")
		print("_")
		print("|")

	printer:
		{
			i++

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
			if finish {
				goto ok
			}
			goto printer
		}

	ok:
	}()

	os.MkdirAll(newDir, os.ModeDir)
	tokens := strings.Split(zipURL, "/")
	fileName := newDir + tokens[len(tokens)-1]
	if !strings.HasSuffix(fileName, ".zip") {
		err = fmt.Errorf("Error while creating %s ,is not a zip", fileName)
		return "", err
	}

	output, err := os.Create(fileName)
	if err != nil {
		fmt.Println("Error while creating", fileName, "-", err)
		return "", nil
	}
	defer output.Close()
	response, err := http.Get(zipURL)
	if err != nil {
		fmt.Println("Error while downloading", zipURL, "-", err)
		return "", nil
	}
	defer response.Body.Close()

	size, err = io.Copy(output, response.Body)
	if err != nil {
		fmt.Println("Error while downloading", zipURL, "-", err)
		return "", nil
	}
	finish = true
	println("\010\010\010\010\010\010OK ", size, " bytes downloaded")
	return fileName, nil

}

// unzip extracts a zipped file to the target location
//
// it removes the zipped file after succesfuly completion
// returns a string with the path of the created folder (if any) and an error (if any)
func unzip(archive string, target string) (string, error) {
	reader, err := zip.OpenReader(archive)
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(target, 0755); err != nil {
		return "", err
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
			return "", err
		}
		defer fileReader.Close()

		targetFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			return "", err
		}
		defer targetFile.Close()

		if _, err := io.Copy(targetFile, fileReader); err != nil {
			return "", err
		}

	}

	reader.Close()
	return createdFolder, nil
}

// removeFile removes a file and returns an error, if any
func removeFile(filePath string) error {
	return os.Remove(filePath)
}

// install is just the flow of: downloadZip -> unzip -> removeFile(zippedFile)
// accepts 2 parameters
//
// first parameter is the remote url file zip
// second parameter is the target directory
// returns a string(installedDirectory) and an error
//
// (string) installedDirectory is the directory which the zip file had, this is the real installation path, you don't need to know what it's because these things maybe change to the future let's keep it to return the correct path.
// the installedDirectory is not empty when the installation is succed, the targetDirectory is not already exists and no error happens
// the installedDirectory is empty when the installation is already done by previous time or an error happens
func install(remoteFileZip string, targetDirectory string) (installedDirectory string, err error) {
	var zipFile string

	zipFile, err = downloadZip(remoteFileZip, targetDirectory)
	if err == nil {
		installedDirectory, err = unzip(zipFile, targetDirectory)
		if err == nil {
			installedDirectory += string(os.PathSeparator)
			removeFile(zipFile)
		}
	}
	return
}

func getParrentDir(targetDirectory string) string {
	lastSlashIndex := strings.LastIndexByte(targetDirectory, os.PathSeparator)
	//check if the slash is at the end , if yes then re- check without the last slash, we don't want /path/to/ , we want /path/to in order to get the /path/ which is the parent directory of the /path/to
	if lastSlashIndex == len(targetDirectory)-1 {
		lastSlashIndex = strings.LastIndexByte(targetDirectory[0:lastSlashIndex], os.PathSeparator)
	}

	parentDirectory := targetDirectory[0:lastSlashIndex]
	return parentDirectory
}
