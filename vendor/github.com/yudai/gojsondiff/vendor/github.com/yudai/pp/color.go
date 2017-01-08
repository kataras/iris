package pp

import (
	"fmt"
	"strings"
)

var (
	// If you set false to this variable, you can use pretty formatter
	// without coloring.
	ColoringEnabled = true

	codeByColor = map[string]int{
		"black":   30,
		"red":     31,
		"green":   32,
		"yellow":  33,
		"blue":    34,
		"magenta": 35,
		"cyan":    36,
		"white":   37,
	}

	black       = colorizer("black")
	red         = colorizer("red")
	green       = colorizer("green")
	yellow      = colorizer("yellow")
	blue        = colorizer("blue")
	magenta     = colorizer("magenta")
	cyan        = colorizer("cyan")
	white       = colorizer("white")
	boldBlack   = colorizer("Black")
	boldRed     = colorizer("Red")
	boldGreen   = colorizer("Green")
	boldYellow  = colorizer("Yellow")
	boldBlue    = colorizer("Blue")
	boldMagenta = colorizer("Magenta")
	boldCyan    = colorizer("Cyan")
	boldWhite   = colorizer("White")
)

func colorize(text, color string) string {
	return colorizer(color)(text)
}

func colorizer(color string) func(string) string {
	if !ColoringEnabled {
		return func(text string) string { return text }
	}

	if code, ok := codeByColor[color]; ok {
		return func(text string) string {
			return fmt.Sprintf("\033[%dm%s\033[0m", code, text)
		}
	} else if code, ok := codeByColor[strings.ToLower(color)]; ok {
		return func(text string) string {
			return fmt.Sprintf("\033[%dm\033[1m%s\033[0m", code, text)
		}
	} else {
		panic("undefined colorizer: " + color)
	}
}
