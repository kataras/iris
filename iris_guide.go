package iris

import (
	"time"

	"github.com/kataras/iris/v12/core/router"

	"github.com/kataras/iris/v12/middleware/cors"
	"github.com/kataras/iris/v12/middleware/modrevision"
	"github.com/kataras/iris/v12/middleware/recover"

	"github.com/kataras/iris/v12/x/errors"
)

// NewGuide returns a simple Iris API builder.
//
// Example Code:
/*
   package main

   import (
       "context"
       "database/sql"
       "time"

       "github.com/kataras/iris/v12"
       "github.com/kataras/iris/v12/x/errors"
   )

   func main() {
       iris.NewGuide().
           AllowOrigin("*").
           Compression(true).
           Health(true, "development", "kataras").
           Timeout(0, 20*time.Second, 20*time.Second).
           Middlewares().
           Services(
               // openDatabase(),
               // NewSQLRepoRegistry,
               NewMemRepoRegistry,
               NewTestService,
           ).
           API("/tests", new(TestAPI)).
           Listen(":80")
   }

   // Recommendation: move it to /api/tests/api.go file.
   type TestAPI struct {
       TestService *TestService
   }

   func (api *TestAPI) Configure(r iris.Party) {
       r.Get("/", api.listTests)
   }

   func (api *TestAPI) listTests(ctx iris.Context) {
       tests, err := api.TestService.ListTests(ctx)
       if err != nil {
           errors.Internal.LogErr(ctx, err)
           return
       }

       ctx.JSON(tests)
   }

   // Recommendation: move it to /pkg/storage/sql/db.go file.
   type DB struct {
       *sql.DB
   }

   func openDatabase( your database configuration... ) *DB {
       conn, err := sql.Open(...)
       // handle error.
       return &DB{DB: conn}
   }

   func (db *DB) Close() error {
       return nil
   }

   // Recommendation: move it to /pkg/repository/registry.go file.
   type RepoRegistry interface {
       Tests() TestRepository

       InTransaction(ctx context.Context, fn func(RepoRegistry) error) error
   }

   // Recommendation: move it to /pkg/repository/registry/memory.go file.
   type repoRegistryMem struct {
       tests TestRepository
   }

   func NewMemRepoRegistry() RepoRegistry {
       return &repoRegistryMem{
           tests: NewMemTestRepository(),
       }
   }

   func (r *repoRegistryMem) Tests() TestRepository {
       return r.tests
   }

   func (r *repoRegistryMem) InTransaction(ctx context.Context, fn func(RepoRegistry) error) error {
       return nil
   }

   // Recommendation: move it to /pkg/repository/registry/sql.go file.
   type repoRegistrySQL struct {
       db *DB

       tests TestRepository
   }

   func NewSQLRepoRegistry(db *DB) RepoRegistry {
       return &repoRegistrySQL{
           db:    db,
           tests: NewSQLTestRepository(db),
       }
   }

   func (r *repoRegistrySQL) Tests() TestRepository {
       return r.tests
   }

   func (r *repoRegistrySQL) InTransaction(ctx context.Context, fn func(RepoRegistry) error) error {
       return nil

       // your own database transaction code, may look something like that:
       // tx, err := r.db.BeginTx(ctx, nil)
       // if err != nil {
       //     return err
       // }
       // defer tx.Rollback()
       // newRegistry := NewSQLRepoRegistry(tx)
       // if err := fn(newRegistry);err!=nil{
       // 	return err
       // }
       // return tx.Commit()
   }

   // Recommendation: move it to /pkg/test/test.go
   type Test struct {
       Name string `db:"name"`
   }

   // Recommendation: move it to /pkg/test/repository.go
   type TestRepository interface {
       ListTests(ctx context.Context) ([]Test, error)
   }

   type testRepositoryMem struct {
       tests []Test
   }

   func NewMemTestRepository() TestRepository {
       list := []Test{
           {Name: "test1"},
           {Name: "test2"},
           {Name: "test3"},
       }

       return &testRepositoryMem{
           tests: list,
       }
   }

   func (r *testRepositoryMem) ListTests(ctx context.Context) ([]Test, error) {
       return r.tests, nil
   }

   type testRepositorySQL struct {
       db *DB
   }

   func NewSQLTestRepository(db *DB) TestRepository {
       return &testRepositorySQL{db: db}
   }

   func (r *testRepositorySQL) ListTests(ctx context.Context) ([]Test, error) {
       query := `SELECT * FROM tests ORDER BY created_at;`

       rows, err := r.db.QueryContext(ctx, query)
       if err != nil {
           return nil, err
       }
       defer rows.Close()

       tests := make([]Test, 0)
       for rows.Next() {
           var t Test
           if err := rows.Scan(&t.Name); err != nil {
               return nil, err
           }
           tests = append(tests, t)
       }

       if err := rows.Err(); err != nil {
           return nil, err
       }

       return tests, nil
   }

   // Recommendation: move it to /pkg/service/test_service.go file.
   type TestService struct {
       repos RepoRegistry
   }

   func NewTestService(registry RepoRegistry) *TestService {
       return &TestService{
           repos: registry,
       }
   }

   func (s *TestService) ListTests(ctx context.Context) ([]Test, error) {
       return s.repos.Tests().ListTests(ctx)
   }
*/
func NewGuide() Step1 {
	return &step1{}
}

type (
	Step1 interface {
		// AllowOrigin defines the CORS allowed domains.
		// Many can be splitted by comma.
		// If "*" is provided then all origins are accepted (use it for public APIs).
		AllowOrigin(originLine string) Step2
	}

	Step2 interface {
		// Compression enables or disables the gzip (or any other client-preferred) compression algorithm
		// for response writes.
		Compression(b bool) Step3
	}

	Step3 interface {
		// Health enables the /health route.
		// If "env" and "developer" are given, these fields will be populated to the client
		// through headers and environment on health route.
		Health(b bool, env, developer string) Step4
	}

	Step4 interface {
		// Timeout defines the http timeout, server read & write timeouts.
		Timeout(requestResponseLife, read time.Duration, write time.Duration) Step5
	}

	Step5 interface {
		// RouterMiddlewares registers one or more handlers to run before everything else.
		RouterMiddlewares(handlers ...Handler) Step5
		// Middlewares registers one or more handlers to run before the requested route's handler.
		Middlewares(handlers ...Handler) Step6
	}

	Step6 interface {
		// Services registers one or more dependencies that APIs can use.
		Services(deps ...interface{}) Step7
	}

	Step7 interface {
		// Handle registers a simple route on specific method and (dynamic) path.
		// It simply calls the Iris Application's Handle method.
		// Use the "API" method instead to keep the app organized.
		Handle(method, path string, handlers ...Handler) Step7
		// API registers a router which is responsible to serve the /api group.
		API(pathPrefix string, c ...router.PartyConfigurator) Step7
		// Build builds the application with the prior configuration and returns the
		// Iris Application instance for further customizations.
		//
		// Use "Build" before "Listen" or "Run" to apply further modifications
		// to the framework before starting the server. Calling "Build" is optional.
		Build() *Application // optional call.
		// Listen calls the Application's Listen method.
		// Use "Run" instead if you need to customize the HTTP/2 server itself.
		Listen(hostPort string, configurators ...Configurator) error // Listen OR Run.
		// Run calls the Application's Run method.
		Run(runner Runner, configurators ...Configurator) error
	}
)

type step1 struct {
	originLine string
}

func (s *step1) AllowOrigin(originLine string) Step2 {
	s.originLine = originLine
	return &step2{
		step1: s,
	}
}

type step2 struct {
	step1 *step1

	enableCompression bool
}

func (s *step2) Compression(b bool) Step3 {
	s.enableCompression = b
	return &step3{
		step2: s,
	}
}

type step3 struct {
	step2 *step2

	enableHealth   bool
	env, developer string
}

func (s *step3) Health(b bool, env, developer string) Step4 {
	s.enableHealth = b
	s.env, s.developer = env, developer
	return &step4{
		step3: s,
	}
}

type step4 struct {
	step3 *step3

	handlerTimeout time.Duration

	serverTimeoutRead  time.Duration
	serverTimeoutWrite time.Duration
}

func (s *step4) Timeout(requestResponseLife, read, write time.Duration) Step5 {
	s.handlerTimeout = requestResponseLife

	s.serverTimeoutRead = read
	s.serverTimeoutWrite = write
	return &step5{
		step4: s,
	}
}

type step5 struct {
	step4 *step4

	routerMiddlewares []Handler // top-level router middlewares, fire even on 404s.
	middlewares       []Handler
}

func (s *step5) RouterMiddlewares(handlers ...Handler) Step5 {
	s.routerMiddlewares = append(s.routerMiddlewares, handlers...)
	return s
}

func (s *step5) Middlewares(handlers ...Handler) Step6 {
	s.middlewares = handlers

	return &step6{
		step5: s,
	}
}

type step6 struct {
	step5 *step5

	deps []interface{}
	// derives from "deps".
	closers []func()
	// derives from "deps".
	configuratorsAsDeps []Configurator
}

func (s *step6) Services(deps ...interface{}) Step7 {
	s.deps = deps
	for _, d := range deps {
		if d == nil {
			continue
		}

		switch cb := d.(type) {
		case func():
			s.closers = append(s.closers, cb)
		case func() error:
			s.closers = append(s.closers, func() { cb() })
		case interface{ Close() }:
			s.closers = append(s.closers, cb.Close)
		case interface{ Close() error }:
			s.closers = append(s.closers, func() {
				cb.Close()
			})
		case Configurator:
			s.configuratorsAsDeps = append(s.configuratorsAsDeps, cb)
		}
	}

	return &step7{
		step6: s,
	}
}

type step7 struct {
	step6 *step6

	app *Application

	m        map[string][]router.PartyConfigurator
	handlers []step7SimpleRoute
}

type step7SimpleRoute struct {
	method, path string
	handlers     []Handler
}

func (s *step7) Handle(method, path string, handlers ...Handler) Step7 {
	s.handlers = append(s.handlers, step7SimpleRoute{method: method, path: path, handlers: handlers})
	return s
}

func (s *step7) API(prefix string, c ...router.PartyConfigurator) Step7 {
	if s.m == nil {
		s.m = make(map[string][]router.PartyConfigurator)
	}

	s.m[prefix] = append(s.m[prefix], c...)
	return s
}

func (s *step7) Build() *Application {
	if s.app != nil {
		return s.app
	}

	app := New()
	app.SetContextErrorHandler(errors.DefaultContextErrorHandler)
	app.Macros().SetErrorHandler(errors.DefaultPathParameterTypeErrorHandler)

	app.UseRouter(recover.New())
	app.UseRouter(s.step6.step5.routerMiddlewares...)
	app.UseRouter(func(ctx Context) {
		ctx.Header("Server", "Iris")
		if dev := s.step6.step5.step4.step3.developer; dev != "" {
			ctx.Header("X-Developer", dev)
		}

		ctx.Next()
	})

	if allowOrigin := s.step6.step5.step4.step3.step2.step1.originLine; allowOrigin != "" && allowOrigin != "none" {
		app.UseRouter(cors.New().AllowOrigin(allowOrigin).Handler())
	}

	if s.step6.step5.step4.step3.step2.enableCompression {
		app.Use(Compression)
	}

	for _, middleware := range s.step6.step5.middlewares {
		if middleware == nil {
			continue
		}

		app.Use(middleware)
	}

	if configAsDeps := s.step6.configuratorsAsDeps; len(configAsDeps) > 0 {
		app.Configure(configAsDeps...)
	}

	if s.step6.step5.step4.step3.enableHealth {
		app.Get("/health", modrevision.New(modrevision.Options{
			ServerName: "Iris Server",
			Env:        s.step6.step5.step4.step3.env,
			Developer:  s.step6.step5.step4.step3.developer,
		}))
	}

	if deps := s.step6.deps; len(deps) > 0 {
		app.EnsureStaticBindings().RegisterDependency(deps...)
	}

	for prefix, c := range s.m {
		app.PartyConfigure("/api"+prefix, c...)
	}

	for _, route := range s.handlers {
		app.Handle(route.method, route.path, route.handlers...)
	}

	if readTimeout := s.step6.step5.step4.serverTimeoutRead; readTimeout > 0 {
		app.ConfigureHost(func(su *Supervisor) {
			su.Server.ReadTimeout = readTimeout
			su.Server.IdleTimeout = readTimeout
			if v, recommended := readTimeout/4, 5*time.Second; v > recommended {
				su.Server.ReadHeaderTimeout = v
			} else {
				su.Server.ReadHeaderTimeout = recommended
			}
		})
	}

	if writeTimeout := s.step6.step5.step4.serverTimeoutWrite; writeTimeout > 0 {
		app.ConfigureHost(func(su *Supervisor) {
			su.Server.WriteTimeout = writeTimeout
		})
	}

	var defaultConfigurators = []Configurator{
		WithoutServerError(ErrServerClosed, ErrURLQuerySemicolon),
		WithOptimizations,
		WithRemoteAddrHeader(
			"X-Real-Ip",
			"X-Forwarded-For",
			"CF-Connecting-IP",
			"True-Client-Ip",
			"X-Appengine-Remote-Addr",
		),
		WithTimeout(s.step6.step5.step4.handlerTimeout),
	}
	app.Configure(defaultConfigurators...)

	s.app = app
	return app
}

func (s *step7) Listen(hostPort string, configurators ...Configurator) error {
	return s.Run(Addr(hostPort), configurators...)
}

func (s *step7) Run(runner Runner, configurators ...Configurator) error {
	app := s.Build()

	defer func() {
		// they will be called on interrupt signals too,
		// because Iris has a builtin mechanism to call server's shutdown on interrupt.
		for _, cb := range s.step6.closers {
			cb()
		}
	}()

	return app.Run(runner, configurators...)
}
