package interpol

import "io"

// Options contains all options supported by an Interpolator.
type Options struct {
	Template io.Reader
	Format   Func
	Output   io.Writer
}

// Option is an option that can be applied to an Interpolator.
type Option func(OptionSetter)

// OptionSetter is an interface that contains the setters for all options
// supported by Interpolator.
type OptionSetter interface {
	SetTemplate(template io.Reader)
	SetFormat(format Func)
	SetOutput(output io.Writer)
}

// WithTemplate assigns Template to Options.
func WithTemplate(template io.Reader) Option {
	return func(setter OptionSetter) {
		setter.SetTemplate(template)
	}
}

// WithFormat assigns Format to Options.
func WithFormat(format Func) Option {
	return func(setter OptionSetter) {
		setter.SetFormat(format)
	}
}

// WithOutput assigns Output to Options.
func WithOutput(output io.Writer) Option {
	return func(setter OptionSetter) {
		setter.SetOutput(output)
	}
}

type optionSetter struct {
	opts *Options
}

func newOptionSetter(opts *Options) *optionSetter {
	return &optionSetter{opts: opts}
}

func (s *optionSetter) SetTemplate(template io.Reader) {
	s.opts.Template = template
}

func (s *optionSetter) SetFormat(format Func) {
	s.opts.Format = format
}

func (s *optionSetter) SetOutput(output io.Writer) {
	s.opts.Output = output
}

func setOptions(opts []Option, setter OptionSetter) {
	for _, opt := range opts {
		opt(setter)
	}
}
