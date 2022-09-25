package view

import (
	"fmt"
	"io/fs"
	"path/filepath"

	"github.com/kataras/iris/v12/context"
)

// walk recursively in "fileSystem" descends "root" path, calling "walkFn".
func walk(fileSystem fs.FS, root string, walkFn filepath.WalkFunc) error {
	if root != "" && root != "/" && root != "." {
		sub, err := fs.Sub(fileSystem, root)
		if err != nil {
			return err
		}

		fileSystem = sub
	}

	if root == "" {
		root = "."
	}

	return fs.WalkDir(fileSystem, root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}

		info, err := d.Info()
		if err != nil {
			if err != filepath.SkipDir {
				return fmt.Errorf("%s: %w", path, err)
			}

			return nil
		}

		if info.IsDir() {
			return nil
		}

		return walkFn(path, info, err)
	})

}

func asset(fileSystem fs.FS, name string) ([]byte, error) {
	return fs.ReadFile(fileSystem, name)
}

func getFS(fsOrDir interface{}) fs.FS {
	return context.ResolveFS(fsOrDir)
}
