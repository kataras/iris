package logger

import (
	"log"
	"os"

	"github.com/kataras/iris/config"
)

var (
	// Prefix is the prefix for the logger, default is [IRIS]
	Prefix = "[IRIS] "
)

// Logger is just a log.Logger
type Logger struct {
	Logger  *log.Logger
	enabled bool
}

// New creates a new Logger.   The out variable sets the
// destination to which log data will be written.
// The prefix appears at the beginning of each generated log line.
// The flag argument defines the logging properties.
func New(cfg ...config.Logger) *Logger {
	c := config.DefaultLogger().Merge(cfg)
	return &Logger{Logger: log.New(c.Out, Prefix+c.Prefix, c.Flag), enabled: true}
}

// SetEnable true enables, false disables the Logger
func (l *Logger) SetEnable(enable bool) {
	l.enabled = enable
}

// IsEnabled returns true if Logger is enabled, otherwise false
func (l *Logger) IsEnabled() bool {
	return l.enabled
}

// Print calls l.Output to print to the logger.
// Arguments are handled in the manner of fmt.Print.
func (l *Logger) Print(v ...interface{}) {
	if l.enabled {
		l.Logger.Print(v...)
	}
}

// Printf calls l.Output to print to the logger.
// Arguments are handled in the manner of fmt.Printf.
func (l *Logger) Printf(format string, a ...interface{}) {
	if l.enabled {
		l.Logger.Printf(format, a...)
	}
}

// Println calls l.Output to print to the logger.
// Arguments are handled in the manner of fmt.Println.
func (l *Logger) Println(a ...interface{}) {
	if l.enabled {
		l.Logger.Println(a...)
	}
}

// Fatal is equivalent to l.Print() followed by a call to os.Exit(1).
func (l *Logger) Fatal(a ...interface{}) {
	if l.enabled {
		l.Logger.Fatal(a...)
	} else {
		os.Exit(1) //we have to exit at any case because this is the Fatal
	}

}

// Fatalf is equivalent to l.Printf() followed by a call to os.Exit(1).
func (l *Logger) Fatalf(format string, a ...interface{}) {
	if l.enabled {
		l.Logger.Fatalf(format, a...)
	} else {
		os.Exit(1)
	}
}

// Fatalln is equivalent to l.Println() followed by a call to os.Exit(1).
func (l *Logger) Fatalln(a ...interface{}) {
	if l.enabled {
		l.Logger.Fatalln(a...)
	} else {
		os.Exit(1)
	}
}

// Panic is equivalent to l.Print() followed by a call to panic().
func (l *Logger) Panic(a ...interface{}) {
	if l.enabled {
		l.Logger.Panic(a...)
	}
}

// Panicf is equivalent to l.Printf() followed by a call to panic().
func (l *Logger) Panicf(format string, a ...interface{}) {
	if l.enabled {
		l.Logger.Panicf(format, a...)
	}
}

// Panicln is equivalent to l.Println() followed by a call to panic().
func (l *Logger) Panicln(a ...interface{}) {
	if l.enabled {
		l.Logger.Panicln(a...)
	}
}
