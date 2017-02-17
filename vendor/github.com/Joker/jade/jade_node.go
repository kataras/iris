package jade

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
)

// NodeType identifies the type of a parse tree node.
type nodeType int

const (
	nodeList nodeType = iota
	nodeText
	nodeTag
	nodeAttr
	nodeDoctype
)

func indentToString(nesting, indent int, fromConf bool) string {
	if PrettyOutput {
		idt := new(bytes.Buffer)
		if fromConf {
			for i := 0; i < nesting; i++ {
				idt.WriteString(OutputIndent)
			}
		} else {
			for i := 0; i < indent; i++ {
				idt.WriteByte(' ')
			}
		}
		return idt.String()
	}
	return ""
}

type nestNode struct {
	nodeType
	psn
	tr    *tree
	Nodes []node

	typ     itemType
	Tag     string
	Indent  int
	Nesting int

	id    string
	class []string
}

func (t *tree) newNest(pos psn, tag string, tp itemType, idt, nst int) *nestNode {
	return &nestNode{tr: t, nodeType: nodeTag, psn: pos, Tag: tag, typ: tp, Indent: idt, Nesting: nst}
}

func (nn *nestNode) append(n node) {
	nn.Nodes = append(nn.Nodes, n)
}
func (nn *nestNode) tree() *tree {
	return nn.tr
}
func (nn *nestNode) tp() itemType {
	return nn.typ
}

func (nn *nestNode) String() string {
	b := new(bytes.Buffer)
	idt := indentToString(nn.Nesting, nn.Indent, nestIndent)

	if PrettyOutput && nn.typ != itemInlineTag && nn.typ != itemInlineVoidTag {
		idt = "\n" + idt
	}

	beginFormat := idt + "<%s"
	endFormat := "</%s>"

	switch nn.typ {
	case itemDiv:
		nn.Tag = "div"
	case itemInlineTag, itemInlineVoidTag:
		beginFormat = "<%s"
	case itemComment:
		nn.Tag = "--"
		beginFormat = idt + "<!%s "
		endFormat = " %s>"
	case itemAction, itemActionEnd:
		beginFormat = idt + "{{ %s }}"
	case itemBlock:
		beginFormat = idt + "{{ block \"%s\" . }}"
	case itemDefine:
		beginFormat = idt + "{{ define \"%s\" }}"
	case itemTemplate:
		beginFormat = idt + "{{ template \"%s\" }}"
	}

	if len(nn.Nodes) > 1 || (len(nn.Nodes) == 1 && nn.Nodes[0].tp() != itemEndAttr) {
		endEl := nn.Nodes[len(nn.Nodes)-1].tp()
		if endEl != itemInlineText && endEl != itemInlineAction && endEl != itemInlineTag {
			endFormat = idt + endFormat
		}
	}

	fmt.Fprintf(b, beginFormat, nn.Tag)

	if len(nn.id) > 0 {
		fmt.Fprintf(b, " id=\"%s\"", nn.id)
	}
	if len(nn.class) > 0 {
		fmt.Fprintf(b, " class=\"%s\"", strings.Join(nn.class, " "))
	}
	for _, n := range nn.Nodes {
		fmt.Fprint(b, n)
	}
	switch nn.typ {
	case itemActionEnd, itemDefine, itemBlock:
		fmt.Fprint(b, idt+"{{ end }}")
	case itemTag, itemDiv, itemInlineTag, itemComment:
		fmt.Fprintf(b, endFormat, nn.Tag)
	}
	return b.String()
}

func (nn *nestNode) CopyNest() *nestNode {
	if nn == nil {
		return nn
	}
	n := nn.tr.newNest(nn.psn, nn.Tag, nn.typ, nn.Indent, nn.Nesting)
	for _, elem := range nn.Nodes {
		n.append(elem.Copy())
	}
	return n
}
func (nn *nestNode) Copy() node {
	return nn.CopyNest()
}

var doctype = map[string]string{
	"xml":           `<?xml version="1.0" encoding="utf-8" ?>`,
	"html":          `<!DOCTYPE html>`,
	"5":             `<!DOCTYPE html>`,
	"1.1":           `<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.1//EN" "http://www.w3.org/TR/xhtml11/DTD/xhtml11.dtd">`,
	"xhtml":         `<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.1//EN" "http://www.w3.org/TR/xhtml11/DTD/xhtml11.dtd">`,
	"basic":         `<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML Basic 1.1//EN" "http://www.w3.org/TR/xhtml-basic/xhtml-basic11.dtd">`,
	"mobile":        `<!DOCTYPE html PUBLIC "-//WAPFORUM//DTD XHTML Mobile 1.2//EN" "http://www.openmobilealliance.org/tech/DTD/xhtml-mobile12.dtd">`,
	"strict":        `<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Strict//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-strict.dtd">`,
	"frameset":      `<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Frameset//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-frameset.dtd">`,
	"transitional":  `<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd">`,
	"4":             `<!DOCTYPE HTML PUBLIC "-//W3C//DTD HTML 4.01//EN" "http://www.w3.org/TR/html4/strict.dtd">`,
	"4strict":       `<!DOCTYPE HTML PUBLIC "-//W3C//DTD HTML 4.01//EN" "http://www.w3.org/TR/html4/strict.dtd">`,
	"4frameset":     `<!DOCTYPE HTML PUBLIC "-//W3C//DTD HTML 4.01 Frameset//EN" "http://www.w3.org/TR/html4/frameset.dtd"> `,
	"4transitional": `<!DOCTYPE HTML PUBLIC "-//W3C//DTD HTML 4.01 Transitional//EN" "http://www.w3.org/TR/html4/loose.dtd">`,
}

type doctypeNode struct {
	nodeType
	psn
	tr      *tree
	Doctype string
}

func (t *tree) newDoctype(pos psn, dt string) *doctypeNode {
	return &doctypeNode{tr: t, nodeType: nodeDoctype, psn: pos, Doctype: dt}
}

func (d *doctypeNode) String() string {
	if dt, ok := doctype[d.Doctype]; ok {
		return fmt.Sprintf("\n%s", dt)
	}
	return fmt.Sprintf("\n<!DOCTYPE html>")
}

func (d *doctypeNode) tp() itemType {
	return itemDoctype
}
func (d *doctypeNode) tree() *tree {
	return d.tr
}
func (d *doctypeNode) Copy() node {
	return &doctypeNode{tr: d.tr, nodeType: nodeDoctype, psn: d.psn, Doctype: d.Doctype}
}

// lineNode holds plain text.
type lineNode struct {
	nodeType
	psn
	tr *tree

	Text    []byte // The text; may span newlines.
	typ     itemType
	Indent  int
	Nesting int
}

func (t *tree) newLine(pos psn, text string, tp itemType, idt, nst int) *lineNode {
	return &lineNode{tr: t, nodeType: nodeText, psn: pos, Text: []byte(text), typ: tp, Indent: idt, Nesting: nst}
}

func (l *lineNode) String() string {
	idt := indentToString(l.Nesting, l.Indent, lineIndent)
	rex := regexp.MustCompile("[!#]{(.+?)}")

	switch l.typ {
	case itemInlineAction:
		return fmt.Sprintf("{{%s }}", l.Text)
	case itemInlineText:
		return fmt.Sprintf("%s", rex.ReplaceAll(l.Text, []byte("{{$1}}")))
	default:
		return fmt.Sprintf("\n"+idt+"%s", rex.ReplaceAll(l.Text, []byte("{{$1}}")))
	}
}

func (l *lineNode) tp() itemType {
	return l.typ
}
func (l *lineNode) tree() *tree {
	return l.tr
}
func (l *lineNode) Copy() node {
	return &lineNode{tr: l.tr, nodeType: nodeText, psn: l.psn, Text: append([]byte{}, l.Text...), typ: l.typ, Indent: l.Indent, Nesting: l.Nesting}
}

type attrNode struct {
	nodeType
	psn
	tr   *tree
	Attr string
	typ  itemType
}

func (t *tree) newAttr(pos psn, attr string, tp itemType) *attrNode {
	return &attrNode{tr: t, nodeType: nodeAttr, psn: pos, Attr: attr, typ: tp}
}

func (a *attrNode) String() string {
	switch a.typ {
	case itemEndAttr:
		return fmt.Sprintf("%s", a.Attr)
	case itemAttr:
		return fmt.Sprintf("=%s", a.Attr)
	case itemAttrN:
		return fmt.Sprintf("=\"%s\"", a.Attr)
	case itemAttrVoid:
		return fmt.Sprintf(" %s=\"%s\"", a.Attr, a.Attr)
	default:
		return fmt.Sprintf(" %s", a.Attr)
	}
}

func (a *attrNode) tp() itemType {
	return a.typ
}
func (a *attrNode) tree() *tree {
	return a.tr
}

func (a *attrNode) Copy() node {
	return &attrNode{tr: a.tr, nodeType: nodeAttr, psn: a.psn, Attr: a.Attr, typ: a.typ}
}
