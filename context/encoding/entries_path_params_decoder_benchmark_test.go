package encoding

import (
	"reflect"
	"strings"
	"testing"

	"github.com/iris-contrib/schema"
)

func BenchmarkReadParams_SchemaDecodeVsKataras(b *testing.B) {
	params := PathParams{ // as provided by the mux handler, the keys always string as we speak about path parameters - they are converted to types on a later step.
		{
			Key:   "embedded_str_param1",
			Value: "embedded_str_param1_value",
		},
		{
			Key:   "embedded_str_param2",
			Value: "embedded_str_param2_value",
		},
		{ // This should not be part of the struct.
			Key:   "exported_str_param1",
			Value: "exported_str_param1_value",
		},
		{
			Key:   "str_param1",
			Value: "str_param1_value",
		},
		{
			Key:   "str_param2",
			Value: "str_param2_value",
		},
		{
			Key:   "int_param1",
			Value: 42,
		},
		{
			Key:   "float32_param1",
			Value: float32(46.2),
		},
		{
			Key:   "slice_param1",
			Value: "a/b/c",
		},
	}

	type ExportedEmbeddedStructParams struct {
		EmbeddedStrParam1 string `param:"embedded_str_param1"`
		EmbeddedStrParam2 string `param:"embedded_str_param2"`
	}

	type ExportedStructParams struct {
		ExportedStructParamsStrParam1 string `param:"exported_str_param1"`
	}

	type structuredParams struct {
		ExportedEmbeddedStructParams // Pointer to an embedded struct is not supported by the path decoder.

		// This can't be handled by any path decoder, just to show that the rest are not affected affected. Only embedded fields are supported.
		Exported ExportedStructParams `param:"-"`

		StrParam1     string   `param:"str_param1"`
		StrParam2     string   `param:"str_param2"`
		IntParam1     int      `param:"int_param1"`
		Float32Param1 float32  `param:"float32_param1"`
		SliceParam1   []string `param:"slice_param1"`
	}

	b.Run("schema.DecodeParams", func(b *testing.B) {
		testCount := b.N
		b.ResetTimer()
		for range testCount {
			values := make(map[string][]string, len(params))
			params.ForEachString(func(key string, value string) {
				// []string on path parameter, e.g.
				// /.../{tail:path}
				// Tail []string `param:"tail"`
				values[key] = strings.Split(value, "/")
			})

			var gotParams structuredParams
			err := schema.DecodeParams(values, &gotParams)
			if err != nil {
				b.Fatal(err)
			}

			if expected, got := "embedded_str_param1_value", gotParams.EmbeddedStrParam1; expected != got {
				b.Fatalf("expected: %q but got: %q", expected, got)
			}

			if expected, got := "embedded_str_param2_value", gotParams.EmbeddedStrParam2; expected != got {
				b.Fatalf("expected: %q but got: %q", expected, got)
			}

			if expected, got := "", gotParams.Exported.ExportedStructParamsStrParam1; expected != got {
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
	b.Run("kataras param decoder", func(b *testing.B) {
		decoder, err := NewPathParamsDecoder[structuredParams](EntriesDecoderOptions[any]{
			FieldTag:    DefaultPathParamTag,
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
			err := decoder.DecodeEntries(params, &gotParams)
			if err != nil {
				b.Fatal(err)
			}

			if expected, got := "embedded_str_param1_value", gotParams.EmbeddedStrParam1; expected != got {
				b.Fatalf("expected: %q but got: %q", expected, got)
			}

			if expected, got := "embedded_str_param2_value", gotParams.EmbeddedStrParam2; expected != got {
				b.Fatalf("expected: %q but got: %q", expected, got)
			}

			if expected, got := "", gotParams.Exported.ExportedStructParamsStrParam1; expected != got {
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
}

/*
$ go test -bench=BenchmarkReadParams_SchemaDecodeVsKataras$ -benchmem -benchtime=10s

BenchmarkReadParams_SchemaDecodeVsKataras/schema.DecodeParams-20                 2298720               6085 ns/op           2065 B/op          83 allocs/op
BenchmarkReadParams_SchemaDecodeVsKataras/kataras_param_decoder-20              32664153               357.1 ns/op          240 B/op           5 allocs/op

We can observe that kataras_param_decoder-20 is significantly more efficient than schema.DecodeParams-20. Here's a breakdown of the differences:

- Time Efficiency: The kataras_param_decoder-20 is approximately 17.03 times faster than schema.DecodeParams-20, with only 357.1 ns/op compared to 6085 ns/op.
- Memory Efficiency: kataras_param_decoder-20 also uses significantly less memory, with 240 B/op compared to 2065 B/op for schema.DecodeParams-20.
- Allocation Efficiency: There are far fewer allocations in kataras_param_decoder-20, with only 5 allocs/op compared to 83 allocs/op for schema.DecodeParams-20.

In summary, kataras_param_decoder-20 outperforms schema.DecodeParams-20 across all measured aspects in these benchmarks,
indicating it's a more optimized choice in terms of speed, memory usage, and allocations.
This makes kataras_param_decoder-20 a preferable option for performance-critical applications where efficiency is paramount.
*/
