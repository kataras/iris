package iris

//  +------------------------------------------------------------+
//  | Bridge code between iris-cli and iris web application      |
//  | https://github.com/kataras/iris-cli                        |
//  +------------------------------------------------------------+

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kataras/iris/v12/context"

	"gopkg.in/yaml.v3"
)

// injectLiveReload tries to check if this application
// runs under https://github.com/kataras/iris-cli and if so
// then it checks if the livereload is enabled and then injects
// the watch listener (js script) on every HTML response.
// It has a slight performance cost but
// this (iris-cli with watch and livereload enabled)
// is meant to be used only in development mode.
// It does a full reload at the moment and if the port changed
// at runtime it will fire 404 instead of redirecting to the correct port (that's a TODO).
//
// tryInjectLiveReload runs right before Build -> BuildRouter.
func injectLiveReload(r Party) (bool, error) {
	conf := struct {
		Running    bool `yaml:"Running,omitempty"`
		LiveReload struct {
			Disable bool `yaml:"Disable"`
			Port    int  `yaml:"Port"`
		} `yaml:"LiveReload"`
	}{}
	// defaults to disabled here.
	conf.LiveReload.Disable = true

	wd, err := os.Getwd()
	if err != nil {
		return false, err
	}

	for _, path := range []string{".iris.yml" /*, "../.iris.yml", "../../.iris.yml" */} {
		path = filepath.Join(wd, path)

		if _, err := os.Stat(path); err == nil {
			inFile, err := os.OpenFile(path, os.O_RDONLY, 0600)
			if err != nil {
				return false, err
			}

			dec := yaml.NewDecoder(inFile)
			err = dec.Decode(&conf)
			inFile.Close()
			if err != nil {
				return false, err
			}

			break
		}
	}

	if !conf.Running || conf.LiveReload.Disable {
		return false, nil
	}

	scriptReloadJS := []byte(fmt.Sprintf(`<script>(function () {
    const scheme = document.location.protocol == "https:" ? "wss" : "ws";
    const endpoint = scheme + "://" + document.location.hostname + ":%d/livereload";

    w = new WebSocket(endpoint);
    w.onopen = function () {
        console.info("LiveReload: initialization");
    };
    w.onclose = function () {
        console.info("LiveReload: terminated");
    };
    w.onmessage = function (message) {
        // NOTE: full-reload, at least for the moment. Also if backend changed its port then we will get 404 here. 
        window.location.reload();
    };
}());</script>`, conf.LiveReload.Port))

	bodyCloseTag := []byte("</body>")

	r.UseRouter(func(ctx Context) {
		rec := ctx.Recorder() // Record everything and write all in once at the Context release.
		ctx.Next()            // call the next, so this is a 'done' handler.
		if strings.HasPrefix(ctx.GetContentType(), "text/html") {
			// delete(rec.Header(), context.ContentLengthHeaderKey)

			body := rec.Body()

			if idx := bytes.LastIndex(body, bodyCloseTag); idx > 0 {
				// add the script right before last </body>.
				body = append(body[:idx], bytes.Replace(body[idx:], bodyCloseTag, append(scriptReloadJS, bodyCloseTag...), 1)...)
				rec.SetBody(body)
			} else {
				// Just append it.
				rec.Write(scriptReloadJS) // nolint:errcheck
			}

			if _, has := rec.Header()[context.ContentLengthHeaderKey]; has {
				rec.Header().Set(context.ContentLengthHeaderKey, fmt.Sprintf("%d", len(rec.Body())))
			}
		}
	})
	return true, nil
}
