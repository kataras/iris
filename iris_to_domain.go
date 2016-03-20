package iris

/*
type IMemoryWriter domain.IMemoryWriter

// Handler the main Iris Handler interface.
type Handler interface {
	Serve(ctx *Context)
}

type internalHandlerImpl struct {
	handler Handler
}

func (i internalHandlerImpl) Serve(ctx domain.IContext) {
	i.handler.Serve(ctx.(*Context))
}

// HandlerFunc type is an adapter to allow the use of
// ordinary functions as HTTP handlers.  If f is a function
// with the appropriate signature, HandlerFunc(f) is a
// Handler that calls f.
type HandlerFunc func(*Context)

// Serve serves the handler, is like ServeHTTP for Iris
func (h HandlerFunc) Serve(ctx *Context) {
	h(ctx)
}

func toDomainHandler(h Handler) domain.Handler {
	return &internalHandlerImpl{h}
}

func toDomainHandlers(hs ...Handler) []domain.Handler {
	if hs == nil || len(hs) == 0 {
		return nil
	}
	hsLen := len(hs)
	converted := make([]domain.Handler, hsLen)
	for i := 0; i < hsLen; i++ {
		converted[i] = toDomainHandler(hs[i])
	}
	return converted
}

func toDomainHandlerFunc(f HandlerFunc) domain.HandlerFunc {
	return func(c domain.IContext) {
		f.Serve(c.(*Context))
	}
}

func toDomainHandlerFuncs(fcs ...HandlerFunc) []domain.HandlerFunc {
	if fcs == nil || len(fcs) == 0 {
		return nil
	}
	fcsLen := len(fcs)
	converted := make([]domain.HandlerFunc, fcsLen)
	for i := 0; i < fcsLen; i++ {
		converted[i] = toDomainHandlerFunc(fcs[i])
	}
	return converted
}*/
