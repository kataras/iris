package config

import (
	"github.com/fatih/color"
	"github.com/imdario/mergo"
)

import (
	"os"
)

// DefaultLoggerPrefix is the prefix (expect the [IRIS]), is empty for now
const DefaultLoggerPrefix = ""

var (
	// TimeFormat default time format for any kind of datetime parsing
	TimeFormat = "Mon, 02 Jan 2006 15:04:05 GMT"
)

type (
	// Logger contains the full configuration options fields for the Logger
	Logger struct {
		// Out the (file) writer which the messages/logs will printed to
		// Default is os.Stdout
		Out *os.File
		// Prefix the prefix for each message
		// Default is ""
		Prefix string
		// Disabled default is false
		Disabled bool

		// foreground colors single SGR Code

		// ColorFgDefault the foreground color for the normal message bodies
		ColorFgDefault int
		// ColorFgInfo the foreground  color for info messages
		ColorFgInfo int
		// ColorFgSuccess the foreground color for success messages
		ColorFgSuccess int
		// ColorFgWarning the foreground color for warning messages
		ColorFgWarning int
		// ColorFgDanger the foreground color for error messages
		ColorFgDanger int
		// OtherFgColor the foreground color for the rest of the message types
		ColorFgOther int

		// background colors single SGR Code

		// ColorBgDefault the background color for the normal messages
		ColorBgDefault int
		// ColorBgInfo the background  color for info messages
		ColorBgInfo int
		// ColorBgSuccess the background color for success messages
		ColorBgSuccess int
		// ColorBgWarning the background color for warning messages
		ColorBgWarning int
		// ColorBgDanger the background color for error messages
		ColorBgDanger int
		// OtherFgColor the background color for the rest of the message types
		ColorBgOther int

		// banners are the force printed/written messages, doesn't care about Disabled field

		// ColorFgBanner the foreground color for the banner
		ColorFgBanner int
	}
)

// DefaultLogger returns the default configs for the Logger
func DefaultLogger() Logger {
	return Logger{
		Out:      os.Stdout,
		Prefix:   "",
		Disabled: false,
		// foreground colors
		ColorFgDefault: int(color.FgHiWhite),
		ColorFgInfo:    int(color.FgHiCyan),
		ColorFgSuccess: int(color.FgHiGreen),
		ColorFgWarning: int(color.FgHiMagenta),
		ColorFgDanger:  int(color.FgHiRed),
		ColorFgOther:   int(color.FgHiYellow),
		// background colors
		ColorBgDefault: int(color.BgHiBlack),
		ColorBgInfo:    int(color.BgHiBlack),
		ColorBgSuccess: int(color.BgHiBlack),
		ColorBgWarning: int(color.BgHiBlack),
		ColorBgDanger:  int(color.BgHiWhite),
		ColorBgOther:   int(color.BgHiBlack),
		// banner color
		ColorFgBanner: int(color.FgHiBlue),
	}
}

// MergeSingle merges the default with the given config and returns the result
func (c Logger) MergeSingle(cfg Logger) (config Logger) {

	config = cfg
	mergo.Merge(&config, c)

	return
}
