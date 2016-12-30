package fs

/// NOTE
// Please be careful when running this test it will create a go-installer-test folder in your $HOME directory
// At the end of these tests this should be removed
// also this test may take some time if low download speed, it downloads real files and installs them
//

import (
	"testing"
)

var testInstalledDir = GetHomePath() + PathSeparator + "go-installer-test" + PathSeparator

// remote file zip | expected output(installed) directory
var filesToInstall = map[string]string{
	"https://github.com/kataras/q/archive/master.zip":         testInstalledDir + "q-master",
	"https://github.com/kataras/iris/archive/master.zip":      testInstalledDir + "iris-master",
	"https://github.com/kataras/go-errors/archive/master.zip": testInstalledDir + "go-errors-master",
	"https://github.com/kataras/go-fs/archive/master.zip":     testInstalledDir + "go-fs-master",
	"https://github.com/kataras/go-events/archive/master.zip": testInstalledDir + "go-events-master",
}

func TestInstallerFull(t *testing.T) {
	defer RemoveFile(testInstalledDir)
	myInstaller := NewInstaller(testInstalledDir)

	for remoteURI := range filesToInstall {
		myInstaller.Add(remoteURI)
	}

	installedDirs, err := myInstaller.Install()

	if err != nil {
		t.Fatal(err)
	}

	// check for created files
	for _, installedDir := range installedDirs {

		if !DirectoryExists(installedDir) {
			t.Logf("Failed: %s\n", installedDir)
			t.Fatalf("Error while installation completed: Directories were not created(expected len = %d but got %d), files are not unzipped correctly to the root destination path(Destination = %s)", len(filesToInstall), len(installedDirs), testInstalledDir)
		}
	}

	// check if any remote file remains to the installer, should be 0
	if len(myInstaller.RemoteFiles) > 0 {
		t.Fatalf("Error while installation completed: Some remote files are reaming to the installer instance, should be len of 0 but got %d", len(myInstaller.RemoteFiles))
	}

}

func TestManualInstalls(t *testing.T) {
	// first check if already exists, from the previous test, if yes then remote the folder first
	RemoveFile(testInstalledDir)
	defer RemoveFile(testInstalledDir)
	for remoteURI, expectedInstalledDir := range filesToInstall {

		installedDir, err := Install(remoteURI, testInstalledDir, false)

		if err != nil {
			t.Fatal(err)
		}

		// check for created file
		if !DirectoryExists(installedDir) {
			t.Logf("Failed: %s\n", installedDir)
		}

		if expectedInstalledDir != installedDir {
			t.Fatalf("Expected installation dir to be: %s but got: %s", expectedInstalledDir, installedDir)
		}
	}

}
