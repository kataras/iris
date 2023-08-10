package apps

import (
	"fmt"
	"net/url"
	"testing"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/context"
	"github.com/kataras/iris/v12/httptest"
)

type testRequests map[string]map[string]int // url -> path -> status code

func TestSwitchHosts(t *testing.T) {
	var (
		expected = func(app context.Application, host string) string {
			return fmt.Sprintf("App Name: %s\nHost: %s", app, host)
		}
		index = func(ctx iris.Context) {
			ctx.WriteString(expected(ctx.Application(), ctx.Host()))
		}
	)

	testdomain1 := iris.New().SetName("test 1 domain")
	testdomain1.Get("/", index) // should match host matching with "testdomain1.com".

	testdomain2 := iris.New().SetName("test 2 domain")
	testdomain2.Get("/", index) // should match host matching with "testdomain2.com".

	mydomain := iris.New().SetName("my domain")
	mydomain.OnErrorCode(iris.StatusNotFound, func(ctx iris.Context) {
		ctx.WriteString(ctx.Host() + " custom not found")
	})
	mydomain.Get("/", index) // should match ALL hosts starting with "my".

	tests := []struct {
		Pattern  string
		Target   *iris.Application
		Requests testRequests
	}{
		{
			"testdomain1.com",
			testdomain1,
			testRequests{
				"http://testdomain1.com": {
					"/": iris.StatusOK,
				},
			},
		},
		{
			"testdomain2.com",
			testdomain2,
			testRequests{
				"http://testdomain2.com": {
					"/": iris.StatusOK,
				},
			},
		},
		{
			"^my.*$",
			mydomain,
			testRequests{
				"http://mydomain.com": {
					"/":   iris.StatusOK,
					"/nf": iris.StatusNotFound,
				},
				"http://myotherdomain.com": {
					"/": iris.StatusOK,
				},
				"http://mymy.com": {
					"/": iris.StatusOK,
				},
				"http://nmy.com": {
					"/": iris.StatusBadGateway, /* 404 hijacked by switch.OnErrorCode */
				},
			},
		},
	}

	var hosts Hosts
	for _, tt := range tests {
		hosts = append(hosts, Host{tt.Pattern, tt.Target})
	}
	switcher := Switch(hosts)
	switcher.OnErrorCode(iris.StatusNotFound, func(ctx iris.Context) {
		// inject the 404 to 502.
		// tests the ctx.Next inside the Hosts switch provider.
		ctx.StatusCode(iris.StatusBadGateway)
		ctx.WriteString("Switcher: Bad Gateway")
	})

	e := httptest.New(t, switcher)
	for i, tt := range tests {
		for URL, paths := range tt.Requests {
			u, err := url.Parse(URL)
			if err != nil {
				t.Fatalf("[%d] %v", i, err)
			}
			targetHost := u.Host
			for requestPath, statusCode := range paths {
				// url := fmt.Sprintf("http://%s", requestHost)
				body := expected(tt.Target, targetHost)
				switch statusCode {
				case 404:
					body = targetHost + " custom not found"
				case 502:
					body = "Switcher: Bad Gateway"
				}

				e.GET(requestPath).WithURL(URL).Expect().Status(statusCode).Body().IsEqual(body)
			}
		}
	}
}

func TestSwitchHostsRedirect(t *testing.T) {
	var (
		expected = func(appName, host, path string) string {
			return fmt.Sprintf("App Name: %s\nHost: %s\nPath: %s", appName, host, path)
		}
		index = func(ctx iris.Context) {
			ctx.WriteString(expected(ctx.Application().String(), ctx.Host(), ctx.Path()))
		}
	)

	mydomain := iris.New().SetName("mydomain")
	mydomain.OnAnyErrorCode(func(ctx iris.Context) {
		ctx.WriteString("custom: " + iris.StatusText(ctx.GetStatusCode()))
	})
	mydomain.Get("/", index)
	mydomain.Get("/f", index)

	tests := []struct {
		Pattern  string
		Target   string
		Requests testRequests
	}{
		{
			"www.mydomain.com",
			"mydomain",
			testRequests{
				"http://www.mydomain.com": {
					"/":   iris.StatusOK,
					"/f":  iris.StatusOK,
					"/nf": iris.StatusNotFound,
				},
			},
		},
		{
			"^test.*$",
			"mydomain",
			testRequests{
				"http://testdomain.com": {
					"/":   iris.StatusOK,
					"/f":  iris.StatusOK,
					"/nf": iris.StatusNotFound,
				},
			},
		},
		// Something like this will panic to protect users:
		// {
		// 	...,
		// 	"^my.*$",
		// 	"mydomain.com",
		// ...
		//
		{
			"^www.*$",
			"google.com",
			testRequests{
				"http://www.mydomain.com": {
					"/": iris.StatusOK,
				},
				"http://www.golang.org": {
					"/": iris.StatusNotFound, // should give not found because this is not a switcher's web app.
				},
			},
		},
	}

	var hostsRedirect Hosts
	for _, tt := range tests {
		hostsRedirect = append(hostsRedirect, Host{tt.Pattern, tt.Target})
	}

	switcher := Switch(hostsRedirect)
	e := httptest.New(t, switcher)

	for i, tt := range tests {
		for requestURL, paths := range tt.Requests {
			u, err := url.Parse(requestURL)
			if err != nil {
				t.Fatalf("[%d] %v", i, err)
			}
			targetHost := u.Host
			for requestPath, statusCode := range paths {
				body := expected(mydomain.String(), targetHost, requestPath)
				if statusCode != 200 {
					if tt.Target != mydomain.String() { // it's external.
						body = "Not Found"
					} else {
						body = "custom: " + iris.StatusText(statusCode)
					}
				}

				e.GET(requestPath).WithURL(requestURL).Expect().Status(statusCode).Body().IsEqual(body)
			}
		}
	}
}
