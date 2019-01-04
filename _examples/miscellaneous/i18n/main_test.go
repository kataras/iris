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

		elgrMulti = fmt.Sprintf("From the language: %s, translated output:\n%s=%s\n%s=%s", "el-GR",
			"key1",
			"αυτό είναι μια τιμή από το πρώτο αρχείο: locale_multi_first",
			"key2",
			"αυτό είναι μια τιμή από το δεύτερο αρχείο μετάφρασης: locale_multi_second")
		enusMulti = fmt.Sprintf("From the language: %s, translated output:\n%s=%s\n%s=%s", "en-US",
			"key1",
			"this is a value from the first file: locale_multi_first",
			"key2",
			"this is a value from the second file: locale_multi_second")
	)

	e := httptest.New(t, app)
	// default is en-US
	e.GET("/").Expect().Status(httptest.StatusOK).Body().Equal(enus)
	// default is en-US if lang query unable to be found
	e.GET("/").Expect().Status(httptest.StatusOK).Body().Equal(enus)

	e.GET("/").WithQueryString("lang=el-GR").Expect().Status(httptest.StatusOK).
		Body().Equal(elgr)
	e.GET("/").WithQueryString("lang=en-US").Expect().Status(httptest.StatusOK).
		Body().Equal(enus)
	e.GET("/").WithQueryString("lang=zh-CN").Expect().Status(httptest.StatusOK).
		Body().Equal(zhcn)

	e.GET("/multi").WithQueryString("lang=el-GR").Expect().Status(httptest.StatusOK).
		Body().Equal(elgrMulti)
	e.GET("/multi").WithQueryString("lang=en-US").Expect().Status(httptest.StatusOK).
		Body().Equal(enusMulti)

}
