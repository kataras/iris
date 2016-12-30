<a href="https://travis-ci.org/kataras/go-fs"><img src="https://img.shields.io/travis/kataras/go-fs.svg?style=flat-square" alt="Build Status"></a>
<a href="https://github.com/kataras/go-fs/blob/master/LICENSE"><img src="https://img.shields.io/badge/%20license-MIT%20%20License%20-E91E63.svg?style=flat-square" alt="License"></a>
<a href="https://github.com/kataras/go-fs/releases"><img src="https://img.shields.io/badge/%20release%20-%20v0.0.5-blue.svg?style=flat-square" alt="Releases"></a>
<a href="#docs"><img src="https://img.shields.io/badge/%20docs-reference-5272B4.svg?style=flat-square" alt="Read me docs"></a>
<a href="https://kataras.rocket.chat/channel/go-fs"><img src="https://img.shields.io/badge/%20community-chat-00BCD4.svg?style=flat-square" alt="Build Status"></a>
<a href="https://golang.org"><img src="https://img.shields.io/badge/powered_by-Go-3362c2.svg?style=flat-square" alt="Built with GoLang"></a>
<a href="#"><img src="https://img.shields.io/badge/platform-Any--OS-yellow.svg?style=flat-square" alt="Platforms"></a>


The package go-fs provides some common utilities which GoLang developers use when working with files, either system files or web files.


Installation
------------
The only requirement is the [Go Programming Language](https://golang.org/dl).

```bash
$ go get -u github.com/kataras/go-fs
```


Docs
------------


### Local file system helpers

```go
// DirectoryExists returns true if a directory(or file) exists, otherwise false
DirectoryExists(dir string) bool

// GetHomePath returns the user's $HOME directory
GetHomePath() string

// GetParentDir returns the parent directory of the passed targetDirectory
GetParentDir(targetDirectory string) string

// RemoveFile removes a file or a directory
RemoveFile(filePath string) error

// RenameDir renames (moves) oldpath to newpath.
// If newpath already exists, Rename replaces it.
// OS-specific restrictions may apply when oldpath and newpath are in different directories.
// If there is an error, it will be of type *LinkError.
//
// It's a copy of os.Rename
RenameDir(oldPath string, newPath string) error

// CopyFile  accepts full path of the source and full path of destination, if file exists it's overrides it
// this function doesn't checks for permissions and all that, it returns an error
CopyFile(source string, destination string) error

// CopyDir recursively copies a directory tree, attempting to preserve permissions.
// Source directory must exist.
CopyDir(source string, dest string)  error

// Unzip extracts a zipped file to the target location
// returns the path of the created folder (if any) and an error (if any)
Unzip(archive string, target string) (string, error)

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
TypeByExtension(fullfilename string) string

```

### Net/http handlers

```go
// FaviconHandler receives the favicon path and serves the favicon
FaviconHandler(favPath string) http.Handler

// StaticContentHandler returns the net/http.Handler interface to handle raw binary data,
// normally the data parameter was read by custom file reader or by variable
StaticContentHandler(data []byte, contentType string) http.Handler

// StaticFileHandler serves a static file such as css,js, favicons, static images
// it stores the file contents to the memory, doesn't supports seek because we read all-in-one the file, but seek is supported by net/http.ServeContent
StaticFileHandler(filename string) http.Handler

// SendStaticFileHandler sends a file for force-download to the client
// it stores the file contents to the memory, doesn't supports seek because we read all-in-one the file, but seek is supported by net/http.ServeContent
SendStaticFileHandler(filename string) http.Handler

// DirHandler serves a directory as web resource
// accepts a system Directory (string),
// a string which will be stripped off if not empty and
// Note 1: this is a dynamic dir handler, means that if a new file is added to the folder it will be served
// Note 2: it doesn't cache the system files, use it with your own risk, otherwise you can use the http.FileServer method, which is different of what I'm trying to do here.
// example:
// staticHandler := http.FileServer(http.Dir("static"))
// http.Handle("/static/", http.StripPrefix("/static/", staticHandler))
// converted to ->
// http.Handle("/static/", fs.DirHandler("./static", "/static/"))
DirHandler(dir string, strippedPrefix string) http.Handler
```

Read the [http_test.go](https://github.com/kataras/go-fs/blob/master/http_test.go) for more.

### Gzip Writer
Writes gzip compressed content to an underline io.Writer. It uses sync.Pool to reduce memory allocations.

**Better performance** through klauspost/compress package which provides us a gzip.Writer which is faster than Go standard's gzip package's writer.

```go
// NewGzipPool returns a new gzip writer pool, ready to use
NewGzipPool(Level int) *GzipPool

// DefaultGzipPool returns a new writer pool with Compressor's level setted to DefaultCompression
DefaultGzipPool() *GzipPool

// AcquireGzipWriter prepares a gzip writer and returns it
//
// see ReleaseGzipWriter
AcquireGzipWriter(w io.Writer) *gzip.Writer

// ReleaseGzipWriter called when flush/close and put the gzip writer back to the pool
//
// see AcquireGzipWriter
ReleaseGzipWriter(gzipWriter *gzip.Writer)

// WriteGzip writes a compressed form of p to the underlying io.Writer. The
// compressed bytes are not necessarily flushed until the Writer is closed
WriteGzip(w io.Writer, b []byte) (int, error)

```


- `AcquireGzipWriter` get a gzip writer, create new if no free writer available from inside the pool (sync.Pool).
- `ReleaseGzipWriter` releases puts a gzip writer to the pool (sync.Pool).
- `WriteGzip` gets a gzip writer, writes a compressed form of p to the underlying io.Writer. The
 compressed bytes are not necessarily flushed until the Writer is closed. Finally it Releases the particular gzip writer.

  > if these called from package level then the default gzip writer's pool is used to get/put and write

- `NewGzipPool` receives a compression level and returns a new gzip writer pool
- `DefaultGzipPool` returns a new gzip writer pool with DefaultCompression as the Compressor's Level

> New & Default are optional, use them to create more than one sync.Pool, if you expect thousands of writers working together


 Using default pool's writer to compress & write content

 ```go
 import "github.com/kataras/go-fs"

 var writer io.Writer

 // ... using default package's Pool to get a gzip writer
 n, err := fs.WriteGzip(writer, []byte("Compressed data and content here"))
 ```

 Using default Pool to get a gzip writer, compress & write content and finally release manually the gzip writer to the default Pool

 ```go
 import "github.com/kataras/go-fs"

 var writer io.Writer

 // ... using default writer's pool to get a gzip.Writer

 mygzipWriter := fs.AcquireGzipWriter(writer) // get a gzip.Writer from the default gzipwriter Pool

 n, err := mygzipWriter.WriteGzip([]byte("Compressed data and content here"))

 gzipwriter.ReleaseGzipWriter(mygzipWriter) // release this gzip.Writer to the default gzipwriter package's gzip writer Pool (sync.Pool)
 ```

Create and use a totally new gzip writer Pool

```go
import "github.com/kataras/go-fs"

var writer io.Writer
var gzipWriterPool = fs.NewGzipPool(fs.DefaultCompression)

// ...
n, err := gzipWriterPool.WriteGzip(writer, []byte("Compressed data and content here"))
```

Get a gzip writer Pool with the default options(compressor's Level)

```go
import "github.com/kataras/go-fs"

var writer io.Writer
var gzipWriterPool = fs.DefaultGzipPool() // returns a new default gzip writer pool

// ...
n, err := gzipWriterPool.WriteGzip(writer, []byte("Compressed data and content here"))
```

Acquire, Write and Release from a new(`.NewGzipPool/.DefaultGzipPool`) gzip writer Pool

 ```go
 import "github.com/kataras/go-fs"

 var writer io.Writer

 var gzipWriterPool = fs.DefaultGzipPool() // returns a new default gzip writer pool

 mygzipWriter := gzipWriterPool.AcquireGzipWriter(writer) // get a gzip.Writer from the new gzipWriterPool

 n, err := mygzipWriter.WriteGzip([]byte("Compressed data and content here"))

 gzipWriterPool.ReleaseGzipWriter(mygzipWriter) // release this gzip.Writer to the gzipWriterPool (sync.Pool)
 ```


### Working with remote zip files

```go
// DownloadZip downloads a zip file returns the downloaded filename and an error.
DownloadZip(zipURL string, newDir string, showOutputIndication bool) (string, error)

// Install is just the flow of: downloadZip -> unzip -> removeFile(zippedFile)
// accepts 3 parameters
//
// first parameter is the remote url file zip
// second parameter is the target directory
// third paremeter is a boolean which you can set to true to print out the progress
// returns a string(installedDirectory) and an error
//
// (string) installedDirectory is the directory which the zip file had, this is the real installation path
// the installedDirectory is not empty when the installation is succed, the targetDirectory is not already exists and no error happens
// the installedDirectory is empty when the installation is already done by previous time or an error happens
Install(remoteFileZip string, targetDirectory string, showOutputIndication bool) (string, error)
```

> Install = DownloadZip -> Unzip to the destination folder, remove the downloaded .zip, copy the inside extracted folder to the destination

Install many remote files(URI) to a single destination folder via installer instance

```go
type Installer struct {
	// InstallDir is the directory which all zipped downloads will be extracted
	// defaults to $HOME path
	InstallDir string
	// Indicator when it's true it shows an indicator about the installation process
	// defaults to false
	Indicator bool
	// RemoteFiles is the list of the files which should be downloaded when Install() called
	RemoteFiles []string
}

// Add adds a remote file(*.zip) to the list for download
Add(...string)

// Install installs all RemoteFiles, when this function called then the RemoteFiles are being resseted
// returns all installed paths and an error (if any)
// it continues on errors and returns them when the operation completed
Install() ([]string, error)
```

**Usage**

```go
package main

import "github.com/kataras/go-fs"

var testInstalledDir = fs.GetHomePath() + fs.PathSeparator + "mydir" + fs.PathSeparator

// remote file zip | expected output(installed) directory
var filesToInstall = map[string]string{
	"https://github.com/kataras/q/archive/master.zip":             testInstalledDir + "q-master",
	"https://github.com/kataras/iris/archive/master.zip":          testInstalledDir + "iris-master",
	"https://github.com/kataras/go-errors/archive/master.zip":     testInstalledDir + "go-errors-master",
	"https://github.com/kataras/go-gzipwriter/archive/master.zip": testInstalledDir + "go-gzipwriter-master",
	"https://github.com/kataras/go-events/archive/master.zip":     testInstalledDir + "go-events-master",
}

func main() {
	myInstaller := fs.NewInstaller(testInstalledDir)

	for remoteURI := range filesToInstall {
		myInstaller.Add(remoteURI)
	}

	installedDirs, err := myInstaller.Install()

	if err != nil {
		panic(err)
	}

	for _, installedDir := range installedDirs {
    println("New folder created: " + installedDir)
	}

}

```

When you want to install different zip files to different destination directories.

**Usage**

```go
package main

import "github.com/kataras/go-fs"

var testInstalledDir = fs.GetHomePath() + fs.PathSeparator + "mydir" + fs.PathSeparator

// remote file zip | expected output(installed) directory
var filesToInstall = map[string]string{
	"https://github.com/kataras/q/archive/master.zip":             testInstalledDir + "q-master",
	"https://github.com/kataras/iris/archive/master.zip":          testInstalledDir + "iris-master",
	"https://github.com/kataras/go-errors/archive/master.zip":     testInstalledDir + "go-errors-master",
	"https://github.com/kataras/go-gzipwriter/archive/master.zip": testInstalledDir + "go-gzipwriter-master",
	"https://github.com/kataras/go-events/archive/master.zip":     testInstalledDir + "go-events-master",
}

func main(){
	for remoteURI, expectedInstalledDir := range filesToInstall {

		installedDir, err := fs.Install(remoteURI, testInstalledDir, false)

		if err != nil {
			panic(err)
		}
		println("Installed: "+installedDir)
	}
}

```

Read the [installer_test.go](https://github.com/kataras/go-fs/blob/master/installer_test.go) for more.



You do not need any other special explanations for this package, just navigate to the [godoc](https://godoc.org/github.com/kataras/go-fs) or the [source](https://github.com/kataras/go-fs/blob/master/http_test.go) [code](https://github.com/kataras/go-fs/blob/master/installer_test.go).


FAQ
------------
Explore [these questions](https://github.com/kataras/go-fs/issues?go-fs=label%3Aquestion) or navigate to the [community chat][Chat].

Versioning
------------

Current: **v0.0.5**



People
------------
The author of go-fs is [@kataras](https://github.com/kataras).

If you're **willing to donate**, feel free to send **any** amount through paypal

[![](https://www.paypalobjects.com/en_US/i/btn/btn_donateCC_LG.gif)](https://www.paypal.com/cgi-bin/webscr?cmd=_donations&business=kataras2006%40hotmail%2ecom&lc=GR&item_name=Iris%20web%20framework&item_number=iriswebframeworkdonationid2016&currency_code=EUR&bn=PP%2dDonationsBF%3abtn_donateCC_LG%2egif%3aNonHosted)


Contributing
------------
If you are interested in contributing to the go-fs project, please make a PR.

License
------------

This project is licensed under the MIT License.

License can be found [here](LICENSE).

[Travis Widget]: https://img.shields.io/travis/kataras/go-fs.svg?style=flat-square
[Travis]: http://travis-ci.org/kataras/go-fs
[License Widget]: https://img.shields.io/badge/license-MIT%20%20License%20-E91E63.svg?style=flat-square
[License]: https://github.com/kataras/go-fs/blob/master/LICENSE
[Release Widget]: https://img.shields.io/badge/release-v0.0.5-blue.svg?style=flat-square
[Release]: https://github.com/kataras/go-fs/releases
[Chat Widget]: https://img.shields.io/badge/community-chat-00BCD4.svg?style=flat-square
[Chat]: https://kataras.rocket.chat/channel/go-fs
[ChatMain]: https://kataras.rocket.chat/channel/go-fs
[ChatAlternative]: https://gitter.im/kataras/go-fs
[Report Widget]: https://img.shields.io/badge/report%20card-A%2B-F44336.svg?style=flat-square
[Report]: http://goreportcard.com/report/kataras/go-fs
[Documentation Widget]: https://img.shields.io/badge/documentation-reference-5272B4.svg?style=flat-square
[Documentation]: https://www.gitbook.com/book/kataras/go-fs/details
[Language Widget]: https://img.shields.io/badge/powered_by-Go-3362c2.svg?style=flat-square
[Language]: http://golang.org
[Platform Widget]: https://img.shields.io/badge/platform-Any--OS-gray.svg?style=flat-square
