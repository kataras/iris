package pongo2

type nodeHTML struct {
	token *Token
}

func (n *nodeHTML) Execute(ctx *ExecutionContext, writer TemplateWriter) *Error {
	writer.WriteString(n.token.Val)
	return nil
}
