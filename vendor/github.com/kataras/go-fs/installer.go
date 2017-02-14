package fs

import (
	"github.com/kataras/go-errors"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

var (
	// errNoZip returns an error with message: 'While creating file '+filename'. It's not a zip'
	errNoZip = errors.New("While installing file '%s'. It's not a zip")
	// errFileDownload returns an error with message: 'While downloading from +specific url. Trace: +specific error'
	errFileDownload = errors.New("While downloading from %s. Trace: %s")
)

// ShowIndicator shows a silly terminal indicator for a process, close of the finish channel is done here.
func ShowIndicator(wr io.Writer, newLine bool) chan bool {
	finish := make(chan bool)
	go func() {
		if newLine {
			wr.Write([]byte("\n"))
		}
		wr.Write([]byte("|"))
		wr.Write([]byte("_"))
		wr.Write([]byte("|"))

		for {
			select {
			case v := <-finish:
				{
					if v {
						wr.Write([]byte("\010\010\010")) //remove the loading chars
						close(finish)
						return
					}
				}
			default:
				wr.Write([]byte("\010\010-"))
				time.Sleep(time.Second / 2)
				wr.Write([]byte("\010\\"))
				time.Sleep(time.Second / 2)
				wr.Write([]byte("\010|"))
				time.Sleep(time.Second / 2)
				wr.Write([]byte("\010/"))
				time.Sleep(time.Second / 2)
				wr.Write([]byte("\010-"))
				time.Sleep(time.Second / 2)
				wr.Write([]byte("|"))
			}
		}

	}()

	return finish
}

// DownloadZip downloads a zip file returns the downloaded filename and an error.
func DownloadZip(zipURL string, newDir string, showOutputIndication bool) (string, error) {
	var err error
	var size int64
	if showOutputIndication {
		finish := ShowIndicator(os.Stdout, true)

		defer func() {
			finish <- true
		}()
	}

	os.MkdirAll(newDir, 0755)
	tokens := strings.Split(zipURL, "/")
	fileName := newDir + tokens[len(tokens)-1]
	if !strings.HasSuffix(fileName, ".zip") {
		return "", errNoZip.Format(fileName)
	}

	output, err := os.Create(fileName)
	if err != nil {
		return "", errFileCreate.Format(err.Error())
	}
	defer output.Close()
	response, err := http.Get(zipURL)
	if err != nil {
		return "", errFileDownload.Format(zipURL, err.Error())
	}
	defer response.Body.Close()

	size, err = io.Copy(output, response.Body)
	if err != nil {
		return "", errFileCopy.Format(err.Error())
	}

	if showOutputIndication {
		print("OK ", size, " bytes downloaded")
	}

	return fileName, nil

}

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
func Install(remoteFileZip string, targetDirectory string, showOutputIndication bool) (installedDirectory string, err error) {
	var zipFile string
	zipFile, err = DownloadZip(remoteFileZip, targetDirectory, showOutputIndication)
	if err == nil {
		installedDirectory, err = Unzip(zipFile, targetDirectory)
		if err == nil {
			RemoveFile(zipFile)
		}
	}
	return
}

// Installer is useful when you have single output-target directory and multiple zip files to install
type Installer struct {
	// InstallDir is the directory which all zipped downloads will be extracted
	// defaults to $HOME path
	InstallDir string
	// Indicator when it's true it shows an indicator about the installation process
	// defaults to false
	Indicator bool
	// RemoteFiles is the list of the files which should be downloaded when Install() called
	RemoteFiles []string

	mu sync.Mutex
}

// NewInstaller returns an new Installer, it's just a builder to add remote files once and call Install to install all of them
// first parameter is the installed directory, if empty then it uses the user's $HOME path
// second parameter accepts optional remote zip files to be install
func NewInstaller(installDir string, remoteFilesZip ...string) *Installer {
	if installDir == "" {
		installDir = GetHomePath()
	}
	return &Installer{InstallDir: installDir, Indicator: false, RemoteFiles: remoteFilesZip}
}

// Add adds a remote file(*.zip) to the list for download
func (i *Installer) Add(remoteFilesZip ...string) {
	i.mu.Lock()
	i.RemoteFiles = append(i.RemoteFiles, remoteFilesZip...)
	i.mu.Unlock()
}

var errNoFilesToInstall = errors.New("No files to install, please use the .Add method to add remote zip files")

// Install installs all RemoteFiles, when this function called then the RemoteFiles are being resseted
// returns all installed paths and an error (if any)
// it continues on errors and returns them when the operation completed
func (i *Installer) Install() ([]string, error) {
	if len(i.RemoteFiles) == 0 {
		return nil, errNoFilesToInstall
	}

	allErrors := *errors.New("")
	var installedDirectories []string // not strict to the remote file len because it continues on error

	// create a copy of the remote files
	remoteFiles := append([]string{}, i.RemoteFiles...)
	// clear the installers's remote files
	i.mu.Lock()
	i.RemoteFiles = nil

	for _, remoteFileZip := range remoteFiles {
		p, err := Install(remoteFileZip, i.InstallDir, i.Indicator)
		if err != nil {
			allErrors = allErrors.AppendErr(err)
			// add back the remote file if the install of this remote file has failed
			i.RemoteFiles = append(i.RemoteFiles, remoteFileZip)
		}

		installedDirectories = append(installedDirectories, p)
	}
	i.mu.Unlock()
	if !allErrors.IsAppended() {
		return installedDirectories, nil
	}

	return installedDirectories, allErrors
}
