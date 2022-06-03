package api

import (
	"context"
	"fmt"
	"time"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/middleware/cors"
	"github.com/kataras/iris/v12/x/errors"
)

type Server struct {
	*iris.Application

	config  Configuration
	closers []func() // See "AddCloser" method.
}

// NewServer returns a new server instance.
// See its "Start" method.
func NewServer(congig Configuration) *Server {
	app := iris.New()
	app.Configure(iris.WithConfiguration(congig.Iris))

	s := &Server{
		config:      congig,
		Application: app,
	}

	return s
}

func (s *Server) Build() error {
	if err := s.Application.Build(); err != nil {
		return err
	}

	// Register your 3rd-party drivers.
	// if err := s.registerDatabase(); err != nil {
	// 	return err
	// }s

	return s.configureRouter()
}

func (s *Server) configureRouter() error {
	s.SetContextErrorHandler(errors.DefaultContextErrorHandler)
	s.Macros().SetErrorHandler(errors.DefaultPathParameterTypeErrorHandler)

	s.registerMiddlewares()
	s.registerRoutes()

	return nil
}

func (s *Server) registerMiddlewares() {
	s.UseRouter(cors.New().AllowOrigin(s.config.AllowOrigin).Handler())

	s.UseRouter(func(ctx iris.Context) {
		ctx.Header("Server", "Iris")
		ctx.Header("X-Developer", "kataras")
		ctx.Next()
	})

	if s.config.EnableCompression {
		s.Use(iris.Compression) // .Use instead of .UseRouter to let it run only on registered routes and not on 404 errors and e.t.c.
	}
}

func (s *Server) registerRoutes() {
	// register your routes...

	s.Get("/health", func(ctx iris.Context) {
		ctx.WriteString("OK")
	})

}

// AddCloser adds one or more function that should be called on
// manual server shutdown or OS interrupt signals.
func (s *Server) AddCloser(closers ...func()) {
	for _, closer := range closers {
		if closer == nil {
			continue
		}

		// Terminate any opened connections on OS interrupt signals.
		iris.RegisterOnInterrupt(closer)
	}

	s.closers = append(s.closers, closers...)
}

// Shutdown gracefully terminates the HTTP server and calls the closers afterwards.
func (s *Server) Shutdown() error {
	ctx, cancelCtx := context.WithTimeout(context.Background(), 5*time.Second)
	err := s.Application.Shutdown(ctx)
	cancelCtx()

	for _, closer := range s.closers {
		if closer == nil {
			continue
		}

		closer()
	}

	return err
}

// Start starts the http server based on the config's host and port.
func (s *Server) Listen() error {
	if err := s.Build(); err != nil {
		return err
	}

	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
	return s.Application.Listen(addr)
}
