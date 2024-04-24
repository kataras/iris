package main

import (
	"testing"

	"github.com/kataras/iris/v12/httptest"
)

func TestSubdomainsHTTPErrorsView(t *testing.T) {
	app := newApp()
	// hard coded.
	expectedHTMLResponse := `<html>
    <head>
    <title>Test Subdomain</title>
    
    </head>
    <body>
        
        <div style="background-color: black; color: red">
        <h1>Oups, you've got an error!</h1>
        
            
            <div style="background-color: white; color: red">
        <h1>Not Found</h1>
    </div>
    
        
    </div>
    
    </body>
    </html>`

	e := httptest.New(t, app)
	got := e.GET("/not_found").WithURL("http://test.mydomain.com").Expect().Status(httptest.StatusNotFound).
		ContentType("text/html", "utf-8").Body().Raw()

	if expected, _ := app.Minifier().String("text/html", expectedHTMLResponse); expected != got {
		t.Fatalf("expected:\n'%s'\nbut got:\n'%s'", expected, got)
	}
}
