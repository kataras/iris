package formatter_test

import (
	. "github.com/yudai/gojsondiff/formatter"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/yudai/gojsondiff/tests"

	diff "github.com/yudai/gojsondiff"
)

var _ = Describe("Delta", func() {
	Describe("Foramt", func() {
		var (
			a, b map[string]interface{}
		)

		Context("There are no difference between the two JSON strings", func() {
			It("Returns empty JSON", func() {
				a = LoadFixture("../FIXTURES/base.json")
				b = LoadFixture("../FIXTURES/base.json")

				diff := diff.New().CompareObjects(a, b)

				f := NewDeltaFormatter()
				deltaJson, err := f.FormatAsJson(diff)
				Expect(err).To(BeNil())
				Expect(deltaJson).To(Equal(map[string]interface{}{}))
			})
		})

		Context("There are some values modified", func() {
			It("Detects changes", func() {
				a = LoadFixture("../FIXTURES/base.json")
				b = LoadFixture("../FIXTURES/base_changed.json")

				diff := diff.New().CompareObjects(a, b)

				f := NewDeltaFormatter()
				deltaJson, err := f.FormatAsJson(diff)
				Expect(err).To(BeNil())
				Expect(deltaJson).To(Equal(
					map[string]interface{}{
						"arr": map[string]interface{}{
							"_t": "a",
							"2": map[string]interface{}{
								"str": []interface{}{
									"pek3f", "changed",
								},
							},
							"3": map[string]interface{}{
								"_t": "a",
								"1": []interface{}{
									"1", "changed",
								},
							},
						},
						"null": []interface{}{nil, 0, 0},
						"obj": map[string]interface{}{
							"arr": map[string]interface{}{
								"_t": "a",
								"2": map[string]interface{}{
									"str": []interface{}{
										"eafeb", "changed",
									},
								},
							},
							"num": []interface{}{
								float64(19),
								0,
								0,
							},
							"obj": map[string]interface{}{
								"num": []interface{}{
									float64(14),
									float64(9999),
								},
								"str": []interface{}{
									"efj3",
									"changed",
								},
							},
							"new": []interface{}{
								"added",
							},
						},
					},
				))
			})
		})

		Context("There are long texts", func() {
			It("Returns empty JSON", func() {
				a = LoadFixture("../FIXTURES/long_text_from.json")
				b = LoadFixture("../FIXTURES/long_text_to.json")

				diff := diff.New().CompareObjects(a, b)

				f := NewDeltaFormatter()
				deltaJson, err := f.FormatAsJson(diff)
				Expect(err).To(BeNil())
				Expect(deltaJson).To(Equal(
					map[string]interface{}{
						"str": []interface{}{
							"@@ -27,14 +27,15 @@\n 40fj\n-q048hf\n+nafefea\n bgvz\n@@ -55,25 +55,20 @@\n m48q\n+a3\n 9p8qfh\n-qn4gbqqq4qp\n+nafe\n 94hq\n",
							0,
							2,
						},
					},
				))
			})
		})

	})
})
