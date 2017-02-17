package color

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/mattn/go-colorable"
	"github.com/mattn/go-isatty"
)

// NoColor defines if the output is colorized or not. It's dynamically set to
// false or true based on the stdout's file descriptor referring to a terminal
// or not. This is a global option and affects all colors. For more control
// over each color block use the methods DisableColor() individually.
var NoColor = !isatty.IsTerminal(os.Stdout.Fd())

// Output defines the default output of the print functions.
// os.Stdout is used.
var DefaultOutput = colorable.NewColorable(os.Stdout)

// Color defines a custom color object which is defined by SGR parameters.
type Color struct {
	Output  io.Writer
	params  []Attribute
	noColor *bool
	mu      sync.Mutex
}

// Attribute defines a single SGR Code
type Attribute int

const escape = "\x1b"

// Base attributes
const (
	Reset Attribute = iota
	Bold
	Faint
	Italic
	Underline
	BlinkSlow
	BlinkRapid
	ReverseVideo
	Concealed
	CrossedOut
)

// Foreground text colors
const (
	FgBlack Attribute = iota + 30
	FgRed
	FgGreen
	FgYellow
	FgBlue
	FgMagenta
	FgCyan
	FgWhite
)

// Foreground Hi-Intensity text colors
const (
	FgHiBlack Attribute = iota + 90
	FgHiRed
	FgHiGreen
	FgHiYellow
	FgHiBlue
	FgHiMagenta
	FgHiCyan
	FgHiWhite
)

// Background text colors
const (
	BgBlack Attribute = iota + 40
	BgRed
	BgGreen
	BgYellow
	BgBlue
	BgMagenta
	BgCyan
	BgWhite
)

// Background Hi-Intensity text colors
const (
	BgHiBlack Attribute = iota + 100
	BgHiRed
	BgHiGreen
	BgHiYellow
	BgHiBlue
	BgHiMagenta
	BgHiCyan
	BgHiWhite
)

// New returns a newly created color object
func New(out io.Writer, value ...Attribute) *Color {
	c := &Color{Output: out, params: make([]Attribute, 0)}
	c.Add(value...)
	return c
}

// Default returns a newly created color object with the default Out stream.
func Default(value ...Attribute) *Color {
	return New(DefaultOutput, value...)
}

// Set sets the given parameters immediately. It will change the color of
// output with the given SGR parameters until color.Unset() is called.
func Set(p ...Attribute) *Color {
	c := Default(p...)
	c.Set()
	return c
}

// Unset resets all escape attributes and clears the output. Usually should
// be called after Set().
func Unset() {
	if NoColor {
		return
	}

	fmt.Fprintf(DefaultOutput, "%s[%dm", escape, Reset)
}

// Set sets the SGR sequence.
func (c *Color) Set() *Color {
	if c.isNoColorSet() {
		return c
	}

	fmt.Fprintf(c.Output, c.format())
	return c
}

func (c *Color) unset() {
	if c.isNoColorSet() {
		return
	}

	Unset()
}

// Add is used to chain SGR parameters. Use as many as parameters to combine
// and create custom color objects. Example: Add(color.FgRed, color.Underline).
func (c *Color) Add(value ...Attribute) *Color {
	c.params = append(c.params, value...)
	return c
}

func (c *Color) prepend(value Attribute) {
	c.params = append(c.params, 0)
	copy(c.params[1:], c.params[0:])
	c.params[0] = value
}

// Print formats using the default formats for its operands and writes to
// standard output. Spaces are added between operands when neither is a
// string. It returns the number of bytes written and any write error
// encountered. This is the standard fmt.Print() method wrapped with the given
// color.
func (c *Color) Print(a ...interface{}) (n int, err error) {
	c.Set()
	defer c.unset()

	return fmt.Fprint(c.Output, a...)
}

// Printf formats according to a format specifier and writes to standard output.
// It returns the number of bytes written and any write error encountered.
// This is the standard fmt.Printf() method wrapped with the given color.
func (c *Color) Printf(format string, a ...interface{}) (n int, err error) {
	c.Set()
	defer c.unset()

	return fmt.Fprintf(c.Output, format, a...)
}

// Println formats using the default formats for its operands and writes to
// standard output. Spaces are always added between operands and a newline is
// appended. It returns the number of bytes written and any write error
// encountered. This is the standard fmt.Print() method wrapped with the given
// color.
func (c *Color) Println(a ...interface{}) (n int, err error) {
	c.Set()
	defer c.unset()

	return fmt.Fprintln(c.Output, a...)
}

// PrintFunc returns a new function that prints the passed arguments as
// colorized with color.Print().
func (c *Color) PrintFunc() func(a ...interface{}) {
	return func(a ...interface{}) { c.Print(a...) }
}

// PrintfFunc returns a new function that prints the passed arguments as
// colorized with color.Printf().
func (c *Color) PrintfFunc() func(format string, a ...interface{}) {
	return func(format string, a ...interface{}) { c.Printf(format, a...) }
}

// PrintlnFunc returns a new function that prints the passed arguments as
// colorized with color.Println().
func (c *Color) PrintlnFunc() func(a ...interface{}) {
	return func(a ...interface{}) { c.Println(a...) }
}

// SprintFunc returns a new function that returns colorized strings for the
// given arguments with fmt.Sprint(). Useful to put into or mix into other
// string. Windows users should use this in conjuction with color.Output, example:
//
//	put := New(FgYellow).SprintFunc()
//	fmt.Fprintf(color.Output, "This is a %s", put("warning"))
func (c *Color) SprintFunc() func(a ...interface{}) string {
	return func(a ...interface{}) string {
		return c.wrap(fmt.Sprint(a...))
	}
}

// SprintfFunc returns a new function that returns colorized strings for the
// given arguments with fmt.Sprintf(). Useful to put into or mix into other
// string. Windows users should use this in conjuction with color.Output.
func (c *Color) SprintfFunc() func(format string, a ...interface{}) string {
	return func(format string, a ...interface{}) string {
		return c.wrap(fmt.Sprintf(format, a...))
	}
}

// SprintlnFunc returns a new function that returns colorized strings for the
// given arguments with fmt.Sprintln(). Useful to put into or mix into other
// string. Windows users should use this in conjuction with color.Output.
func (c *Color) SprintlnFunc() func(a ...interface{}) string {
	return func(a ...interface{}) string {
		return c.wrap(fmt.Sprintln(a...))
	}
}

// sequence returns a formated SGR sequence to be plugged into a "\x1b[...m"
// an example output might be: "1;36" -> bold cyan
func (c *Color) sequence() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	format := make([]string, len(c.params))
	for i, v := range c.params {
		if len(format) > i {
			format[i] = strconv.Itoa(int(v))
		}
	}

	return strings.Join(format, ";")
}

// wrap wraps the s string with the colors attributes. The string is ready to
// be printed.
func (c *Color) wrap(s string) string {
	if c.isNoColorSet() {
		return s
	}

	return c.format() + s + c.unformat()
}

func (c *Color) format() string {
	return fmt.Sprintf("%s[%sm", escape, c.sequence())
}

func (c *Color) unformat() string {
	return fmt.Sprintf("%s[%dm", escape, Reset)
}

// DisableColor disables the color output. Useful to not change any existing
// code and still being able to output. Can be used for flags like
// "--no-color". To enable back use EnableColor() method.
func (c *Color) DisableColor() {
	c.noColor = boolPtr(true)
}

// EnableColor enables the color output. Use it in conjuction with
// DisableColor(). Otherwise this method has no side effects.
func (c *Color) EnableColor() {
	c.noColor = boolPtr(false)
}

func (c *Color) isNoColorSet() bool {
	// check first if we have user setted action
	if c.noColor != nil {
		return *c.noColor
	}

	// if not return the global option, which is disabled by default
	return NoColor
}

// Equals returns a boolean value indicating whether two colors are equal.
func (c *Color) Equals(c2 *Color) bool {
	if len(c.params) != len(c2.params) {
		return false
	}

	for _, attr := range c.params {
		if !c2.attrExists(attr) {
			return false
		}
	}

	return true
}

func (c *Color) attrExists(a Attribute) bool {
	for _, attr := range c.params {
		if attr == a {
			return true
		}
	}

	return false
}

func boolPtr(v bool) *bool {
	return &v
}

// Black is an convenient helper function to print with black foreground. A
// newline is appended to format by default.
func Black(format string, a ...interface{}) { printColor(format, FgBlack, a...) }

// Red is an convenient helper function to print with red foreground. A
// newline is appended to format by default.
func Red(format string, a ...interface{}) { printColor(format, FgRed, a...) }

// Green is an convenient helper function to print with green foreground. A
// newline is appended to format by default.
func Green(format string, a ...interface{}) { printColor(format, FgGreen, a...) }

// Yellow is an convenient helper function to print with yellow foreground.
// A newline is appended to format by default.
func Yellow(format string, a ...interface{}) { printColor(format, FgYellow, a...) }

// Blue is an convenient helper function to print with blue foreground. A
// newline is appended to format by default.
func Blue(format string, a ...interface{}) { printColor(format, FgBlue, a...) }

// Magenta is an convenient helper function to print with magenta foreground.
// A newline is appended to format by default.
func Magenta(format string, a ...interface{}) { printColor(format, FgMagenta, a...) }

// Cyan is an convenient helper function to print with cyan foreground. A
// newline is appended to format by default.
func Cyan(format string, a ...interface{}) { printColor(format, FgCyan, a...) }

// White is an convenient helper function to print with white foreground. A
// newline is appended to format by default.
func White(format string, a ...interface{}) { printColor(format, FgWhite, a...) }

func printColor(format string, p Attribute, a ...interface{}) {
	if !strings.HasSuffix(format, "\n") {
		format += "\n"
	}

	c := &Color{Output: DefaultOutput, params: []Attribute{p}}
	c.Printf(format, a...)
}

// BlackString is an convenient helper function to return a string with black
// foreground.
func BlackString(format string, a ...interface{}) string {
	return Default(FgBlack).SprintfFunc()(format, a...)
}

// RedString is an convenient helper function to return a string with red
// foreground.
func RedString(format string, a ...interface{}) string {
	return Default(FgRed).SprintfFunc()(format, a...)
}

// GreenString is an convenient helper function to return a string with green
// foreground.
func GreenString(format string, a ...interface{}) string {
	return Default(FgGreen).SprintfFunc()(format, a...)
}

// YellowString is an convenient helper function to return a string with yellow
// foreground.
func YellowString(format string, a ...interface{}) string {
	return Default(FgYellow).SprintfFunc()(format, a...)
}

// BlueString is an convenient helper function to return a string with blue
// foreground.
func BlueString(format string, a ...interface{}) string {
	return Default(FgBlue).SprintfFunc()(format, a...)
}

// MagentaString is an convenient helper function to return a string with magenta
// foreground.
func MagentaString(format string, a ...interface{}) string {
	return Default(FgMagenta).SprintfFunc()(format, a...)
}

// CyanString is an convenient helper function to return a string with cyan
// foreground.
func CyanString(format string, a ...interface{}) string {
	return Default(FgCyan).SprintfFunc()(format, a...)
}

// WhiteString is an convenient helper function to return a string with white
// foreground.
func WhiteString(format string, a ...interface{}) string {
	return Default(FgWhite).SprintfFunc()(format, a...)
}
