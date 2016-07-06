package iris

import "testing"

func TestContextReset(t *testing.T) {
	var context Context
	context.Params = PathParameters{PathParameter{Key: "testkey", Value: "testvalue"}}
	context.Reset(nil)
	if len(context.Params) > 0 {
		t.Fatalf("Expecting to have %d params but got: %d", 0, len(context.Params))
	}
}

func TestContextClone(t *testing.T) {
	var context Context
	context.Params = PathParameters{
		PathParameter{Key: "testkey", Value: "testvalue"},
		PathParameter{Key: "testkey2", Value: "testvalue2"},
	}
	c := context.Clone()
	if v := c.Param("testkey"); v != context.Param("testkey") {
		t.Fatalf("Expecting to have parameter value: %s but got: %s", context.Param("testkey"), v)
	}
	if v := c.Param("testkey2"); v != context.Param("testkey2") {
		t.Fatalf("Expecting to have parameter value: %s but got: %s", context.Param("testkey2"), v)
	}
}

func TestContextDoNextStop(t *testing.T) {
	var context Context
	ok := false
	afterStop := false
	context.middleware = Middleware{HandlerFunc(func(*Context) {
		ok = true
	}), HandlerFunc(func(*Context) {
		ok = true
	}), HandlerFunc(func(*Context) {
		// this will never execute
		afterStop = true
	})}
	context.Do()
	if context.pos != 0 {
		t.Fatalf("Expecting position 0 for context's middleware but we got: %d", context.pos)
	}
	if !ok {
		t.Fatalf("Unexpected behavior, first context's middleware didn't executed")
	}
	ok = false

	context.Next()

	if int(context.pos) != 1 {
		t.Fatalf("Expecting to have position %d but we got: %d", 1, context.pos)
	}
	if !ok {
		t.Fatalf("Next context's middleware didn't executed")
	}

	context.StopExecution()
	if context.pos != stopExecutionPosition {
		t.Fatalf("Context's StopExecution didn't worked, we expected to have position %d but we got %d", stopExecutionPosition, context.pos)
	}

	if !context.IsStopped() {
		t.Fatalf("Should be stopped")
	}

	context.Next()

	if afterStop {
		t.Fatalf("We stopped the execution but the next handler was executed")
	}
}

func TestContextParam(t *testing.T) {
	var context Context
	params := PathParameters{
		PathParameter{Key: "testkey", Value: "testvalue"},
		PathParameter{Key: "testkey2", Value: "testvalue2"},
		PathParameter{Key: "id", Value: "3"},
		PathParameter{Key: "bigint", Value: "548921854390354"},
	}
	context.Params = params

	if v := context.Param(params[0].Key); v != params[0].Value {
		t.Fatalf("Expecting parameter value to be %s but we got %s", params[0].Value, context.Param("testkey"))
	}
	if v := context.Param(params[1].Key); v != params[1].Value {
		t.Fatalf("Expecting parameter value to be %s but we got %s", params[1].Value, context.Param("testkey2"))
	}

	if len(context.Params) != len(params) {
		t.Fatalf("Expecting to have %d parameters but we got %d", len(params), len(context.Params))
	}

	if vi, err := context.ParamInt(params[2].Key); err != nil {
		t.Fatalf("Unexpecting error on context's ParamInt while trying to get the integer of the %s", params[2].Value)
	} else if vi != 3 {
		t.Fatalf("Expecting to receive %d but we got %d", 3, vi)
	}

	if vi, err := context.ParamInt64(params[3].Key); err != nil {
		t.Fatalf("Unexpecting error on context's ParamInt while trying to get the integer of the %s", params[2].Value)
	} else if vi != 548921854390354 {
		t.Fatalf("Expecting to receive %d but we got %d", 548921854390354, vi)
	}
}

func TestContextURLParam(t *testing.T) {

}
