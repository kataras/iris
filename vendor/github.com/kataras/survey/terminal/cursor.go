// +build !windows

package terminal

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// CursorUp moves the cursor n cells to up.
func CursorUp(n int) {
	fmt.Printf("\x1b[%dA", n)
}

// CursorDown moves the cursor n cells to down.
func CursorDown(n int) {
	fmt.Printf("\x1b[%dB", n)
}

// CursorForward moves the cursor n cells to right.
func CursorForward(n int) {
	fmt.Printf("\x1b[%dC", n)
}

// CursorBack moves the cursor n cells to left.
func CursorBack(n int) {
	fmt.Printf("\x1b[%dD", n)
}

// CursorNextLine moves cursor to beginning of the line n lines down.
func CursorNextLine(n int) {
	fmt.Printf("\x1b[%dE", n)
}

// CursorPreviousLine moves cursor to beginning of the line n lines up.
func CursorPreviousLine(n int) {
	fmt.Printf("\x1b[%dF", n)
}

// CursorHorizontalAbsolute moves cursor horizontally to x.
func CursorHorizontalAbsolute(x int) {
	fmt.Printf("\x1b[%dG", x)
}

// CursorShow shows the cursor.
func CursorShow() {
	fmt.Print("\x1b[?25h")
}

// CursorHide hide the cursor.
func CursorHide() {
	fmt.Print("\x1b[?25l")
}

// CursorMove moves the cursor to a specific x,y location.
func CursorMove(x int, y int) {
	fmt.Printf("\x1b[%d;%df", x, y)
}

// CursorLocation returns the current location of the cursor in the terminal
func CursorLocation() (*Coord, error) {
	// print the escape sequence to receive the position in our stdin
	fmt.Print("\x1b[6n")

	// read from stdin to get the response
	reader := bufio.NewReader(os.Stdin)
	// spec says we read 'til R, so do that
	text, err := reader.ReadSlice('R')
	if err != nil {
		return nil, err
	}

	// spec also says they're split by ;, so do that too
	if strings.Contains(string(text), ";") {
		// a regex to parse the output of the ansi code
		re := regexp.MustCompile(`\d+;\d+`)
		line := re.FindString(string(text))

		// find the column and rows embedded in the string
		coords := strings.Split(line, ";")

		// try to cast the col number to an int
		col, err := strconv.Atoi(coords[1])
		if err != nil {
			return nil, err
		}

		// try to cast the row number to an int
		row, err := strconv.Atoi(coords[0])
		if err != nil {
			return nil, err
		}

		// return the coordinate object with the col and row we calculated
		return &Coord{Short(col), Short(row)}, nil
	}

	// it didn't work so return an error
	return nil, fmt.Errorf("could not compute the cursor position using ascii escape sequences")
}

// Size returns the height and width of the terminal.
func Size() (*Coord, error) {
	// the general approach here is to move the cursor to the very bottom
	// of the terminal, ask for the current location and then move the
	// cursor back where we started

	// save the current location of the cursor
	origin, err := CursorLocation()
	if err != nil {
		return nil, err
	}

	// move the cursor to the very bottom of the terminal
	CursorMove(999, 999)

	// ask for the current location
	bottom, err := CursorLocation()
	if err != nil {
		return nil, err
	}

	// move back where we began
	CursorUp(int(bottom.Y - origin.Y))
	CursorHorizontalAbsolute(int(origin.X))

	// sice the bottom was calcuated in the lower right corner, it
	// is the dimensions we are looking for
	return bottom, nil
}
