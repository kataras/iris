```go
app.PartyConfigure("/api", errors.NewParty[CreateRequest, CreateResponse, ListFilter]().
	Create(service.Create).
	Update(service.Update).
	Delete(service.DeleteWithFeedback).
	List(service.ListPaginated).
	Get(service.GetByID).Validation(validateCreateRequest))
```

```go
type Party[T, R, F any] struct {
	validators         []ContextRequestFunc[T]
	filterValidators   []ContextRequestFunc[F]
	filterIntercepters []ContextResponseFunc[F, R]
	intercepters       []ContextResponseFunc[T, R]

	serviceCreateFunc func(stdContext.Context, T) (R, error)
	serviceUpdateFunc func(stdContext.Context, T) (bool, error)
	serviceDeleteFunc func(stdContext.Context, string) (bool, error)
	serviceListFunc   func(stdContext.Context, pagination.ListOptions, F /* filter options */) ([]R, int, error)
	serviceGetFunc    func(stdContext.Context, string) (R, error)
}

func (p *Party[T, R, F]) Configure(r router.Party) {
	if p.serviceCreateFunc != nil {
		r.Post("/", Validation(p.validators...), Intercept(p.intercepters...), CreateHandler(p.serviceCreateFunc))
	}

	if p.serviceUpdateFunc != nil {
		r.Put("/{id:string}", Validation(p.validators...), Intercept(p.intercepters...), NoContentOrNotModifiedHandler(p.serviceUpdateFunc))
	}

	if p.serviceListFunc != nil {
		r.Post("/list", Validation(p.filterValidators...), Intercept(p.filterIntercepters...), ListHandler(p.serviceListFunc))
	}

	if p.serviceDeleteFunc != nil {
		r.Delete("/{id:string}", NoContentOrNotModifiedHandler(p.serviceDeleteFunc, PathParam[string]("id")))
	}

	if p.serviceGetFunc != nil {
		r.Get("/{id:string}", Handler(p.serviceGetFunc, PathParam[string]("id")))
	}
}

func NewParty[T, R, F any]() *Party[T, R, F] {
	return &Party[T, R, F]{}
}

func (p *Party[T, R, F]) Validation(validators ...ContextRequestFunc[T]) *Party[T, R, F] {
	p.validators = append(p.validators, validators...)
	return p
}

func (p *Party[T, R, F]) FilterValidation(filterValidators ...ContextRequestFunc[F]) *Party[T, R, F] {
	p.filterValidators = append(p.filterValidators, filterValidators...)
	return p
}

func (p *Party[T, R, F]) Intercept(intercepters ...ContextResponseFunc[T, R]) *Party[T, R, F] {
	p.intercepters = append(p.intercepters, intercepters...)
	return p
}

func (p *Party[T, R, F]) FilterIntercept(filterIntercepters ...ContextResponseFunc[F, R]) *Party[T, R, F] {
	p.filterIntercepters = append(p.filterIntercepters, filterIntercepters...)
	return p
}

func (p *Party[T, R, F]) Create(fn func(stdContext.Context, T) (R, error)) *Party[T, R, F] {
	p.serviceCreateFunc = fn
	return p
}

func (p *Party[T, R, F]) Update(fn func(stdContext.Context, T) (bool, error)) *Party[T, R, F] {
	p.serviceUpdateFunc = fn
	return p
}

func (p *Party[T, R, F]) Delete(fn func(stdContext.Context, string) (bool, error)) *Party[T, R, F] {
	p.serviceDeleteFunc = fn
	return p
}

func (p *Party[T, R, F]) List(fn func(stdContext.Context, pagination.ListOptions, F /* filter options */) ([]R, int, error)) *Party[T, R, F] {
	p.serviceListFunc = fn
	return p
}

func (p *Party[T, R, F]) Get(fn func(stdContext.Context, string) (R, error)) *Party[T, R, F] {
	p.serviceGetFunc = fn
	return p
}

```
