package bindings

import (
	"encoding/json"

	"io"
	"strings"

	"github.com/kataras/iris/context"
)

// BindJSON reads JSON from request's body
func BindJSON(ctx context.IContext, jsonObject interface{}) error {
	data := ctx.GetRequestCtx().Request.Body()

	decoder := json.NewDecoder(strings.NewReader(string(data)))
	err := decoder.Decode(jsonObject)

	//err != nil fix by @shiena
	if err != nil && err != io.EOF {
		return ErrReadBody.Format("JSON", err.Error())
	}

	return nil
}
