package apps

import (
	"fmt"
	"testing"

	"github.com/kataras/iris/v12"
)

func TestSwitchJoin(t *testing.T) {
	myapp := iris.New()
	customFilter := func(ctx iris.Context) bool {
		pass, _ := ctx.URLParamBool("filter")
		return pass
	}

	joinedCases := Join{
		SwitchCase{
			Filter: customFilter,
			App:    myapp,
		},
		Hosts{{Pattern: "^test.*$", Target: myapp}},
	}

	cases := []SwitchCase{
		{
			Filter: customFilter,
			App:    myapp,
		},
		{Filter: hostFilter("^test.*$"), App: myapp},
	}

	if expected, got := fmt.Sprintf("%#+v", cases), fmt.Sprintf("%#+v", joinedCases.GetSwitchCases()); expected != got {
		t.Fatalf("join does not match with the expected slice of cases, expected:\n%s\nbut got:\n%s", expected, got)
	}

}
