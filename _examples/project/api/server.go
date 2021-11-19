package api

import (
	"context"
	"fmt"
	"time"

	"github.com/username/project/pkg/http/handlers"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/middleware/accesslog"

	"github.com/kataras/golog"
)

// Server is a wrapper of the main iris application and our project's custom configuration fields.
type Server struct {
	*iris.Application
	config Configuration

	// Here you can keep an instance of the database too.
	// db      *mydatabase_pkg.DB
	closers []func() // See `AddCloser` method.
}

// NewServer initializes a new HTTP/2 server.
// Use its Run/Listen methods to start it based on network options.
func NewServer(c Configuration) *Server {
	app := iris.New().SetName(c.ServerName)
	app.Configure(iris.WithConfiguration(c.Iris), iris.WithLowercaseRouting)

	srv := &Server{
		Application: app,
		config:      c,
	}

	if err := srv.prepare(); err != nil {
		srv.Logger().Fatal(err)
		return nil
	}

	return srv
}

func (srv *Server) prepare() error {
	// Here you can register the database instance
	// and prepare any project-relative fields.

	if srv.Logger().Level == golog.DebugLevel {
		srv.registerDebugFeatures()
	}

	srv.registerMiddlewares()
	srv.buildRouter()
	return nil
}

// registers application-level middlewares.
func (srv *Server) registerMiddlewares() {
	if srv.config.RequestLog != "" {
		srv.registerAccessLogger()
	}

	srv.UseRouter(handlers.CORS(srv.config.AllowOrigin))

	if srv.config.EnableCompression {
		srv.Use(iris.Compression)
	}
}

func (srv *Server) registerDebugFeatures() {}

func (srv *Server) registerAccessLogger() {
	// Initialize a new request access log middleware,
	// note that we use unbuffered data so we can have the results as fast as possible,
	// this has its cost use it only on debug.
	// Also, in the future see the iris example to
	// enable log rotation (date eand filesize-based files).
	ac := accesslog.FileUnbuffered(srv.config.RequestLog)

	// The default configuration:
	ac.Delim = '|'
	ac.TimeFormat = "2006-01-02 15:04:05"
	ac.Async = false
	ac.IP = true
	ac.BytesReceivedBody = true
	ac.BytesSentBody = true
	ac.BytesReceived = false
	ac.BytesSent = false
	ac.BodyMinify = false
	ac.RequestBody = true
	ac.ResponseBody = false
	ac.KeepMultiLineError = true
	ac.PanicLog = accesslog.LogHandler

	// Default line format if formatter is missing:
	// Time|Latency|Code|Method|Path|IP|Path Params Query Fields|Bytes Received|Bytes Sent|Request|Response|
	//
	// Set Custom Formatter:
	ac.SetFormatter(&accesslog.JSON{
		Indent:    "  ",
		HumanTime: true,
	})

	// ac.SetFormatter(&accesslog.CSV{})
	// ac.SetFormatter(&accesslog.Template{Text: "{{.Code}}"})

	srv.UseRouter(ac.Handler)
}

// Start runs the server on the TCP network address "0.0.0.0:port" which
// handles HTTP/1.1 & 2 requests on incoming connections.
func (srv *Server) Start() error {
	if srv.config.Domain != "" {
		srv.config.Port = 80 // not required but let's force-modify it.
		return srv.Application.Run(iris.AutoTLS(
			":443",
			srv.config.Domain,
			"kataras2006@hotmail.com",
		))
	}

	srv.ConfigureHost(func(su *iris.Supervisor) {
		// Set timeouts. More than enough, normally we use 20-30 seconds.
		su.Server.ReadTimeout = 5 * time.Minute
		su.Server.WriteTimeout = 5 * time.Minute
		su.Server.IdleTimeout = 10 * time.Minute
		su.Server.ReadHeaderTimeout = 2 * time.Minute
	})

	addr := fmt.Sprintf("%s:%d", srv.config.Host, srv.config.Port)
	return srv.Listen(addr)
}

// AddCloser adds one or more function that should be called on
// manual server shutdown or OS interrupt signals.
func (srv *Server) AddCloser(closers ...func()) {
	for _, closer := range closers {
		if closer == nil {
			continue
		}

		// Terminate any opened connections on OS interrupt signals.
		iris.RegisterOnInterrupt(closer)
	}

	srv.closers = append(srv.closers, closers...)
}

// Close gracefully terminates the HTTP server and calls the closers afterwards.
func (srv *Server) Close() error {
	ctx, cancelCtx := context.WithTimeout(context.Background(), 5*time.Second)
	err := srv.Shutdown(ctx)
	cancelCtx()

	for _, closer := range srv.closers {
		if closer == nil {
			continue
		}

		closer()
	}

	return err
}
