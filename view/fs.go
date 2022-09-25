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
			return fmt.Errorf("walk: %s: %w", path, err)
		}

		info, err := d.Info()
		if err != nil {
			if err != filepath.SkipDir {
				return fmt.Errorf("walk stat: %s: %w", path, err)
			}

			return nil
		}

		if info.IsDir() {
			return nil
		}

		walkFnErr := walkFn(path, info, err)
		if walkFnErr != nil {
			return fmt.Errorf("walk: walkFn: %w", walkFnErr)
		}

		return nil
	})

}

func asset(fileSystem fs.FS, name string) ([]byte, error) {
	data, err := fs.ReadFile(fileSystem, name)
	if err != nil {
		return nil, fmt.Errorf("asset: read file: %w", err)
	}

	return data, nil
}

func getFS(fsOrDir interface{}) fs.FS {
	return context.ResolveFS(fsOrDir)
}

func getRootDirName(fileSystem fs.FS) string {
	rootDirFile, err := fileSystem.Open(".")
	if err == nil {
		rootDirStat, err := rootDirFile.Stat()
		if err == nil {
			return rootDirStat.Name()
		}
	}

	return ""
}
