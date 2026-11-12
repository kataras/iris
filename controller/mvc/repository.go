package mvc

import (
	stdContext "context"
	"reflect"

	"github.com/kataras/iris/v14/context"
	"github.com/kataras/iris/v14/controller/mvc/handler"
	"github.com/kataras/iris/v14/core/router"
	"github.com/kataras/iris/v14/rest/errors"
)

// Repository is the persistence half of a resource, the five methods every
// service's repository already has. Its context is the standard context: a
// repository never sees Iris.
//
// T is the model, ID its identifier and Filter the value the list route
// decodes from the request body.
type Repository[T any, ID context.IDType, Filter any] interface {
	// Create stores v and returns its new identifier.
	Create(ctx stdContext.Context, v T) (ID, error)
	// Get returns the stored T of the given id.
	Get(ctx stdContext.Context, id ID) (T, error)
	// Update stores v over the stored T of the given id, limited to
	// onlyColumns when the caller named any, and reports whether anything
	// changed.
	Update(ctx stdContext.Context, id ID, v T, onlyColumns ...string) (updated bool, err error)
	// Delete removes the stored T of the given id and reports whether
	// anything was removed.
	Delete(ctx stdContext.Context, id ID) (deleted bool, err error)
	// List returns a page of T matching the filter, and the total count of
	// the matches, ignoring limit and offset.
	List(ctx stdContext.Context, filter Filter, limit, offset int) ([]T, int64, error)
}

// UseRepository generates the six resource routes over repo:
//
//	POST   /             -> 201 {"id": ...}
//	GET    /{model_id}   -> 200 the model
//	PUT    /{model_id}   -> 204, or 304 when nothing changed
//	PATCH  /{model_id}   -> the same, honouring ?columns=a&columns=b
//	DELETE /{model_id}   -> 204, or 304 when nothing was deleted
//	POST   /list         -> 200 a page of models
//
// It is `UseController` without the controller: when a resource's HTTP half
// adds nothing to its repository, there is no type left to write.
//
// The path parameter name and its macro derive from T and ID exactly as they
// do for `UseController`, so a uuid.UUID id registers "{model_id:uuid}" and
// validates before any handler runs. Errors from the repository render
// through the rest/errors pipeline.
//
// The update routes read their columns through `Context.BindUpdate`, so a
// column that is not a member of T answers 400 naming it and the members it
// could have been. The names reaching the repository are still the client's,
// so a repository that builds SQL from them must map them to its own columns
// and never interpolate them.
//
//	app.Router("/invoices", mvc.UseRepository(invoiceRepository))
//
// Reach for a controller instead as soon as a route needs anything the
// repository does not do: a mail on create, a computed response, a second
// store to keep in sync.
func UseRepository[T any, ID context.IDType, Filter any](repo Repository[T, ID, Filter], opts ...Option) router.Controller {
	o := makeOptions(opts)

	param := newRawPathParam[T, ID]()
	if o.idParamName != "" {
		param = newNamedPathParam(o.idParamName, param.Type)
	}

	// readID reports false after it has already answered the client, exactly
	// like a ctx.Bind* helper. The conversion is required: the macro
	// evaluator stores the uuid form as a string, and a bare assertion on a
	// value the client controls would panic once per bad request.
	readID := func(ctx *context.Context) (ID, bool) {
		id, ok := handler.ConvertPathParamValue[ID](ctx.Params().Get(param.Name))
		if !ok {
			errors.HandleRequestError(ctx, errors.InvalidArgument)
			return id, false
		}

		return id, true
	}

	// beforeWrite runs the hooks over a decoded body and reports false after
	// it has answered the client, like a ctx.Bind* helper.
	beforeWrite := func(ctx *context.Context, v *T) bool {
		if err := o.runBeforeWrite(ctx, v); err != nil {
			errors.HandleRequestError(ctx, err)
			return false
		}

		return true
	}

	createHandler := func(ctx *context.Context) {
		v, ok := ctx.BindJSON[T]()
		if !ok {
			return
		}

		if !beforeWrite(ctx, &v) {
			return
		}

		id, err := repo.Create(ctx, v)
		ctx.Created(id, err)
	}

	readHandler := func(ctx *context.Context) {
		id, ok := readID(ctx)
		if !ok {
			return
		}

		v, err := repo.Get(ctx, id)
		ctx.OK(v, err)
	}

	updateHandler := func(ctx *context.Context) {
		id, ok := readID(ctx)
		if !ok {
			return
		}

		// BindUpdate is the one reader of the "columns" query key: it decodes
		// the body, parses both spellings (?columns=a&columns=b and
		// ?columns=a,b) and rejects a column that is not a member of T, so a
		// typo answers 400 instead of updating nothing and reporting success.
		v, columns, ok := ctx.BindUpdate[T]()
		if !ok {
			return
		}

		if !beforeWrite(ctx, &v) {
			return
		}

		updated, err := repo.Update(ctx, id, v, columns...)
		ctx.NoContent(updated, err)
	}

	deleteHandler := func(ctx *context.Context) {
		id, ok := readID(ctx)
		if !ok {
			return
		}

		deleted, err := repo.Delete(ctx, id)
		ctx.NoContent(deleted, err)
	}

	listHandler := func(ctx *context.Context) {
		ctx.Paginated(func(stdCtx stdContext.Context, pageOpts context.PageOptions, filter Filter) ([]T, int64, error) {
			return repo.List(stdCtx, filter, pageOpts.GetLimit(), pageOpts.GetOffset())
		}, o.pagination...)
	}

	return router.Routing(func(r router.Router) {
		o.checkBeforeWrite("UseRepository", r.GetRelPath(), reflect.TypeFor[*T]())

		r.UseOnce(errors.Handler)

		if !o.skip(Create) {
			r.Post("/", o.handlers(Create, createHandler)...)
		}

		if !o.skip(Get) {
			r.Get(param.RoutePath, o.handlers(Get, readHandler)...)
		}

		if !o.skip(Update) {
			r.Put(param.RoutePath, o.handlers(Update, updateHandler)...)
			r.Patch(param.RoutePath, o.handlers(Update, updateHandler)...)
		}

		if !o.skip(Delete) {
			r.Delete(param.RoutePath, o.handlers(Delete, deleteHandler)...)
		}

		if !o.skip(List) {
			r.Handle(router.MethodPost+" "+router.MethodQuery, "/list", o.handlers(List, listHandler)...)
		}
	})
}
