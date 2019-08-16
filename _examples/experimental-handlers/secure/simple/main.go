package main

import (
	"github.com/kataras/iris"

	"github.com/iris-contrib/middleware/secure"
)

func main() {
	s := secure.New(secure.Options{
		AllowedHosts:            []string{"ssl.example.com"},                                                                                                                         // AllowedHosts is a list of fully qualified domain names that are allowed. Default is empty list, which allows any and all host names.
		SSLRedirect:             true,                                                                                                                                                // If SSLRedirect is set to true, then only allow HTTPS requests. Default is false.
		SSLTemporaryRedirect:    false,                                                                                                                                               // If SSLTemporaryRedirect is true, the a 302 will be used while redirecting. Default is false (301).
		SSLHost:                 "ssl.example.com",                                                                                                                                   // SSLHost is the host name that is used to redirect HTTP requests to HTTPS. Default is "", which indicates to use the same host.
		SSLProxyHeaders:         map[string]string{"X-Forwarded-Proto": "https"},                                                                                                     // SSLProxyHeaders is set of header keys with associated values that would indicate a valid HTTPS request. Useful when using Nginx: `map[string]string{"X-Forwarded-Proto": "https"}`. Default is blank map.
		STSSeconds:              315360000,                                                                                                                                           // STSSeconds is the max-age of the Strict-Transport-Security header. Default is 0, which would NOT include the header.
		STSIncludeSubdomains:    true,                                                                                                                                                // If STSIncludeSubdomains is set to true, the `includeSubdomains` will be appended to the Strict-Transport-Security header. Default is false.
		STSPreload:              true,                                                                                                                                                // If STSPreload is set to true, the `preload` flag will be appended to the Strict-Transport-Security header. Default is false.
		ForceSTSHeader:          false,                                                                                                                                               // STS header is only included when the connection is HTTPS. If you want to force it to always be added, set to true. `IsDevelopment` still overrides this. Default is false.
		FrameDeny:               true,                                                                                                                                                // If FrameDeny is set to true, adds the X-Frame-Options header with the value of `DENY`. Default is false.
		CustomFrameOptionsValue: "SAMEORIGIN",                                                                                                                                        // CustomFrameOptionsValue allows the X-Frame-Options header value to be set with a custom value. This overrides the FrameDeny option.
		ContentTypeNosniff:      true,                                                                                                                                                // If ContentTypeNosniff is true, adds the X-Content-Type-Options header with the value `nosniff`. Default is false.
		BrowserXSSFilter:        true,                                                                                                                                                // If BrowserXssFilter is true, adds the X-XSS-Protection header with the value `1; mode=block`. Default is false.
		ContentSecurityPolicy:   "default-src 'self'",                                                                                                                                // ContentSecurityPolicy allows the Content-Security-Policy header value to be set with a custom value. Default is "".
		PublicKey:               `pin-sha256="base64+primary=="; pin-sha256="base64+backup=="; max-age=5184000; includeSubdomains; report-uri="https://www.example.com/hpkp-report"`, // PublicKey implements HPKP to prevent MITM attacks with forged certificates. Default is "".

		IsDevelopment: true, // This will cause the AllowedHosts, SSLRedirect, and STSSeconds/STSIncludeSubdomains options to be ignored during development. When deploying to production, be sure to set this to false.
	})

	app := iris.New()
	app.Use(s.Serve)

	app.Get("/home", func(ctx iris.Context) {
		ctx.Writef("Hello from /home")
	})

	app.Run(iris.Addr(":8080"))
}
