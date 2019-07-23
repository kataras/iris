package router

import (
	"fmt"
)

func ExampleParty_StaticWeb() {
	api := NewAPIBuilder()
	api.StaticWeb("/static", "./assets")

	err := api.GetReport()
	if err == nil {
		panic("expected report for deprecation")
	}

	fmt.Print(err)
	// Output: StaticWeb is DEPRECATED and it will be removed eventually.
	// Source: ./deprecated_example_test.go:9
	// Use .HandleDir("/static", "./assets") instead.
}

func ExampleParty_StaticHandler() {
	api := NewAPIBuilder()
	api.StaticHandler("./assets", false, true)

	err := api.GetReport()
	if err == nil {
		panic("expected report for deprecation")
	}

	fmt.Print(err)
	// Output: StaticHandler is DEPRECATED and it will be removed eventually.
	// Source: ./deprecated_example_test.go:24
	// Use iris.FileServer("./assets", iris.DirOptions{ShowList: false, Gzip: true}) instead.
}

func ExampleParty_StaticEmbedded() {
	api := NewAPIBuilder()
	api.StaticEmbedded("/static", "./assets", nil, nil)

	err := api.GetReport()
	if err == nil {
		panic("expected report for deprecation")
	}

	fmt.Print(err)
	// Output: StaticEmbedded is DEPRECATED and it will be removed eventually.
	// It is also miss the AssetInfo bindata function, which is required now.
	// Source: ./deprecated_example_test.go:39
	// Use .HandleDir("/static", "./assets", iris.DirOptions{Asset: Asset, AssetInfo: AssetInfo, AssetNames: AssetNames}) instead.
}

func ExampleParty_StaticEmbeddedGzip() {
	api := NewAPIBuilder()
	api.StaticEmbeddedGzip("/static", "./assets", nil, nil)

	err := api.GetReport()
	if err == nil {
		panic("expected report for deprecation")
	}

	fmt.Print(err)
	// Output: StaticEmbeddedGzip is DEPRECATED and it will be removed eventually.
	// It is also miss the AssetInfo bindata function, which is required now.
	// Source: ./deprecated_example_test.go:55
	// Use .HandleDir("/static", "./assets", iris.DirOptions{Gzip: true, Asset: Asset, AssetInfo: AssetInfo, AssetNames: AssetNames}) instead.
}
