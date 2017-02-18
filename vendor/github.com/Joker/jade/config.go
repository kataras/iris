package jade

var (
	// Pretty print if true
	PrettyOutput = true

	// Output indent strring for pretty print
	OutputIndent = "    "

	// Tabulation size of parse file
	TabSize = 4

	// Left go-template delim
	LeftDelim = "{{"

	// Right go-template delim
	RightDelim = "}}"
)

const (
	nestIndent = true
	lineIndent = false

	tabComment  = "//-"
	htmlComment = "//"

	interDelim      = "#{"
	unEscInterDelim = "!{"
	rightInterDelim = "}"
)

var itemToStr = map[itemType]string{
	itemError:         "itemError",
	itemEOF:           "itemEOF",
	itemEndL:          "itemEndL",
	itemEndTag:        "itemEndTag",
	itemEndAttr:       "itemEndAttr",
	itemIdentSpace:    "itemIdentSpace",
	itemIdentTab:      "itemIdentTab",
	itemTag:           "itemTag",
	itemDiv:           "itemDiv",
	itemInlineTag:     "itemInlineTag",
	itemVoidTag:       "itemVoidTag",
	itemInlineVoidTag: "itemInlineVoidTag",
	itemComment:       "itemComment",
	itemID:            "itemID",
	itemClass:         "itemClass",
	itemAttr:          "itemAttr",
	itemAttrN:         "itemAttrN",
	itemAttrName:      "itemAttrName",
	itemAttrVoid:      "itemAttrVoid",
	itemParentIdent:   "itemParentIdent",
	itemChildIdent:    "itemChildIdent",
	itemText:          "itemText",
	itemEmptyLine:     "itemEmptyLine",
	itemInlineText:    "itemInlineText",
	itemHTMLTag:       "itemHTMLTag",
	itemDoctype:       "itemDoctype",
	itemBlank:         "itemBlank",
	itemFilter:        "itemFilter",
	itemAction:        "itemAction",
	itemActionEnd:     "itemActionEnd",
	itemInlineAction:  "itemInlineAction",
	itemExtends:       "itemExtends",
	itemDefine:        "itemDefine",
	itemBlock:         "itemBlock",
	itemElse:          "itemElse",
	itemEnd:           "itemEnd",
	itemIf:            "itemIf",
	itemRange:         "itemRange",
	itemNil:           "itemNil",
	itemTemplate:      "itemTemplate",
	itemWith:          "itemWith",
}

var key = map[string]itemType{
	"area":    itemVoidTag,
	"base":    itemVoidTag,
	"col":     itemVoidTag,
	"command": itemVoidTag,
	"embed":   itemVoidTag,
	"hr":      itemVoidTag,
	"input":   itemVoidTag,
	"keygen":  itemVoidTag,
	"link":    itemVoidTag,
	"meta":    itemVoidTag,
	"param":   itemVoidTag,
	"source":  itemVoidTag,
	"track":   itemVoidTag,
	"wbr":     itemVoidTag,

	"template": itemAction,
	"end":      itemAction,
	"include":  itemAction,

	"extends": itemExtends,
	"block":   itemBlock,
	"mixin":   itemDefine,

	// "define": 	itemDefine,
	// "block":  itemActionEnd,
	"define": itemActionEnd,

	"if":     itemActionEnd,
	"else":   itemActionEnd,
	"range":  itemActionEnd,
	"with":   itemActionEnd,
	"each":   itemActionEnd,
	"for":    itemActionEnd,
	"while":  itemActionEnd,
	"unless": itemActionEnd,
	"case":   itemActionEnd,

	// "if": 		itemIf,
	// "else": 		itemElse,
	// "end": 		itemEnd,
	// "range": 	itemRange,
	// "with": 		itemWith,
	// "nil": 		itemNil,
	// "template": 	itemTemplate,

	"a":       itemInlineTag,
	"abbr":    itemInlineTag,
	"acronym": itemInlineTag,
	"b":       itemInlineTag,
	"code":    itemInlineTag,
	"em":      itemInlineTag,
	"font":    itemInlineTag,
	"i":       itemInlineTag,
	"ins":     itemInlineTag,
	"kbd":     itemInlineTag,
	"map":     itemInlineTag,
	"samp":    itemInlineTag,
	"small":   itemInlineTag,
	"span":    itemInlineTag,
	"strong":  itemInlineTag,
	"sub":     itemInlineTag,
	"sup":     itemInlineTag,

	"br":  itemInlineVoidTag,
	"img": itemInlineVoidTag,
}
