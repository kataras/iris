package encoding

import (
	"net/url"
	"reflect"
	"testing"

	"github.com/iris-contrib/schema"
)

func BenchmarkReadQuery_SchemaDecodeVsKataras(b *testing.B) {
	values := url.Values{
		"embedded_str_param1": {"embedded_str_param1_value"},
		"embedded_str_param2": {"embedded_str_param2_value"},
		"str_param1":          {"str_param1_value"},
		"str_param2":          {"str_param2_value"},
		"int_param1":          {"42"},
		"float32_param1":      {"46.2"},
		"slice_param1":        {"a", "b", "c"},
	}

	type ExportedEmbeddedStructParams struct {
		EmbeddedStrParam1 string `url:"embedded_str_param1"`
		EmbeddedStrParam2 string `url:"embedded_str_param2"`
	}

	type structuredParams struct {
		ExportedEmbeddedStructParams

		StrParam1     string   `url:"str_param1"`
		StrParam2     string   `url:"str_param2"`
		IntParam1     int      `url:"int_param1"`
		Float32Param1 float32  `url:"float32_param1"`
		SliceParam1   []string `url:"slice_param1"`
	}

	b.Run("schema.DecodeQuery", func(b *testing.B) {
		testCount := b.N
		b.ResetTimer()
		for range testCount {
			var gotParams structuredParams
			err := schema.DecodeQuery(values, &gotParams)
			if err != nil {
				b.Fatal(err)
			}

			if expected, got := "embedded_str_param1_value", gotParams.EmbeddedStrParam1; expected != got {
				b.Fatalf("expected: %q but got: %q", expected, got)
			}

			if expected, got := "embedded_str_param2_value", gotParams.EmbeddedStrParam2; expected != got {
				b.Fatalf("expected: %q but got: %q", expected, got)
			}

			if expected, got := "str_param1_value", gotParams.StrParam1; expected != got {
				b.Fatalf("expected: %q but got: %q", expected, got)
			}

			if expected, got := "str_param2_value", gotParams.StrParam2; expected != got {
				b.Fatalf("expected: %q but got: %q", expected, got)
			}

			if expected, got := 42, gotParams.IntParam1; expected != got {
				b.Fatalf("expected: %d but got: %d", expected, got)
			}

			if expected, got := float32(46.2), gotParams.Float32Param1; expected != got {
				b.Fatalf("expected: %f but got: %f", expected, got)
			}

			if expected, got := []string{"a", "b", "c"}, gotParams.SliceParam1; !reflect.DeepEqual(expected, got) {
				b.Fatalf("expected: %v but got: %v", expected, got)
			}
		}
	})

	b.Run("kataras query decoder", func(b *testing.B) {
		decoder, err := NewQueryDecoder[structuredParams](KeyValuesDecoderOptions{
			FieldTag:    DefaultQueryTag,
			Strict:      false,
			FieldSetter: nil,
		})
		if err != nil {
			b.Fatal(err)
		}

		testCount := b.N
		b.ResetTimer()
		for range testCount {
			var gotParams structuredParams
			err := decoder.DecodeKeyValues(values, &gotParams)
			if err != nil {
				b.Fatal(err)
			}

			if expected, got := "embedded_str_param1_value", gotParams.EmbeddedStrParam1; expected != got {
				b.Fatalf("expected: %q but got: %q", expected, got)
			}

			if expected, got := "embedded_str_param2_value", gotParams.EmbeddedStrParam2; expected != got {
				b.Fatalf("expected: %q but got: %q", expected, got)
			}

			if expected, got := "str_param1_value", gotParams.StrParam1; expected != got {
				b.Fatalf("expected: %q but got: %q", expected, got)
			}

			if expected, got := "str_param2_value", gotParams.StrParam2; expected != got {
				b.Fatalf("expected: %q but got: %q", expected, got)
			}

			if expected, got := 42, gotParams.IntParam1; expected != got {
				b.Fatalf("expected: %d but got: %d", expected, got)
			}

			if expected, got := float32(46.2), gotParams.Float32Param1; expected != got {
				b.Fatalf("expected: %f but got: %f", expected, got)
			}

			if expected, got := []string{"a", "b", "c"}, gotParams.SliceParam1; !reflect.DeepEqual(expected, got) {
				b.Fatalf("expected: %v but got: %v", expected, got)
			}
		}
	})

	/*
		$ go test -bench=BenchmarkReadQuery_SchemaDecodeVsKataras$ -benchmem

		BenchmarkReadQuery_SchemaDecodeVsKataras/schema.DecodeQuery-20            248250              5281 ns/op            2352 B/op         89 allocs/op
		BenchmarkReadQuery_SchemaDecodeVsKataras/kataras_query_decoder-20        2421475              504.0 ns/op           208 B/op          4 allocs/op

		- kataras_query_decoder-20 is approximately 10.48 times faster.
		- kataras_query_decoder-20 allocates about 11.31 times less memory.
		- kataras_query_decoder-20 has 22.25 times fewer allocations.

		In summary, kataras_query_decoder-20 is significantly faster, more efficient in memory usage, and has fewer allocations compared to schema.DecodeQuery-20.
		These improvements are substantial and would likely lead to better performance in practical applications.
	*/
}
