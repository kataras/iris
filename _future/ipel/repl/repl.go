package repl

import (
	"bufio"
	"fmt"
	"io"

	"gopkg.in/kataras/iris.v6/_future/ipel/lexer"
	"gopkg.in/kataras/iris.v6/_future/ipel/token"
)

const PROMPT = ">> "

func Start(in io.Reader, out io.Writer) {
	scanner := bufio.NewScanner(in)
	for {
		fmt.Printf(PROMPT)
		scanned := scanner.Scan()
		if !scanned {
			return
		}
		line := scanner.Text()
		if line == "exit" {
			break
		}
		l := lexer.New(line)
		for tok := l.NextToken(); tok.Type != token.EOF; tok = l.NextToken() {
			fmt.Printf("%+v\n", tok)
		}
	}
}
