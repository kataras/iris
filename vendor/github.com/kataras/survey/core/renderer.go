package core

import (
	"strings"

	"github.com/kataras/survey/terminal"
)

type Renderer struct {
	lineCount      int
	errorLineCount int
}

var ErrorTemplate = `{{color "red"}}{{ ErrorIcon }} Sorry, your reply was invalid: {{.Error}}{{color "reset"}}
`

func (r *Renderer) Error(invalid error) error {
	// since errors are printed on top we need to reset the prompt
	// as well as any previous error print
	r.resetPrompt(r.lineCount + r.errorLineCount)
	// we just cleared the prompt lines
	r.lineCount = 0
	out, err := RunTemplate(ErrorTemplate, invalid)
	if err != nil {
		return err
	}
	// keep track of how many lines are printed so we can clean up later
	r.errorLineCount = strings.Count(out, "\n")

	// send the message to the user
	terminal.Print(out)
	return nil
}

func (r *Renderer) resetPrompt(lines int) {
	// clean out current line in case tmpl didnt end in newline
	terminal.CursorHorizontalAbsolute(0)
	terminal.EraseLine(terminal.ERASE_LINE_ALL)
	// clean up what we left behind last time
	for i := 0; i < lines; i++ {
		terminal.CursorPreviousLine(1)
		terminal.EraseLine(terminal.ERASE_LINE_ALL)
	}
}

func (r *Renderer) Render(tmpl string, data interface{}) error {
	r.resetPrompt(r.lineCount)
	// render the template summarizing the current state
	out, err := RunTemplate(tmpl, data)
	if err != nil {
		return err
	}

	// keep track of how many lines are printed so we can clean up later
	r.lineCount = strings.Count(out, "\n")

	// print the summary
	terminal.Print(out)

	// nothing went wrong
	return nil
}
