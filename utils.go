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
// DISCLAIMED. IN NO EVENT SHALL JULIEN SCHMIDT BE LIABLE FOR ANY
// DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
// (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
// LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
// ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
// SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
package iris

import (
	"archive/zip"
	"errors"
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

// these are experimentals, will be used inside plugins to extend their power.

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
func DownloadZip(zipUrl string, newDir string) (string, error) {
	var err error
	var size int64
	finish := false

	go func() {
		i := 0
		print("|")
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
	tokens := strings.Split(zipUrl, "/")
	fileName := newDir + tokens[len(tokens)-1]
	if !strings.HasSuffix(fileName, ".zip") {
		err = errors.New(fmt.Sprintf("Error while creating %s ,is not a zip", fileName))
		return "", err
	}

	output, err := os.Create(fileName)
	if err != nil {
		fmt.Println("Error while creating", fileName, "-", err)
		return "", nil
	}
	defer output.Close()
	response, err := http.Get(zipUrl)
	if err != nil {
		fmt.Println("Error while downloading", zipUrl, "-", err)
		return "", nil
	}
	defer response.Body.Close()

	size, err = io.Copy(output, response.Body)
	if err != nil {
		fmt.Println("Error while downloading", zipUrl, "-", err)
		return "", nil
	}
	finish = true
	println("\010\010\010\010\010\010OK ", size, " bytes downloaded")
	return fileName, nil

}

// Unzip extracts a zipped file to the target location
func Unzip(archive, target string) error {
	reader, err := zip.OpenReader(archive)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(target, 0755); err != nil {
		return err
	}

	for _, file := range reader.File {
		path := filepath.Join(target, file.Name)
		if file.FileInfo().IsDir() {
			os.MkdirAll(path, file.Mode())
			continue
		}

		fileReader, err := file.Open()
		if err != nil {
			return err
		}
		defer fileReader.Close()

		targetFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			return err
		}
		defer targetFile.Close()

		if _, err := io.Copy(targetFile, fileReader); err != nil {
			return err
		}
	}

	return nil
}
