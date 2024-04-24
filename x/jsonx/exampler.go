package jsonx

// Exampler is an interface used by testing to generate examples.
type Exampler interface {
	ListExamples() any
}
