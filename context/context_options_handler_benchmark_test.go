package context_test

/* TODO:
// go test -bench=BenchmarkContextOptionsHandler$ -benchmem
func BenchmarkContextOptionsHandler(b *testing.B) {
	getOptionsHandler := func(ctx iris.Context) {
		// reqOpts := ctx.RequestOptions()
		respOpts := ctx.ResponseOptions()
		// fmt.Printf("%#+v, is nil = %v\n", respOpts, respOpts == nil)
		// ctx.Writef("%v, %s", reqOpts.GetEnableCompression(), respOpts.GetCharset())
		// ctx.Writef("%v", reqOpts.GetEnableCompression())
		ctx.Writef("%s", respOpts.GetCharset())

		// opts := ctx.Options(false)
		// ctx.Writef("%s", opts.GetCharset())
	}

	// setOptionsHandler := func(ctx iris.Context) {
	// 	respOpts := ctx.
	// }

	app := iris.New()

	app.Get("/", getOptionsHandler)

	if err := app.Build(); err != nil {
		b.Fatal(err)
	}

	newHTTPRequest := func() *http.Request {
		return httptest.NewRequest("GET", "/", nil)
	}

	request := newHTTPRequest()

	// b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		recorder := httptest.NewRecorder()
		app.ServeHTTP(recorder, request)
		if body := recorder.Body.String(); body != "utf-8" {
			b.Fatalf("unexpected body: %s", body)
		}
	}
}

// if body := recorder.Body.String(); body != "false" {
// 	b.Fatalf("unexpected body: %s", body)
// }
// recorder.Body.Reset()
*/
