package wsocketio

import (
	"github.com/kataras/iris/context"
)


func (s *Server) Handler() context.Handler {
	return func(ctx context.Context) {
		go s.Serve()
	}
}
