package main

import (
	"html/template"
	"math/rand"
	"net/url"
	"sync"
	"time"

	"gopkg.in/kataras/iris.v6"
	"gopkg.in/kataras/iris.v6/adaptors/httprouter"
	"gopkg.in/kataras/iris.v6/adaptors/view"
)

func main() {
	app := iris.New()
	app.Adapt(
		iris.DevLogger(),
		httprouter.New(),
		view.HTML("./templates", ".html").Reload(true),
	)

	// Serve static files (css)
	app.StaticWeb("/static", "./static_files")

	var mu sync.Mutex
	var urls = map[string]string{
		"iris": "http://support.iris-go.com",
	}

	app.Get("/", func(ctx *iris.Context) {
		ctx.Render("index.html", iris.Map{"url_count": len(urls)})
	})

	// find and execute a short url by its key
	// used on http://localhost:8080/url/dsaoj41u321dsa
	execShortURL := func(ctx *iris.Context, key string) {
		if key == "" {
			ctx.EmitError(iris.StatusBadRequest)
			return
		}

		value, found := urls[key]
		if !found {
			ctx.SetStatusCode(iris.StatusNotFound)
			ctx.Writef("Short URL for key: '%s' not found", key)
			return
		}

		ctx.Redirect(value, iris.StatusTemporaryRedirect)
	}
	app.Get("/url/:shortkey", func(ctx *iris.Context) {
		execShortURL(ctx, ctx.Param("shortkey"))
	})

	//  for wildcard subdomain (yeah.. cool) http://dsaoj41u321dsa.localhost:8080
	// Note:
	// if you want subdomains (chrome doesn't works on localhost, so you have to define other hostname on app.Listen)
	// app.Party("*.", func(ctx *iris.Context) {
	// 	execShortURL(ctx, ctx.Subdomain())
	// })

	app.Post("/url/shorten", func(ctx *iris.Context) {
		data := make(map[string]interface{}, 0)
		data["url_count"] = len(urls)
		value := ctx.FormValue("url")
		if value == "" {
			data["form_result"] = "You need to a enter a URL."
		} else {
			urlValue, err := url.ParseRequestURI(value)
			if err != nil {
				// ctx.JSON(iris.StatusInternalServerError,
				// 	iris.Map{"status": iris.StatusInternalServerError,
				// 		"error":  err.Error(),
				// 		"reason": "Invalid URL",
				// 	})
				data["form_result"] = "Invalid URL."
			} else {
				key := randomString(12)
				// Make sure that the key is unique
				for {
					if _, exists := urls[key]; !exists {
						break
					}
					key = randomString(8)
				}
				mu.Lock()
				urls[key] = urlValue.String()
				mu.Unlock()
				ctx.SetStatusCode(iris.StatusOK)
				shortenURL := "http://" + app.Config.VHost + "/url/" + key
				data["form_result"] = template.HTML("<pre>Here is your short URL: <a href='" + shortenURL + "'>" + shortenURL + " </a></pre>")
			}

		}
		ctx.Render("index.html", data)
	})

	app.Listen("localhost:8080")
}

//  +------------------------------------------------------------+
//  |                                                            |
//  |                      Random String                         |
//  |                                                            |
//  +------------------------------------------------------------+

const (
	letterBytes   = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

func randomString(n int) string {
	src := rand.NewSource(time.Now().UnixNano())
	b := make([]byte, n)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return string(b)
}
