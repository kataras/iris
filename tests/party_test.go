package tests

import (
	"testing"

	"github.com/gavv/httpexpect"
	"github.com/gavv/httpexpect/fasthttpexpect"
	"github.com/kataras/iris"
)

func TestSimpleParty(t *testing.T) {
	h := func(c *iris.Context) { c.WriteString(c.HostString() + c.PathString()) }

	/*
		// subdomain first, but this test will fail on your machine, so I just commend it, you can imagine what will be
		party2 := iris.Party("kataras.")
		{
			party2.Get("/", h)
			party2.Get("/path1", h)
			party2.Get("/path2", h)
			party2.Get("/namedpath/:param1/something/:param2", h)
			party2.Get("/namedpath/:param1/something/:param2/else", h)
		}*/

	// simple
	party1 := iris.Party("/party1")
	{
		party1.Get("/", h)
		party1.Get("/path1", h)
		party1.Get("/path2", h)
		party1.Get("/namedpath/:param1/something/:param2", h)
		party1.Get("/namedpath/:param1/something/:param2/else", h)
	}

	// create httpexpect instance that will call fasthtpp.RequestHandler directly
	e := httpexpect.WithConfig(httpexpect.Config{
		Reporter: httpexpect.NewAssertReporter(t),
		Client:   fasthttpexpect.NewBinder(iris.NoListen().Handler),
	})

	request := func(reqPath string) {
		e.Request("GET", reqPath).
			Expect().
			Status(iris.StatusOK).Body().Equal(reqPath)
	}

	// run the tests
	request("/party1/")
	request("/party1/path1")
	request("/party1/path2")
	request("/party1/namedpath/theparam1/something/theparam2")
	request("/party1/namedpath/theparam1/something/theparam2/else")
}
