package jade

//go:generate stringer -type=itemType,NodeType -trimprefix=item -output=config_string.go

var TabSize = 4

var (
	golang_mode  = false
	tag__bgn     = "<%s%s>"
	tag__end     = "</%s>"
	tag__void    = "<%s%s/>"
	tag__arg_esc = ` %s="{{ print %s }}"`
	tag__arg_une = ` %s="{{ print %s }}"`
	tag__arg_str = ` %s="%s"`
	tag__arg_add = `%s " " %s`
	tag__arg_bgn = ""
	tag__arg_end = ""

	cond__if     = "{{ if %s }}"
	cond__unless = "{{ if not %s }}"
	cond__case   = "{{/* switch %s */}}"
	cond__while  = "{{ range %s }}"
	cond__for    = "{{/* %s, %s */}}{{ range %s }}"
	cond__end    = "{{ end }}"

	cond__for_if   = "{{ if gt len %s 0 }}{{/* %s, %s */}}{{ range %s }}"
	code__for_else = "{{ end }}{{ else }}"

	code__longcode  = "{{/* %s */}}"
	code__buffered  = "{{ %s }}"
	code__unescaped = "{{ %s }}"
	code__else      = "{{ else }}"
	code__else_if   = "{{ else if %s }}"
	code__case_when = "{{/* case %s: */}}"
	code__case_def  = "{{/* default: */}}"
	code__mix_block = "{{/* block */}}"

	text__str     = "%s"
	text__comment = "<!--%s -->"

	mixin__bgn           = "\n%s"
	mixin__end           = ""
	mixin__var_bgn       = ""
	mixin__var           = "{{ $%s := %s }}"
	mixin__var_rest      = "{{ $%s := %#v }}"
	mixin__var_end       = "\n"
	mixin__var_block_bgn = ""
	mixin__var_block     = ""
	mixin__var_block_end = ""
)

type itemType int8

const (
	itemError itemType = iota // error occurred; value is text of error
	itemEOF

	itemEndL
	itemIdent
	itemEmptyLine // empty line

	itemText // plain text

	itemComment // html comment
	itemHTMLTag // html <tag>
	itemDoctype // Doctype tag

	itemDiv           // html div for . or #
	itemTag           // html tag
	itemTagInline     // inline tags
	itemTagEnd        // for <tag />
	itemTagVoid       // self-closing tags
	itemTagVoidInline // inline + self-closing tags

	itemID    // id    attribute
	itemClass // class attribute

	itemAttrStart
	itemAttrEnd
	itemAttr
	itemAttrSpace
	itemAttrComma
	itemAttrEqual
	itemAttrEqualUn

	itemFilter
	itemFilterText

	// itemKeyword // used only to delimit the keywords

	itemInclude
	itemExtends
	itemBlock
	itemBlockAppend
	itemBlockPrepend
	itemMixin
	itemMixinCall
	itemMixinBlock

	itemCode
	itemCodeBuffered
	itemCodeUnescaped

	itemIf
	itemElse
	itemElseIf
	itemUnless

	itemEach
	itemWhile
	itemFor
	itemForIfNotContain
	itemForElse

	itemCase
	itemCaseWhen
	itemCaseDefault
)

var key = map[string]itemType{
	"include": itemInclude,
	"extends": itemExtends,
	"block":   itemBlock,
	"append":  itemBlockAppend,
	"prepend": itemBlockPrepend,
	"mixin":   itemMixin,

	"if":      itemIf,
	"else":    itemElse,
	"unless":  itemUnless,
	"for":     itemFor,
	"each":    itemEach,
	"while":   itemWhile,
	"case":    itemCase,
	"when":    itemCaseWhen,
	"default": itemCaseDefault,

	"doctype": itemDoctype,

	"a":       itemTagInline,
	"abbr":    itemTagInline,
	"acronym": itemTagInline,
	"b":       itemTagInline,
	"code":    itemTagInline,
	"em":      itemTagInline,
	"font":    itemTagInline,
	"i":       itemTagInline,
	"ins":     itemTagInline,
	"kbd":     itemTagInline,
	"map":     itemTagInline,
	"samp":    itemTagInline,
	"small":   itemTagInline,
	"span":    itemTagInline,
	"strong":  itemTagInline,
	"sub":     itemTagInline,
	"sup":     itemTagInline,

	"area":    itemTagVoid,
	"base":    itemTagVoid,
	"col":     itemTagVoid,
	"command": itemTagVoid,
	"embed":   itemTagVoid,
	"hr":      itemTagVoid,
	"input":   itemTagVoid,
	"keygen":  itemTagVoid,
	"link":    itemTagVoid,
	"meta":    itemTagVoid,
	"param":   itemTagVoid,
	"source":  itemTagVoid,
	"track":   itemTagVoid,
	"wbr":     itemTagVoid,

	"br":  itemTagVoidInline,
	"img": itemTagVoidInline,
}

// NodeType identifies the type of a parse tree node.
type NodeType int

// Type returns itself and provides an easy default implementation
// for embedding in a Node. Embedded in all non-trivial Nodes.
func (t NodeType) Type() NodeType {
	return t
}

const (
	NodeText NodeType = iota
	NodeList
	NodeTag
	NodeCode
	NodeCond
	NodeString
	NodeDoctype
	NodeMixin
	NodeBlock
)
