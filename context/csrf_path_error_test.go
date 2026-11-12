package context_test

import (
	"fmt"
	"testing"

	"github.com/iris-contrib/schema"
	"github.com/kataras/iris/v14/context"
)

func TestIsErrPathCSRFTokenWrapped(t *testing.T) {
	csrf := schema.MultiError{context.CSRFTokenFormKey: schema.UnknownKeyError{Key: context.CSRFTokenFormKey}}

	if !context.IsErrPathCSRFToken(csrf) {
		t.Fatal("expected the plain schema.MultiError to be recognised")
	}

	if wrapped := fmt.Errorf("read form: %w", csrf); !context.IsErrPathCSRFToken(wrapped) {
		t.Fatalf("expected a wrapped schema.MultiError to be recognised: %v", wrapped)
	}

	other := schema.MultiError{"name": schema.UnknownKeyError{Key: "name"}}
	if context.IsErrPathCSRFToken(fmt.Errorf("read form: %w", other)) {
		t.Fatal("expected an unknown key other than the CSRF one not to be recognised")
	}
}
