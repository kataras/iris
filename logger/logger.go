package logger

import (
	"os"

	"github.com/fatih/color"
	"github.com/kataras/iris/config"
	"github.com/mattn/go-colorable"
)

var (
	// Prefix is the prefix for the logger, default is [IRIS]
	Prefix = "[IRIS] "
	// bannersRan keeps track of the logger's print banner count
	bannersRan = 0
)

// Logger the logger
type Logger struct {
	config    *config.Logger
	underline *color.Color
}

// attr takes a color integer and converts it to color.Attribute
func attr(sgr int) color.Attribute {
	return color.Attribute(sgr)
}

// check if background color > 0 and if so then set it
func (l *Logger) setBg(sgr int) {
	if sgr > 0 {
		l.underline.Add(attr(sgr))
	}
}

// New creates a new Logger from config.Logger configuration
func New(c config.Logger) *Logger {
	color.Output = colorable.NewColorable(c.Out)

	l := &Logger{&c, color.New(attr(c.ColorFgDefault))}
	l.setBg(c.ColorBgDefault)

	return l
}

// SetEnable true enables, false disables the Logger
func (l *Logger) SetEnable(enable bool) {
	l.config.Disabled = !enable
}

// IsEnabled returns true if Logger is enabled, otherwise false
func (l *Logger) IsEnabled() bool {
	return !l.config.Disabled
}

// ResetColors sets the colors to the default
// this func is called every time a success, info, warning, or danger message is printed
func (l *Logger) ResetColors() {
	l.underline.Add(attr(l.config.ColorFgDefault))
	l.setBg(l.config.ColorBgDefault)
}

// PrintBanner prints a text (banner) with BannerFgColor, BannerBgColor and a success message at the end
// It doesn't cares if the logger is disabled or not, it will print this
func (l *Logger) PrintBanner(banner string, successMessage string) {
	c := color.New(attr(l.config.ColorFgBanner))
	if l.config.ColorBgDefault > 0 {
		c.Add(attr(l.config.ColorBgDefault))
	}
	c.Println(banner)
	bannersRan++

	if successMessage != "" {
		c.Add(attr(l.config.ColorFgSuccess))
		if l.config.ColorBgSuccess > 0 {
			c.Add(attr(l.config.ColorBgSuccess))
		}
		if bannersRan > 1 {
			c.Printf("Server[%#v]\n", bannersRan)

		}
		c.Println(successMessage)
	}

	c.DisableColor()
	c = nil
}

// Printf calls l.Output to print to the logger.
// Arguments are handled in the manner of fmt.Printf.
func (l *Logger) Printf(format string, a ...interface{}) {
	if !l.config.Disabled {
		l.underline.Printf(l.config.Prefix+format, a...)
	}
}

// Print calls l.Output to print to the logger.
// Arguments are handled in the manner of fmt.Print.
func (l *Logger) Print(a interface{}) {
	if !l.config.Disabled {
		l.ResetColors()
		l.Printf("%#v", a)
	}
}

// Println calls l.Output to print to the logger.
// Arguments are handled in the manner of fmt.Println.
func (l *Logger) Println(a interface{}) {
	if !l.config.Disabled {
		l.Printf("%#v\n", a)
	}
}

// Fatal is equivalent to l.Dangerf("%#v",interface{}) followed by a call to panic().
func (l *Logger) Fatal(a interface{}) {
	l.Warningf("%#v", a)
	panic("")
}

// Fatalf is equivalent to l.Warningf() followed by a call to os.Exit(1).
func (l *Logger) Fatalf(format string, a ...interface{}) {
	l.Warningf(format, a...)
	os.Exit(1)
}

// Panic is equivalent to l.Dangerf("%#v",interface{}) followed by a call to panic().
func (l *Logger) Panic(a interface{}) {
	l.Dangerf("%s\n", a)
	panic(a)
}

// Panicf is equivalent to l.Dangerf() followed by a call to panic().
func (l *Logger) Panicf(format string, a ...interface{}) {
	l.Dangerf(format, a...)
	panic("")
}

// Successf calls l.Output to print to the logger with the Success colors.
// Arguments are handled in the manner of fmt.Printf.
func (l *Logger) Successf(format string, a ...interface{}) {
	if !l.config.Disabled {
		l.underline.Add(attr(l.config.ColorFgSuccess))
		l.setBg(l.config.ColorBgSuccess)
		l.Printf(format, a...)
	}
}

// Infof calls l.Output to print to the logger with the Info colors.
// Arguments are handled in the manner of fmt.Printf.
func (l *Logger) Infof(format string, a ...interface{}) {
	if !l.config.Disabled {
		l.underline.Add(attr(l.config.ColorFgInfo))
		l.setBg(l.config.ColorBgInfo)
		l.Printf(format, a...)
	}
}

// Warningf calls l.Output to print to the logger with the Warning colors.
// Arguments are handled in the manner of fmt.Printf.
func (l *Logger) Warningf(format string, a ...interface{}) {
	if !l.config.Disabled {
		l.underline.Add(attr(l.config.ColorFgWarning))
		l.setBg(l.config.ColorBgWarning)
		l.Printf(format, a...)
	}
}

// Dangerf calls l.Output to print to the logger with the Danger colors.
// Arguments are handled in the manner of fmt.Printf.
func (l *Logger) Dangerf(format string, a ...interface{}) {
	if !l.config.Disabled {
		l.underline.Add(attr(l.config.ColorFgDanger))
		l.setBg(l.config.ColorBgDanger)
		l.Printf(format, a...)
	}
}

// Otherf calls l.Output to print to the logger with the Other colors.
// Arguments are handled in the manner of fmt.Printf.
func (l *Logger) Otherf(format string, a ...interface{}) {
	if !l.config.Disabled {
		l.underline.Add(attr(l.config.ColorFgOther))
		l.setBg(l.config.ColorBgOther)
		l.Printf(format, a...)
	}
}
