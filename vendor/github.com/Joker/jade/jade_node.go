package jade

import (
	"bytes"
	"fmt"
	"go/parser"
	"html"
	"io"
	"log"
	"regexp"
	"strings"
)

type TagNode struct {
	NodeType
	Pos
	tr       *Tree
	Nodes    []Node
	AttrName []string
	AttrCode []string
	AttrUesc []bool
	TagName  string
	tagType  itemType
}

func (t *Tree) newTag(pos Pos, name string, tagType itemType) *TagNode {
	return &TagNode{tr: t, NodeType: NodeTag, Pos: pos, TagName: name, tagType: tagType}
}

func (l *TagNode) append(n Node) {
	l.Nodes = append(l.Nodes, n)
}

func (l *TagNode) tree() *Tree {
	return l.tr
}

func (l *TagNode) attr(a, b string, c bool) {
	for k, v := range l.AttrName {
		// add to existing attribute
		if v == a {
			l.AttrCode[k] = fmt.Sprintf(tag__arg_add, l.AttrCode[k], b)
			return
		}
	}

	l.AttrName = append(l.AttrName, a)
	l.AttrCode = append(l.AttrCode, b)
	l.AttrUesc = append(l.AttrUesc, c)
}

func (l *TagNode) ifAttrArgBollean() {
	for k, v := range l.AttrCode {
		if v == "true" {
			l.AttrCode[k] = `"` + l.AttrName[k] + `"`
		} else if v == "false" {
			l.AttrName = append(l.AttrName[:k], l.AttrName[k+1:]...)
			l.AttrCode = append(l.AttrCode[:k], l.AttrCode[k+1:]...)
			l.AttrUesc = append(l.AttrUesc[:k], l.AttrUesc[k+1:]...)
		}
	}
}

func ifAttrArgString(a string, unesc bool) (string, bool) {
	var (
		str   = []rune(a)
		lng   = len(str)
		first = str[0]
		last  = str[lng-1]
	)

	switch first {
	case '"', '\'':
		if first == last {
			for k, v := range str[1 : lng-1] {
				if v == first && str[k] != '\\' {
					return "", false
				}
			}
			if unesc {
				return string(str[1 : lng-1]), true
			}
			return html.EscapeString(string(str[1 : lng-1])), true
		}
	case '`':
		if first == last {
			if !strings.ContainsAny(string(str[1:lng-1]), "`") {
				if unesc {
					return string(str[1 : lng-1]), true
				}
				return html.EscapeString(string(str[1 : lng-1])), true
			}
		}
	}
	return "", false
}

func query(a string) (string, bool) {
	var (
		re    = regexp.MustCompile(`^(.+)\?(.+):(.+)$`)
		match = re.FindStringSubmatch(a)
	)
	if len(match) == 4 {
		for _, v := range match[1:4] {
			if _, err := parser.ParseExpr(v); err != nil {
				return "", false
			}
		}
		return "qf(" + match[1] + ", " + match[2] + ", " + match[3] + ")", true
	}
	return "", false
}

func (l *TagNode) String() string {
	var b = new(bytes.Buffer)
	l.WriteIn(b)
	return b.String()
}
func (l *TagNode) WriteIn(b io.Writer) {
	var (
		attr = new(bytes.Buffer)
	)
	l.ifAttrArgBollean()

	if len(l.AttrName) > 0 {
		fmt.Fprint(attr, tag__arg_bgn)
		for k, name := range l.AttrName {
			if arg, ok := ifAttrArgString(l.AttrCode[k], l.AttrUesc[k]); ok {
				fmt.Fprintf(attr, tag__arg_str, name, arg)

			} else if !golang_mode {
				fmt.Fprintf(attr, tag__arg_esc, name, l.AttrCode[k])

			} else if _, err := parser.ParseExpr(l.AttrCode[k]); err == nil {
				if l.AttrUesc[k] {
					fmt.Fprintf(attr, tag__arg_une, name, l.Pos, l.AttrCode[k])
				} else {
					fmt.Fprintf(attr, tag__arg_esc, name, l.Pos, l.AttrCode[k])
				}

			} else if arg, ok := query(l.AttrCode[k]); ok {
				if l.AttrUesc[k] {
					fmt.Fprintf(attr, tag__arg_une, name, l.Pos, arg)
				} else {
					fmt.Fprintf(attr, tag__arg_esc, name, l.Pos, arg)
				}

			} else {
				log.Fatalln("Error tag attribute value ==> ", l.AttrCode[k])
			}
		}
		fmt.Fprint(attr, tag__arg_end)
	}
	switch l.tagType {
	case itemTagVoid:
		fmt.Fprintf(b, tag__void, l.TagName, attr)
	case itemTagVoidInline:
		fmt.Fprintf(b, tag__void, l.TagName, attr)
	default:
		fmt.Fprintf(b, tag__bgn, l.TagName, attr)
		for _, inner := range l.Nodes {
			inner.WriteIn(b)
		}
		fmt.Fprintf(b, tag__end, l.TagName)
	}
}

func (l *TagNode) CopyTag() *TagNode {
	if l == nil {
		return l
	}
	n := l.tr.newTag(l.Pos, l.TagName, l.tagType)
	n.AttrCode = l.AttrCode
	n.AttrName = l.AttrName
	n.AttrUesc = l.AttrUesc
	for _, elem := range l.Nodes {
		n.append(elem.Copy())
	}
	return n
}

func (l *TagNode) Copy() Node {
	return l.CopyTag()
}

//
//

type CondNode struct {
	NodeType
	Pos
	tr       *Tree
	Nodes    []Node
	cond     string
	condType itemType
}

func (t *Tree) newCond(pos Pos, cond string, condType itemType) *CondNode {
	return &CondNode{tr: t, NodeType: NodeCond, Pos: pos, cond: cond, condType: condType}
}

func (l *CondNode) append(n Node) {
	l.Nodes = append(l.Nodes, n)
}

func (l *CondNode) tree() *Tree {
	return l.tr
}

func (l *CondNode) String() string {
	var b = new(bytes.Buffer)
	l.WriteIn(b)
	return b.String()
}
func (l *CondNode) WriteIn(b io.Writer) {
	switch l.condType {
	case itemIf:
		fmt.Fprintf(b, cond__if, l.cond)
	case itemUnless:
		fmt.Fprintf(b, cond__unless, l.cond)
	case itemCase:
		fmt.Fprintf(b, cond__case, l.cond)
	case itemWhile:
		fmt.Fprintf(b, cond__while, l.cond)
	case itemFor, itemEach:
		if k, v, name, ok := l.parseForArgs(); ok {
			fmt.Fprintf(b, cond__for, k, v, name)
		} else {
			fmt.Fprintf(b, "\n{{ Error malformed each: %s }}", l.cond)
		}
	case itemForIfNotContain:
		if k, v, name, ok := l.parseForArgs(); ok {
			fmt.Fprintf(b, cond__for_if, name, k, v, name)
		} else {
			fmt.Fprintf(b, "\n{{ Error malformed each: %s }}", l.cond)
		}
	default:
		fmt.Fprintf(b, "{{ Error Cond %s }}", l.cond)
	}

	for _, n := range l.Nodes {
		n.WriteIn(b)
	}

	fmt.Fprint(b, cond__end)
}

func (l *CondNode) parseForArgs() (k, v, name string, ok bool) {
	sp := strings.SplitN(l.cond, " in ", 2)
	if len(sp) != 2 {
		return
	}
	name = strings.Trim(sp[1], " ")
	re := regexp.MustCompile(`^(\w+)\s*,\s*(\w+)$`)
	kv := re.FindAllStringSubmatch(strings.Trim(sp[0], " "), -1)
	if len(kv) == 1 && len(kv[0]) == 3 {
		k = kv[0][2]
		v = kv[0][1]
		ok = true
		return
	}
	r2 := regexp.MustCompile(`^\w+$`)
	kv2 := r2.FindAllString(strings.Trim(sp[0], " "), -1)
	if len(kv2) == 1 {
		k = "_"
		v = kv2[0]
		ok = true
		return
	}
	return
}

func (l *CondNode) CopyCond() *CondNode {
	if l == nil {
		return l
	}
	n := l.tr.newCond(l.Pos, l.cond, l.condType)
	for _, elem := range l.Nodes {
		n.append(elem.Copy())
	}
	return n
}

func (l *CondNode) Copy() Node {
	return l.CopyCond()
}

//
//

type CodeNode struct {
	NodeType
	Pos
	tr       *Tree
	codeType itemType
	Code     []byte // The text; may span newlines.
}

func (t *Tree) newCode(pos Pos, text string, codeType itemType) *CodeNode {
	return &CodeNode{tr: t, NodeType: NodeCode, Pos: pos, Code: []byte(text), codeType: codeType}
}

func (t *CodeNode) String() string {
	var b = new(bytes.Buffer)
	t.WriteIn(b)
	return b.String()
}
func (t *CodeNode) WriteIn(b io.Writer) {
	switch t.codeType {
	case itemCodeBuffered:
		if !golang_mode {
			fmt.Fprintf(b, code__buffered, t.Code)
			return
		}
		if code, ok := ifAttrArgString(string(t.Code), false); ok {
			fmt.Fprintf(b, code__buffered, t.Pos, `"`+code+`"`)
		} else {
			fmt.Fprintf(b, code__buffered, t.Pos, t.Code)
		}
	case itemCodeUnescaped:
		if !golang_mode {
			fmt.Fprintf(b, code__unescaped, t.Code)
			return
		}
		fmt.Fprintf(b, code__unescaped, t.Pos, t.Code)
	case itemCode:
		fmt.Fprintf(b, code__longcode, t.Code)
	case itemElse:
		fmt.Fprintf(b, code__else)
	case itemElseIf:
		fmt.Fprintf(b, code__else_if, t.Code)
	case itemForElse:
		fmt.Fprintf(b, code__for_else)
	case itemCaseWhen:
		fmt.Fprintf(b, code__case_when, t.Code)
	case itemCaseDefault:
		fmt.Fprintf(b, code__case_def)
	case itemMixinBlock:
		fmt.Fprintf(b, code__mix_block)
	default:
		fmt.Fprintf(b, "{{ Error Code %s }}", t.Code)
	}
}

func (t *CodeNode) tree() *Tree {
	return t.tr
}

func (t *CodeNode) Copy() Node {
	return &CodeNode{tr: t.tr, NodeType: NodeCode, Pos: t.Pos, codeType: t.codeType, Code: append([]byte{}, t.Code...)}
}

//
//

type BlockNode struct {
	NodeType
	Pos
	tr        *Tree
	blockType itemType
	Name      string
}

func (t *Tree) newBlock(pos Pos, name string, textType itemType) *BlockNode {
	return &BlockNode{tr: t, NodeType: NodeBlock, Pos: pos, Name: name, blockType: textType}
}

func (t *BlockNode) String() string {
	var b = new(bytes.Buffer)
	t.WriteIn(b)
	return b.String()
}
func (t *BlockNode) WriteIn(b io.Writer) {
	var (
		out_blk         = t.tr.block[t.Name]
		out_pre, ok_pre = t.tr.block[t.Name+"_prepend"]
		out_app, ok_app = t.tr.block[t.Name+"_append"]
	)
	if ok_pre {
		out_pre.WriteIn(b)
	}
	out_blk.WriteIn(b)

	if ok_app {
		out_app.WriteIn(b)
	}
}

func (t *BlockNode) tree() *Tree {
	return t.tr
}

func (t *BlockNode) Copy() Node {
	return &BlockNode{tr: t.tr, NodeType: NodeBlock, Pos: t.Pos, blockType: t.blockType, Name: t.Name}
}

//
//

type TextNode struct {
	NodeType
	Pos
	tr       *Tree
	textType itemType
	Text     []byte // The text; may span newlines.
}

func (t *Tree) newText(pos Pos, text []byte, textType itemType) *TextNode {
	return &TextNode{tr: t, NodeType: NodeText, Pos: pos, Text: text, textType: textType}
}

func (t *TextNode) String() string {
	var b = new(bytes.Buffer)
	t.WriteIn(b)
	return b.String()
}
func (t *TextNode) WriteIn(b io.Writer) {
	switch t.textType {
	case itemComment:
		fmt.Fprintf(b, text__comment, t.Text)
	default:
		fmt.Fprintf(b, text__str, t.Text)
	}
}

func (t *TextNode) tree() *Tree {
	return t.tr
}

func (t *TextNode) Copy() Node {
	return &TextNode{tr: t.tr, NodeType: NodeText, Pos: t.Pos, textType: t.textType, Text: append([]byte{}, t.Text...)}
}

//
//

type MixinNode struct {
	NodeType
	Pos
	tr        *Tree
	Nodes     []Node
	AttrName  []string
	AttrCode  []string
	AttrRest  []string
	MixinName string
	block     []Node
	tagType   itemType
}

func (t *Tree) newMixin(pos Pos) *MixinNode {
	return &MixinNode{tr: t, NodeType: NodeMixin, Pos: pos}
}

func (l *MixinNode) append(n Node) {
	l.Nodes = append(l.Nodes, n)
}
func (l *MixinNode) appendToBlock(n Node) {
	l.block = append(l.block, n)
}

func (l *MixinNode) attr(a, b string, c bool) {
	l.AttrName = append(l.AttrName, a)
	l.AttrCode = append(l.AttrCode, b)
}

func (l *MixinNode) tree() *Tree {
	return l.tr
}

func (l *MixinNode) String() string {
	var b = new(bytes.Buffer)
	l.WriteIn(b)
	return b.String()
}
func (l *MixinNode) WriteIn(b io.Writer) {
	var (
		attr = new(bytes.Buffer)
		an   = len(l.AttrName)
		rest = len(l.AttrRest)
	)

	if an > 0 {
		fmt.Fprintf(attr, mixin__var_bgn)
		if rest > 0 {
			fmt.Fprintf(attr, mixin__var_rest, strings.TrimLeft(l.AttrName[an-1], "."), l.AttrRest)
			l.AttrName = l.AttrName[:an-1]
		}
		for k, name := range l.AttrName {
			fmt.Fprintf(attr, mixin__var, name, l.AttrCode[k])
		}
		fmt.Fprintf(attr, mixin__var_end)
	}
	fmt.Fprintf(b, mixin__bgn, attr)

	if len(l.block) > 0 {
		b.Write([]byte(mixin__var_block_bgn))
		for _, n := range l.block {
			n.WriteIn(b)
		}
		b.Write([]byte(mixin__var_block_end))
	} else {
		b.Write([]byte(mixin__var_block))
	}

	for _, n := range l.Nodes {
		n.WriteIn(b)
	}
	fmt.Fprintf(b, mixin__end)
}

func (l *MixinNode) CopyMixin() *MixinNode {
	if l == nil {
		return l
	}
	n := l.tr.newMixin(l.Pos)
	for _, elem := range l.Nodes {
		n.append(elem.Copy())
	}
	return n
}

func (l *MixinNode) Copy() Node {
	return l.CopyMixin()
}

//
//

type DoctypeNode struct {
	NodeType
	Pos
	tr      *Tree
	doctype string
}

func (t *Tree) newDoctype(pos Pos, text string) *DoctypeNode {
	doc := ""
	txt := strings.Trim(text, " ")
	if len(txt) > 0 {
		sls := strings.SplitN(txt, " ", 2)
		switch sls[0] {
		case "5", "html":
			doc = `<!DOCTYPE html%s>`
		case "xml":
			doc = `<?xml version="1.0" encoding="utf-8"%s ?>`
		case "1.1", "xhtml":
			doc = `<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.1//EN" "http://www.w3.org/TR/xhtml11/DTD/xhtml11.dtd"%s>`
		case "basic":
			doc = `<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML Basic 1.1//EN" "http://www.w3.org/TR/xhtml-basic/xhtml-basic11.dtd"%s>`
		case "strict":
			doc = `<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Strict//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-strict.dtd"%s>`
		case "frameset":
			doc = `<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Frameset//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-frameset.dtd"%s>`
		case "transitional":
			doc = `<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd"%s>`
		case "mobile":
			doc = `<!DOCTYPE html PUBLIC "-//WAPFORUM//DTD XHTML Mobile 1.2//EN" "http://www.openmobilealliance.org/tech/DTD/xhtml-mobile12.dtd"%s>`
		case "4", "4strict":
			doc = `<!DOCTYPE HTML PUBLIC "-//W3C//DTD HTML 4.01//EN" "http://www.w3.org/TR/html4/strict.dtd"%s>`
		case "4frameset":
			doc = `<!DOCTYPE HTML PUBLIC "-//W3C//DTD HTML 4.01 Frameset//EN" "http://www.w3.org/TR/html4/frameset.dtd"%s>`
		case "4transitional":
			doc = `<!DOCTYPE HTML PUBLIC "-//W3C//DTD HTML 4.01 Transitional//EN" "http://www.w3.org/TR/html4/loose.dtd"%s>`
		}
		if doc == "" {
			doc = fmt.Sprintf("<!DOCTYPE %s>", txt)
		} else if doc != "" && len(sls) == 2 {
			doc = fmt.Sprintf(doc, " "+sls[1])
		} else {
			doc = fmt.Sprintf(doc, "")
		}
	} else {
		doc = `<!DOCTYPE html>`
	}
	return &DoctypeNode{tr: t, NodeType: NodeDoctype, Pos: pos, doctype: doc}
}
func (d *DoctypeNode) String() string {
	return fmt.Sprintf(text__str, d.doctype)
}
func (d *DoctypeNode) WriteIn(b io.Writer) {
	fmt.Fprintf(b, text__str, d.doctype)
	// b.Write([]byte(d.doctype))
}
func (d *DoctypeNode) tree() *Tree {
	return d.tr
}
func (d *DoctypeNode) Copy() Node {
	return &DoctypeNode{tr: d.tr, NodeType: NodeDoctype, Pos: d.Pos, doctype: d.doctype}
}
