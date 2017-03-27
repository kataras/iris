package parser

import (
	"fmt"
	"strings"
	"testing"

	"gopkg.in/kataras/iris.v6/_future/ipel/lexer"
)

// Test is failing because we are not finished with the Parser yet
// 27/03
func TestParseError(t *testing.T) {
	// fail
	illegalChar := '$'

	input := "{id" + string(illegalChar) + "int range(1,5) else 404}"
	l := lexer.New(input)
	p := New(l)

	_, err := p.Parse()

	if err == nil {
		t.Fatalf("expecting not empty error on input '%s'", input)
	}

	// println(input[8:9])
	// println(input[13:17])

	illIdx := strings.IndexRune(input, illegalChar)
	expectedErr := fmt.Sprintf("[%d:%d] illegal token: %s", illIdx, illIdx, "$")
	if got := err.Error(); got != expectedErr {
		t.Fatalf("expecting error to be '%s' but got: %s", expectedErr, got)
	}
	//

	// success
	input2 := "{id:int range(1,5) else 404}"
	l2 := lexer.New(input2)
	p2 := New(l2)

	_, err = p2.Parse()

	if err != nil {
		t.Fatalf("expecting empty error on input '%s', but got: %s", input2, err.Error())
	}
	//
}
