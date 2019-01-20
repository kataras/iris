// Package interpol provides utility functions for doing format-string like
// string interpolation using named parameters.
// Currently, a template only accepts variable placeholders delimited by brace
// characters (eg. "Hello {foo} {bar}").
package interpol

import (
	"bytes"
	"errors"
	"io"
	"strings"
)

// Errors returned when formatting templates.
var (
	ErrUnexpectedClose = errors.New("interpol: unexpected close in template")
	ErrExpectingClose  = errors.New("interpol: expecting close in template")
	ErrKeyNotFound     = errors.New("interpol: key not found")
	ErrReadByteFailed  = errors.New("interpol: read byte failed")
)

// Func receives the placeholder key and writes to the io.Writer. If an error
// happens, the function can return an error, in which case the interpolation
// will be aborted.
type Func func(key string, w io.Writer) error

// New creates a new interpolator with the given list of options.
// You can use options such as the ones returned by WithTemplate, WithFormat
// and WithOutput.
func New(opts ...Option) *Interpolator {
	opts2 := &Options{}
	setOptions(opts, newOptionSetter(opts2))
	return NewWithOptions(opts2)
}

// NewWithOptions creates a new interpolator with the given options.
func NewWithOptions(opts *Options) *Interpolator {
	return &Interpolator{
		template: templateReader(opts),
		output:   outputWriter(opts),
		format:   opts.Format,
		rb:       make([]rune, 0, 64),
		start:    -1,
		closing:  false,
	}
}

// Interpolator interpolates Template to Output, according to Format.
type Interpolator struct {
	template io.RuneReader
	output   runeWriter
	format   Func
	rb       []rune
	start    int
	closing  bool
}

// Interpolate reads runes from Template and writes them to Output, with the
// exception of placeholders which are passed to Format.
func (i *Interpolator) Interpolate() error {
	for pos := 0; ; pos++ {
		r, _, err := i.template.ReadRune()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		if err := i.parse(r, pos); err != nil {
			return err
		}
	}
	return i.finish()
}

func (i *Interpolator) parse(r rune, pos int) error {
	switch r {
	case '{':
		return i.open(pos)
	case '}':
		return i.close()
	default:
		return i.append(r)
	}
}

func (i *Interpolator) open(pos int) error {
	if i.closing {
		return ErrUnexpectedClose
	}
	if i.start >= 0 {
		if _, err := i.output.WriteRune('{'); err != nil {
			return err
		}
		i.start = -1
	} else {
		i.start = pos + 1
	}
	return nil
}

func (i *Interpolator) close() error {
	if i.start >= 0 {
		if err := i.format(string(i.rb), i.output); err != nil {
			return err
		}
		i.rb = i.rb[:0]
		i.start = -1
	} else if i.closing {
		i.closing = false
		if _, err := i.output.WriteRune('}'); err != nil {
			return err
		}
	} else {
		i.closing = true
	}
	return nil
}

func (i *Interpolator) append(r rune) error {
	if i.closing {
		return ErrUnexpectedClose
	}
	if i.start < 0 {
		_, err := i.output.WriteRune(r)
		return err
	}
	i.rb = append(i.rb, r)
	return nil
}

func (i *Interpolator) finish() error {
	if i.start >= 0 {
		return ErrExpectingClose
	}
	if i.closing {
		return ErrUnexpectedClose
	}
	return nil
}

// WithFunc interpolates the specified template with replacements using the
// given function.
func WithFunc(template string, format Func) (string, error) {
	buffer := bytes.NewBuffer(make([]byte, 0, len(template)))
	opts := &Options{
		Template: strings.NewReader(template),
		Output:   buffer,
		Format:   format,
	}
	i := NewWithOptions(opts)
	if err := i.Interpolate(); err != nil {
		return "", err
	}
	return buffer.String(), nil
}

// WithMap interpolates the specified template with replacements using the
// given map. If a placeholder is used for which a value is not found, an error
// is returned.
func WithMap(template string, m map[string]string) (string, error) {
	format := func(key string, w io.Writer) error {
		value, ok := m[key]
		if !ok {
			return ErrKeyNotFound
		}
		_, err := w.Write([]byte(value))
		return err
	}
	return WithFunc(template, format)
}
