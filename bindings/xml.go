package bindings

import (
	"encoding/xml"
	"io"
	"strings"

	"github.com/kataras/iris/context"
)

// BindXML reads XML from request's body
func BindXML(ctx context.IContext, xmlObject interface{}) error {
	data := ctx.GetRequestCtx().Request.Body()

	decoder := xml.NewDecoder(strings.NewReader(string(data)))
	err := decoder.Decode(xmlObject)
	//err != nil fix by @shiena
	if err != nil && err != io.EOF {
		return ErrReadBody.Format("XML", err.Error())
	}

	return nil
}
