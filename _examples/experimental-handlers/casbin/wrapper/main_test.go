package main

import (
	"testing"

	"github.com/iris-contrib/httpexpect"
	"github.com/kataras/iris/httptest"
)

func TestCasbinWrapper(t *testing.T) {
	app := newApp()
	e := httptest.New(t, app)

	type ttcasbin struct {
		username string
		path     string
		method   string
		status   int
	}

	tt := []ttcasbin{
		{"alice", "/dataset1/resource1", "GET", 200},
		{"alice", "/dataset1/resource1", "POST", 200},
		{"alice", "/dataset1/resource2", "GET", 200},
		{"alice", "/dataset1/resource2", "POST", 403},

		{"bob", "/dataset2/resource1", "GET", 200},
		{"bob", "/dataset2/resource1", "POST", 200},
		{"bob", "/dataset2/resource1", "DELETE", 200},
		{"bob", "/dataset2/resource2", "GET", 200},
		{"bob", "/dataset2/resource2", "POST", 403},
		{"bob", "/dataset2/resource2", "DELETE", 403},

		{"bob", "/dataset2/folder1/item1", "GET", 403},
		{"bob", "/dataset2/folder1/item1", "POST", 200},
		{"bob", "/dataset2/folder1/item1", "DELETE", 403},
		{"bob", "/dataset2/folder1/item2", "GET", 403},
		{"bob", "/dataset2/folder1/item2", "POST", 200},
		{"bob", "/dataset2/folder1/item2", "DELETE", 403},
	}

	for _, tt := range tt {
		check(e, tt.method, tt.path, tt.username, tt.status)
	}

	ttAdmin := []ttcasbin{
		{"cathrin", "/dataset1/item", "GET", 200},
		{"cathrin", "/dataset1/item", "POST", 200},
		{"cathrin", "/dataset1/item", "DELETE", 200},
		{"cathrin", "/dataset2/item", "GET", 403},
		{"cathrin", "/dataset2/item", "POST", 403},
		{"cathrin", "/dataset2/item", "DELETE", 403},
	}

	for _, tt := range ttAdmin {
		check(e, tt.method, tt.path, tt.username, tt.status)
	}

	Enforcer.DeleteRolesForUser("cathrin")

	ttAdminDeleted := []ttcasbin{
		{"cathrin", "/dataset1/item", "GET", 403},
		{"cathrin", "/dataset1/item", "POST", 403},
		{"cathrin", "/dataset1/item", "DELETE", 403},
		{"cathrin", "/dataset2/item", "GET", 403},
		{"cathrin", "/dataset2/item", "POST", 403},
		{"cathrin", "/dataset2/item", "DELETE", 403},
	}

	for _, tt := range ttAdminDeleted {
		check(e, tt.method, tt.path, tt.username, tt.status)
	}

}

func check(e *httpexpect.Expect, method, path, username string, status int) {
	e.Request(method, path).WithBasicAuth(username, "password").Expect().Status(status)
}
