package main

import (
	"crypto/md5"
	"fmt"
	"html/template"
	"io"
	"mime/multipart"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/middleware/basicauth"
)

func init() {
	os.Mkdir("./uploads", 0700)
}

const (
	maxSize   = 1 * iris.GB
	uploadDir = "./uploads"
)

func main() {
	app := iris.New()

	app.RegisterView(iris.HTML("./views", ".html"))
	// Serve assets (e.g. javascript, css).
	// app.HandleDir("/public", "./public")

	app.Get("/", index)

	app.Get("/upload", uploadView)
	app.Post("/upload", upload)

	filesRouter := app.Party("/files")
	{
		filesRouter.HandleDir("/", uploadDir, iris.DirOptions{
			Gzip:     true,
			ShowList: true,
			DirList: iris.DirListRich(iris.DirListRichOptions{
				// Optionally, use a custom template for listing:
				Tmpl: dirListRichTemplate,
			}),
		})

		auth := basicauth.New(basicauth.Config{
			Users: map[string]string{
				"myusername": "mypassword",
			},
		})

		filesRouter.Delete("/{file:path}", auth, deleteFile)
	}

	app.Listen(":8080")
}

func index(ctx iris.Context) {
	ctx.Redirect("/upload")
}

func uploadView(ctx iris.Context) {
	now := time.Now().Unix()
	h := md5.New()
	io.WriteString(h, strconv.FormatInt(now, 10))
	token := fmt.Sprintf("%x", h.Sum(nil))

	ctx.View("upload.html", token)
}

func upload(ctx iris.Context) {
	ctx.SetMaxRequestBodySize(maxSize)

	_, err := ctx.UploadFormFiles(uploadDir, beforeSave)
	if err != nil {
		ctx.StopWithError(iris.StatusPayloadTooRage, err)
		return
	}

	ctx.Redirect("/files")
}

func beforeSave(ctx iris.Context, file *multipart.FileHeader) {
	ip := ctx.RemoteAddr()
	ip = strings.ReplaceAll(ip, ".", "_")
	ip = strings.ReplaceAll(ip, ":", "_")

	file.Filename = ip + "-" + file.Filename
}

func deleteFile(ctx iris.Context) {
	// It does not contain the system path,
	// as we are not exposing it to the user.
	fileName := ctx.Params().Get("file")

	filePath := path.Join(uploadDir, fileName)

	if err := os.RemoveAll(filePath); err != nil {
		ctx.StopWithError(iris.StatusInternalServerError, err)
		return
	}

	ctx.Redirect("/files")
}

var dirListRichTemplate = template.Must(template.New("dirlist").
	Funcs(template.FuncMap{
		"formatBytes": func(b int64) string {
			const unit = 1000
			if b < unit {
				return fmt.Sprintf("%d B", b)
			}
			div, exp := int64(unit), 0
			for n := b / unit; n >= unit; n /= unit {
				div *= unit
				exp++
			}
			return fmt.Sprintf("%.1f %cB",
				float64(b)/float64(div), "kMGTPE"[exp])
		},
	}).Parse(`
<!DOCTYPE html>
<html lang="en">

<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>{{.Title}}</title>
    <style>
        a {
            padding: 8px 8px;
            text-decoration:none;
            cursor:pointer;
            color: #10a2ff;
        }
        table {
            position: absolute;
            top: 0;
            bottom: 0;
            left: 0;
            right: 0;
            height: 100%;
            width: 100%;
            border-collapse: collapse;
            border-spacing: 0;
            empty-cells: show;
            border: 1px solid #cbcbcb;
        }
        
        table caption {
            color: #000;
            font: italic 85%/1 arial, sans-serif;
            padding: 1em 0;
            text-align: center;
        }
        
        table td,
        table th {
            border-left: 1px solid #cbcbcb;
            border-width: 0 0 0 1px;
            font-size: inherit;
            margin: 0;
            overflow: visible;
            padding: 0.5em 1em;
        }
        
        table thead {
            background-color: #10a2ff;
            color: #fff;
            text-align: left;
            vertical-align: bottom;
        }
        
        table td {
            background-color: transparent;
        }

        .table-odd td {
            background-color: #f2f2f2;
        }

        .table-bordered td {
            border-bottom: 1px solid #cbcbcb;
        }
        .table-bordered tbody > tr:last-child > td {
            border-bottom-width: 0;
        }
	</style>
</head>
<body>
    <table class="table-bordered table-odd">
        <thead>
            <tr>
                <th>#</th>
                <th>Name</th>
				<th>Size</th>
				<th>Actions</th>
            </tr>
        </thead>
        <tbody>
            {{ range $idx, $file := .Files }}
            <tr>
                <td>{{ $idx }}</td>
				<td><a href="{{ $file.Path }}" title="{{ $file.ModTime }}">{{ $file.Name }}</a></td>
				{{ if $file.Info.IsDir }}
				<td>Dir</td>
				{{ else }}
				<td>{{ formatBytes $file.Info.Size }}</td>
				{{ end }}
				
				<td><input type="button" style="background-color:transparent;border:0px;cursor:pointer;" value="❌" onclick="deleteFile({{ $file.RelPath }})"/></td>
            </tr>
            {{ end }}
        </tbody>
	</table>
    <script type="text/javascript">
        function deleteFile(filename) {
			if (!confirm("Are you sure you want to delete "+filename+" ?")) {
				return;
			}

            fetch('/files/'+filename,
                {
					method: "DELETE",
					// If you don't want server to prompt for username/password:
					// credentials:"include",
					headers: {
						// "Authorization": "Basic " + btoa("myusername:mypassword")
						"X-Requested-With": "XMLHttpRequest",
					},
                }).
                then(data => location.reload()).
                catch(e => alert(e));
        }
    </script>
</body></html>
`))
