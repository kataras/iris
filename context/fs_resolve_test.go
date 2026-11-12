package context_test

import (
	"io/fs"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/kataras/iris/v14/context"
)

// templatesTree is the shape a views directory has: pages at the root,
// layouts and partials below it, and one file two levels deep.
var templatesTree = map[string]string{
	"about.html":               "about",
	"index.html":               "index",
	"layouts/admin/panel.html": "panel",
	"layouts/main.html":        "main",
	"partials/footer.html":     "footer",
}

// writeTemplatesTree writes templatesTree to a new directory, next to a
// directory that holds nothing.
func writeTemplatesTree(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	for name, contents := range templatesTree {
		full := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}

		if err := os.WriteFile(full, []byte(contents), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	if err := os.Mkdir(filepath.Join(dir, "empty"), 0o755); err != nil {
		t.Fatal(err)
	}

	return dir
}

// walk returns what fs.WalkDir visits in fsys, in order: "name/" for a
// directory and "name=contents" for a file.
func walk(fsys fs.FS) ([]string, error) {
	var visited []string

	err := fs.WalkDir(fsys, ".", func(name string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			visited = append(visited, name+"/")
			return nil
		}

		contents, err := fs.ReadFile(fsys, name)
		if err != nil {
			return err
		}

		visited = append(visited, name+"="+string(contents))
		return nil
	})

	return visited, err
}

// list returns the entries of one directory of fsys, a directory as "name/".
func list(fsys fs.FS, name string) ([]string, error) {
	entries, err := fs.ReadDir(fsys, name)
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			names = append(names, entry.Name()+"/")
			continue
		}

		names = append(names, entry.Name())
	}

	return names, nil
}

// An http.FileSystem given to ResolveFS has to walk like any other fs.FS.
// The engines of iris-contrib/views read their templates with fs.WalkDir.
func TestResolveFSHTTPFileSystem(t *testing.T) {
	dir := writeTemplatesTree(t)

	tests := []struct {
		name       string
		fileSystem http.FileSystem
		// emptyDir is whether the file system can hold a directory with no
		// files, which one built from a list of files cannot.
		emptyDir bool
	}{
		{
			name:       "http.Dir",
			fileSystem: http.Dir(dir),
			emptyDir:   true,
		},
		{
			// An fs.FS behind http.FS, which is how an embed.FS reaches a
			// file server. Its Open returns an http.File, so ResolveFS
			// takes it as an http.FileSystem.
			name:       "http.FS",
			fileSystem: http.FS(os.DirFS(dir)),
			emptyDir:   true,
		},
		{
			// go-bindata names the entries of a directory by their whole
			// path with the OS separator.
			name:       "entries named by their path",
			fileSystem: memFS{files: templatesTree, pathNames: true},
		},
		{
			// net/http asks a file system for rooted names, so one keyed by
			// the request path knows no other form.
			name:       "rooted names only",
			fileSystem: memFS{files: templatesTree, rooted: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fsys := context.ResolveFS(tt.fileSystem)

			// keep drops the empty directory where there cannot be one.
			keep := func(names ...string) []string {
				if tt.emptyDir {
					return names
				}

				return slices.DeleteFunc(names, func(name string) bool {
					return name == "empty/"
				})
			}

			t.Run("WalkDir", func(t *testing.T) {
				want := keep(
					"./",
					"about.html=about",
					"empty/",
					"index.html=index",
					"layouts/",
					"layouts/admin/",
					"layouts/admin/panel.html=panel",
					"layouts/main.html=main",
					"partials/",
					"partials/footer.html=footer",
				)

				got, err := walk(fsys)
				if err != nil {
					t.Fatalf("walk: %v", err)
				}

				if !slices.Equal(got, want) {
					t.Fatalf("walk:\n got: %q\nwant: %q", got, want)
				}
			})

			// The direct entries only, a directory as a directory, sorted
			// by file name: the contract of fs.ReadDirFS.
			t.Run("ReadDir", func(t *testing.T) {
				dirs := []struct {
					name string
					want []string
				}{
					{".", keep("about.html", "empty/", "index.html", "layouts/", "partials/")},
					{"layouts", []string{"admin/", "main.html"}},
					{"layouts/admin", []string{"panel.html"}},
					{"partials", []string{"footer.html"}},
				}

				if tt.emptyDir {
					dirs = append(dirs, struct {
						name string
						want []string
					}{"empty", []string{}})
				}

				for _, d := range dirs {
					got, err := list(fsys, d.name)
					if err != nil {
						t.Errorf("ReadDir(%q): %v", d.name, err)
						continue
					}

					if !slices.Equal(got, d.want) {
						t.Errorf("ReadDir(%q):\n got: %q\nwant: %q", d.name, got, d.want)
					}
				}
			})
		})
	}

	// The same directory through os.DirFS is the reference.
	t.Run("http.Dir walks like os.DirFS", func(t *testing.T) {
		want, err := walk(os.DirFS(dir))
		if err != nil {
			t.Fatal(err)
		}

		got, err := walk(context.ResolveFS(http.Dir(dir)))
		if err != nil {
			t.Fatalf("walk: %v", err)
		}

		if !slices.Equal(got, want) {
			t.Fatalf("walk:\n got: %q\nwant: %q", got, want)
		}
	})
}

// memFS is an http.FileSystem over a map of slash separated file names. It
// stands in for the implementations that http.Dir does not behave like.
type memFS struct {
	files map[string]string
	// pathNames names the entries of a directory by their whole path with a
	// backslash, as the file system go-bindata generates does on Windows.
	pathNames bool
	// rooted refuses a name that does not start with a slash.
	rooted bool
}

func (m memFS) Open(name string) (http.File, error) {
	notExist := &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}

	if m.rooted && !strings.HasPrefix(name, "/") {
		return nil, notExist
	}

	name = strings.Trim(name, "/") // the root is "".

	if contents, ok := m.files[name]; ok {
		return &memFile{
			Reader: strings.NewReader(contents),
			info:   memInfo{name: m.entryName(name), size: int64(len(contents))},
		}, nil
	}

	// A directory exists when a file lives below it.
	prefix := ""
	if name != "" {
		prefix = name + "/"
	}

	children := make(map[string]fs.FileInfo)
	for file, contents := range m.files {
		rest, ok := strings.CutPrefix(file, prefix)
		if !ok {
			continue
		}

		child, _, isDir := strings.Cut(rest, "/")
		info := memInfo{name: m.entryName(prefix + child), dir: isDir}
		if !isDir {
			info.size = int64(len(contents))
		}

		children[child] = info
	}

	if len(children) == 0 {
		return nil, notExist
	}

	// Map order: whoever lists this directory has to sort it.
	infos := make([]fs.FileInfo, 0, len(children))
	for _, info := range children {
		infos = append(infos, info)
	}

	return &memFile{
		Reader:   strings.NewReader(""),
		info:     memInfo{name: m.entryName(name), dir: true},
		children: infos,
	}, nil
}

func (m memFS) entryName(name string) string {
	if m.pathNames {
		return strings.ReplaceAll(name, "/", `\`)
	}

	return path.Base(name)
}

type memFile struct {
	*strings.Reader
	info     memInfo
	children []fs.FileInfo
}

func (f *memFile) Close() error                       { return nil }
func (f *memFile) Stat() (fs.FileInfo, error)         { return f.info, nil }
func (f *memFile) Readdir(int) ([]fs.FileInfo, error) { return f.children, nil }

type memInfo struct {
	name string
	size int64
	dir  bool
}

func (i memInfo) Name() string       { return i.name }
func (i memInfo) Size() int64        { return i.size }
func (i memInfo) ModTime() time.Time { return time.Time{} }
func (i memInfo) IsDir() bool        { return i.dir }
func (i memInfo) Sys() any           { return nil }

func (i memInfo) Mode() fs.FileMode {
	if i.dir {
		return fs.ModeDir | 0o555
	}

	return 0o444
}
