package main_test

import (
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/httptest"
	"github.com/kataras/iris/v12/middleware/basicauth"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Ginkgotest", func() {
	var (
		e              *httptest.Expect
		app            *iris.Application
		opts           basicauth.Options
		authentication iris.Handler // or just: basicauth.Default(map...)
	)

	BeforeEach(func() {
		opts = basicauth.Options{
			Allow: basicauth.AllowUsers(map[string]string{"myusername": "mypassword"}),
		}
		authentication = basicauth.New(opts)
		app = newApp(authentication)
		e = httptest.New(GinkgoT(), app, httptest.Strict(true))
	})

	When("no basic auth", Ordered, func() {
		It("redirects to /admin without basic auth", func() {
			response := e.GET("/").Expect().Raw()
			Expect(httptest.StatusUnauthorized).To(Equal(response.StatusCode))
		})
		It("without basic auth", func() {
			// without basic auth
			response := e.GET("/").Expect().Raw()
			Expect(httptest.StatusUnauthorized).To(Equal(response.StatusCode))
		})

	})

	When("valid basic auth", func() {
		It("with basic auth /admin", func() {
			expect := e.GET("/admin").WithBasicAuth("myusername", "mypassword").Expect()
			Expect(httptest.StatusOK).To(Equal(expect.Raw().StatusCode))
			Expect("/admin myusername:mypassword").To(Equal(expect.Body().Raw()))
		})
		It("with basic auth /admin/profile", func() {
			expect := e.GET("/admin/profile").WithBasicAuth("myusername", "mypassword").Expect()
			Expect(httptest.StatusOK).To(Equal(expect.Raw().StatusCode))
			Expect("/admin/profile myusername:mypassword").To(Equal(expect.Body().Raw()))
		})
		It("with basic auth /admin/profile", func() {
			expect := e.GET("/admin/settings").WithBasicAuth("myusername", "mypassword").Expect()
			Expect(httptest.StatusOK).To(Equal(expect.Raw().StatusCode))
			Expect("/admin/settings myusername:mypassword").To(Equal(expect.Body().Raw()))
		})
	})

	When("invalid basic auth", func() {
		It("invalid basic auth /admin/settings", func() {
			expect := e.GET("/admin/settings").WithBasicAuth("invalidusername", "invalidpassword").Expect()
			Expect(httptest.StatusUnauthorized).To(Equal(expect.Raw().StatusCode))
		})
	})
})
