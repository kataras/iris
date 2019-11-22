package main

import (
	"fmt"
	"strings"
	"testing"

	"github.com/kataras/iris/v12/httptest"
)

func TestI18n(t *testing.T) {
	app := newApp()

	const (
		expectedf = "From the language %s translated output: %s"

		enUS = "hello, iris"
		elGR = "γεια, iris"
		zhCN = "您好，iris"
	)

	var (
		tests = map[string]string{
			"en-US": fmt.Sprintf(expectedf, "en-US", enUS),
			"el-GR": fmt.Sprintf(expectedf, "el-GR", elGR),
			"zh-CN": fmt.Sprintf(expectedf, "zh-CN", zhCN),
		}

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
	// default should be en-US.
	e.GET("/").Expect().Status(httptest.StatusOK).Body().Equal(tests["en-US"])

	for lang, body := range tests {
		e.GET("/").WithQueryString("lang=" + lang).Expect().Status(httptest.StatusOK).
			Body().Equal(body)

		// test lowercase.
		e.GET("/").WithQueryString("lang=" + strings.ToLower(lang)).Expect().Status(httptest.StatusOK).
			Body().Equal(body)

		// test first part (e.g. en instead of en-US).
		langFirstPart := strings.Split(lang, "-")[0]
		e.GET("/").WithQueryString("lang=" + langFirstPart).Expect().Status(httptest.StatusOK).
			Body().Equal(body)

		// test accept-language header prefix (i18n wrapper).
		e.GET("/"+lang).WithHeader("Accept-Language", lang).Expect().Status(httptest.StatusOK).
			Body().Equal(body)

		// test path prefix (i18n router wrapper).
		e.GET("/" + lang).Expect().Status(httptest.StatusOK).
			Body().Equal(body)

		// test path prefix with first part.
		e.GET("/" + langFirstPart).Expect().Status(httptest.StatusOK).
			Body().Equal(body)
	}

	e.GET("/multi").WithQueryString("lang=el-GR").Expect().Status(httptest.StatusOK).
		Body().Equal(elgrMulti)
	e.GET("/multi").WithQueryString("lang=en-US").Expect().Status(httptest.StatusOK).
		Body().Equal(enusMulti)

	// test path prefix (i18n router wrapper).
	e.GET("/el-gr/multi").Expect().Status(httptest.StatusOK).
		Body().Equal(elgrMulti)
	e.GET("/en/multi").Expect().Status(httptest.StatusOK).
		Body().Equal(enusMulti)

	e.GET("/el-GRtemplates").Expect().Status(httptest.StatusNotFound)
	e.GET("/el-templates").Expect().Status(httptest.StatusNotFound)

	e.GET("/el/templates").Expect().Status(httptest.StatusOK).Body().Contains(elGR).Contains(zhCN)
}
