package rizla

import (
	"os"

	"github.com/iris-contrib/color"
	"github.com/mattn/go-colorable"
)

// Printer the printer for a rizla instance
type Printer struct {
	*color.Color
	// stream is the output stream which the program will use
	stream *os.File
}

// NewPrinter returns a new colorable printer
func NewPrinter(out *os.File) *Printer {
	c := color.New(colorable.NewColorable(out))
	return &Printer{
		Color:  c,
		stream: out,
	}
}

// Dangerf prints a message with red colored letters
func (printer *Printer) Dangerf(format string, a ...interface{}) {
	printer.Add(color.FgRed)
	printer.Printf(format, a...)
}

// Infof prints a message with cyan colored letters
func (printer *Printer) Infof(format string, a ...interface{}) {
	printer.Add(color.FgCyan)
	printer.Printf(format, a...)
}

// Successf prints a message with green colored letters
func (printer *Printer) Successf(format string, a ...interface{}) {
	printer.Add(color.FgGreen)
	printer.Printf(format, a...)
}

// Name returns the underline output stream Name
func (printer *Printer) Name() string {
	return printer.stream.Name()
}

// Close closes the underline output stream
func (printer *Printer) Close() error {
	return printer.stream.Close()
}
