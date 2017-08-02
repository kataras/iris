package main

import (
	"fmt"
	"testing"

	"github.com/kataras/iris/httptest"
)

func TestI18n(t *testing.T) {
	app := newApp()

	expectedf := "From the language %s translated output: %s"
	var (
		elgr = fmt.Sprintf(expectedf, "el-GR", "γεια, iris")
		enus = fmt.Sprintf(expectedf, "en-US", "hello, iris")
		zhcn = fmt.Sprintf(expectedf, "zh-CN", "您好，iris")
	)

	e := httptest.New(t, app)
	// default is en-US
	e.GET("/").Expect().Status(httptest.StatusOK).Body().Equal(enus)
	// default is en-US if lang query unable to be found
	e.GET("/").WithQueryString("lang=un-EX").Expect().Status(httptest.StatusOK).Body().Equal(enus)

	e.GET("/").WithQueryString("lang=el-GR").Expect().Status(httptest.StatusOK).Body().Equal(elgr)
	e.GET("/").WithQueryString("lang=en-US").Expect().Status(httptest.StatusOK).Body().Equal(enus)
	e.GET("/").WithQueryString("lang=zh-CN").Expect().Status(httptest.StatusOK).Body().Equal(zhcn)
}
