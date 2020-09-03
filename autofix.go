package iris

import (
	"archive/zip"
	"bytes"
	stdContext "context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/kataras/golog"
)

const defaultModuleName = "app"

// simple function does not uses AST, it simply replaces import paths,
// creates a go.mod file if not exists and then run the `go mod tidy`
// command to remove old dependencies and install the new ones.
// It does NOT replaces breaking changes.
// The developer SHOULD visit the changelog(HISTORY.md) in order to learn
// everything about the new features and any breaking changes that comes with it.
func tryFix() error {
	wdir, err := filepath.Abs(".") // should return the current directory (on both go run & executable).
	if err != nil {
		return fmt.Errorf("can not resolve current working directory: %w", err)
	}

	// First of all, backup the current project,
	// so any changes can be reverted by the end developer.
	backupDest := wdir + "_irisbckp.zip"
	golog.Infof("Backup <%s> to <%s>", wdir, backupDest)

	err = zipDir(wdir, backupDest)
	if err != nil {
		return fmt.Errorf("backup dir: %w", err)
	}

	// go module.
	goModFile := filepath.Join(wdir, "go.mod")
	if !fileExists(goModFile) {

		golog.Warnf("Project is not a go module. Executing <go.mod init app>")
		f, err := os.Create(goModFile)
		if err != nil {
			return fmt.Errorf("go.mod: %w", os.ErrNotExist)
		}

		fmt.Fprintf(f, "module %s\ngo 1.15\n", defaultModuleName)
		f.Close()
	}

	// contnets replacements.
	golog.Infof("Updating...") // note: we will not replace GOPATH project paths.

	err = replaceDirContents(wdir, map[string]string{
		`"github.com/kataras/iris`: `"github.com/kataras/iris/v12`,
		// Note: we could use
		// regexp's FindAllSubmatch, take the dir part and replace
		// any HandleDir and e.t.c, but we are not going to do this.
		// Look the comment of the tryFix() function.
	})
	if err != nil {
		return fmt.Errorf("replace import paths: %w", err)
	}

	commands := []string{
		// "go clean --modcache",
		"go env -w GOPROXY=https://goproxy.cn,https://gocenter.io,https://goproxy.io,direct",
		"go mod tidy",
	}

	for _, c := range commands {
		if err = runCmd(wdir, c); err != nil {
			// print out the command, especially
			// with go env -w  the user should know it.
			// We use that because many of our users are living in China,
			// which the default goproxy is blocked).
			golog.Infof("$ %s", c)
			return fmt.Errorf("command <%s>: %w", c, err)
		}
	}

	return nil
}

func fileExists(path string) bool {
	stat, err := os.Stat(path)
	if err != nil {
		return os.IsExist(err)
	}

	return !stat.IsDir() && stat.Mode().IsRegular()
}

func runCmd(wdir, c string) error {
	ctx, cancel := stdContext.WithTimeout(stdContext.Background(), 2*time.Minute)
	defer cancel()

	parts := strings.Split(c, " ")
	name, args := parts[0], parts[1:]
	cmd := exec.CommandContext(ctx, name, args...)
	// cmd.Path = wdir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// zipDir zips a directory, recursively.
// It accepts a source directory and a destination zip file.
func zipDir(src, dest string) error {
	folderName := filepath.Base(src)

	file, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer file.Close()

	w := zip.NewWriter(file)
	defer w.Close()

	walkFunc := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		relPath := filepath.Join(folderName, strings.TrimPrefix(path, src))
		f, err := w.Create(relPath)
		if err != nil {
			return err
		}

		_, err = io.Copy(f, file)
		return err
	}

	return filepath.Walk(src, walkFunc)
}

func replaceDirContents(target string, replacements map[string]string) error {
	walkFunc := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !info.Mode().IsRegular() {
			return nil
		}

		file, err := os.OpenFile(path, os.O_RDWR, 0666)
		if err != nil {
			return err
		}
		defer file.Close()

		contents, ioErr := ioutil.ReadAll(file)
		if ioErr != nil {
			return ioErr
		}

		replaced := false
		for oldContent, newContent := range replacements {
			newContents := bytes.ReplaceAll(contents, []byte(oldContent), []byte(newContent))
			if len(newContents) > 0 {
				replaced = true
				contents = newContents[0:]
			}
		}

		if replaced {
			file.Truncate(0)
			file.Seek(0, 0)
			_, err = file.Write(contents)
			return err
		}

		return nil
	}

	return filepath.Walk(target, walkFunc)
}
