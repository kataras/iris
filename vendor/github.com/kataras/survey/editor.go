package survey

import (
	"bytes"
	"io/ioutil"
	"os"
	"os/exec"
	"runtime"

	"github.com/kataras/survey/core"
	"github.com/kataras/survey/terminal"
)

/*
Editor launches an instance of the users preferred editor on a temporary file.
The editor to use is determined by reading the $VISUAL or $EDITOR environment
variables. If neither of those are present, notepad (on Windows) or vim
(others) is used.
The launch of the editor is triggered by the enter key. Since the response may
be long, it will not be echoed as Input does, instead, it print <Received>.
Response type is a string.

	message := ""
	prompt := &survey.Editor{ Message: "What is your commit message?" }
	survey.AskOne(prompt, &message, nil)
*/
type Editor struct {
	core.Renderer
	Message string
	Default string
	Help    string
}

// data available to the templates when processing
type EditorTemplateData struct {
	Editor
	Answer     string
	ShowAnswer bool
	ShowHelp   bool
}

// Templates with Color formatting. See Documentation: https://github.com/mgutz/ansi#style-format
var EditorQuestionTemplate = `
{{- if .ShowHelp }}{{- color "cyan"}}{{ HelpIcon }} {{ .Help }}{{color "reset"}}{{"\n"}}{{end}}
{{- color "green+hb"}}{{ QuestionIcon }} {{color "reset"}}
{{- color "default+hb"}}{{ .Message }} {{color "reset"}}
{{- if .ShowAnswer}}
  {{- color "cyan"}}{{.Answer}}{{color "reset"}}{{"\n"}}
{{- else }}
  {{- if and .Help (not .ShowHelp)}}{{color "cyan"}}[{{ HelpInputRune }} for help]{{color "reset"}} {{end}}
  {{- if .Default}}{{color "white"}}({{.Default}}) {{color "reset"}}{{end}}
  {{- color "cyan"}}[Enter to launch editor] {{color "reset"}}
{{- end}}`

var (
	bom    = []byte{0xef, 0xbb, 0xbf}
	editor = "vim"
)

func init() {
	if runtime.GOOS == "windows" {
		editor = "notepad"
	}
	if v := os.Getenv("VISUAL"); v != "" {
		editor = v
	} else if e := os.Getenv("EDITOR"); e != "" {
		editor = e
	}
}

func (e *Editor) Prompt() (interface{}, error) {
	// render the template
	err := e.Render(
		EditorQuestionTemplate,
		EditorTemplateData{Editor: *e},
	)
	if err != nil {
		return "", err
	}

	// start reading runes from the standard in
	rr := terminal.NewRuneReader(os.Stdin)
	rr.SetTermMode()
	defer rr.RestoreTermMode()

	terminal.CursorHide()
	defer terminal.CursorShow()

	for {
		r, _, err := rr.ReadRune()
		if err != nil {
			return "", err
		}
		if r == '\r' || r == '\n' {
			break
		}
		if r == terminal.KeyInterrupt {
			return "", terminal.InterruptErr
		}
		if r == terminal.KeyEndTransmission {
			break
		}
		if r == core.HelpInputRune && e.Help != "" {
			err = e.Render(
				EditorQuestionTemplate,
				EditorTemplateData{Editor: *e, ShowHelp: true},
			)
			if err != nil {
				return "", err
			}
		}
		continue
	}

	// prepare the temp file
	f, err := ioutil.TempFile("", "survey")
	if err != nil {
		return "", err
	}
	defer os.Remove(f.Name())

	// write utf8 BOM header
	// The reason why we do this is because notepad.exe on Windows determines the
	// encoding of an "empty" text file by the locale, for example, GBK in China,
	// while golang string only handles utf8 well. However, a text file with utf8
	// BOM header is not considered "empty" on Windows, and the encoding will then
	// be determined utf8 by notepad.exe, instead of GBK or other encodings.
	if _, err := f.Write(bom); err != nil {
		return "", err
	}
	// close the fd to prevent the editor unable to save file
	if err := f.Close(); err != nil {
		return "", err
	}

	// open the editor
	cmd := exec.Command(editor, f.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	terminal.CursorShow()
	if err := cmd.Run(); err != nil {
		return "", err
	}

	// raw is a BOM-unstripped UTF8 byte slice
	raw, err := ioutil.ReadFile(f.Name())
	if err != nil {
		return "", err
	}

	// strip BOM header
	text := string(bytes.TrimPrefix(raw, bom))

	// check length, return default value on empty
	if len(text) == 0 {
		return e.Default, nil
	}

	return text, nil
}

func (e *Editor) Cleanup(val interface{}) error {
	return e.Render(
		EditorQuestionTemplate,
		EditorTemplateData{Editor: *e, Answer: "<Received>", ShowAnswer: true},
	)
}
