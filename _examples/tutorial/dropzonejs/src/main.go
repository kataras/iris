package main

import (
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"github.com/kataras/iris"

	"github.com/nfnt/resize"
)

// $ go get -u github.com/nfnt/resize

const uploadsDir = "./public/uploads/"

type uploadedFile struct {
	// {name: "", size: } are the dropzone's only requirements.
	Name string `json:"name"`
	Size int64  `json:"size"`
}

type uploadedFiles struct {
	dir   string
	items []uploadedFile
	mu    sync.RWMutex // slices are safe but RWMutex is a good practise for you.
}

func scanUploads(dir string) *uploadedFiles {

	f := new(uploadedFiles)

	lindex := dir[len(dir)-1]
	if lindex != os.PathSeparator && lindex != '/' {
		dir += string(os.PathSeparator)
	}

	// create directories if necessary
	// and if, then return empty uploaded files; skipping the scan.
	if err := os.MkdirAll(dir, os.FileMode(0666)); err != nil {
		return f
	}

	// otherwise scan the given "dir" for files.
	f.scan(dir)
	return f
}

func (f *uploadedFiles) scan(dir string) {
	f.dir = dir
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {

		// if it's directory or a thumbnail we saved earlier, skip it.
		if info.IsDir() || strings.HasPrefix(info.Name(), "thumbnail_") {
			return nil
		}

		f.add(info.Name(), info.Size())
		return nil
	})
}

func (f *uploadedFiles) add(name string, size int64) uploadedFile {
	uf := uploadedFile{
		Name: name,
		Size: size,
	}

	f.mu.Lock()
	f.items = append(f.items, uf)
	f.mu.Unlock()

	return uf
}

func (f *uploadedFiles) createThumbnail(uf uploadedFile) {
	file, err := os.Open(path.Join(f.dir, uf.Name))
	if err != nil {
		return
	}
	defer file.Close()

	name := strings.ToLower(uf.Name)

	out, err := os.OpenFile(f.dir+"thumbnail_"+uf.Name,
		os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return
	}
	defer out.Close()

	if strings.HasSuffix(name, ".jpg") {
		// decode jpeg into image.Image
		img, err := jpeg.Decode(file)
		if err != nil {
			return
		}

		// write new image to file
		resized := resize.Thumbnail(180, 180, img, resize.Lanczos3)
		jpeg.Encode(out, resized,
			&jpeg.Options{Quality: jpeg.DefaultQuality})

	} else if strings.HasSuffix(name, ".png") {
		img, err := png.Decode(file)
		if err != nil {
			return
		}

		// write new image to file
		resized := resize.Thumbnail(180, 180, img, resize.Lanczos3) // slower but better res
		png.Encode(out, resized)
	}
	// and so on... you got the point, this code can be simplify, as a practise.

}

func main() {
	app := iris.New()
	app.RegisterView(iris.HTML("./views", ".html"))

	app.StaticWeb("/public", "./public")

	app.Get("/", func(ctx iris.Context) {
		ctx.View("upload.html")
	})

	files := scanUploads(uploadsDir)

	app.Get("/uploads", func(ctx iris.Context) {
		ctx.JSON(files.items)
	})

	app.Post("/upload", iris.LimitRequestBodySize(10<<20), func(ctx iris.Context) {
		// Get the file from the dropzone request
		file, info, err := ctx.FormFile("file")
		if err != nil {
			ctx.StatusCode(iris.StatusInternalServerError)
			ctx.Application().Logger().Warnf("Error while uploading: %v", err.Error())
			return
		}

		defer file.Close()
		fname := info.Filename

		// Create a file with the same name
		// assuming that you have a folder named 'uploads'
		out, err := os.OpenFile(uploadsDir+fname,
			os.O_WRONLY|os.O_CREATE, 0666)

		if err != nil {
			ctx.StatusCode(iris.StatusInternalServerError)
			ctx.Application().Logger().Warnf("Error while preparing the new file: %v", err.Error())
			return
		}
		defer out.Close()

		io.Copy(out, file)

		// optionally, add that file to the list in order to be visible when refresh.
		uploadedFile := files.add(fname, info.Size)
		go files.createThumbnail(uploadedFile)
	})

	// start the server at http://localhost:8080
	app.Run(iris.Addr(":8080"))
}
