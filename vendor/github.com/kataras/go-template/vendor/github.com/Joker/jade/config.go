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
	itemEndAttr:       "itemEndAttr",
	itemIdentSpace:    "itemIdentSpace",
	itemIdentTab:      "itemIdentTab",
	itemTag:           "itemTag",
	itemVoidTag:       "itemVoidTag",
	itemInlineTag:     "itemInlineTag",
	itemInlineVoidTag: "itemInlineVoidTag",
	itemHTMLTag:       "itemHTMLTag",
	itemDiv:           "itemDiv",
	itemID:            "itemID",
	itemClass:         "itemClass",
	itemAttr:          "itemAttr",
	itemAttrN:         "itemAttrN",
	itemAttrName:      "itemAttrName",
	itemAttrVoid:      "itemAttrVoid",
	itemAction:        "itemAction",
	itemInlineAction:  "itemInlineAction",
	itemInlineText:    "itemInlineText",
	itemFilter:        "itemFilter",
	itemDoctype:       "itemDoctype",
	itemComment:       "itemComment",
	itemBlank:         "itemBlank",
	itemParentIdent:   "itemParentIdent",
	itemChildIdent:    "itemChildIdent",
	itemText:          "itemText",
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
	"extends":  itemAction,

	"if":     itemActionEnd,
	"else":   itemActionEnd,
	"range":  itemActionEnd,
	"with":   itemActionEnd,
	"block":  itemActionEnd,
	"define": itemActionEnd,
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
	// "define": 	itemDefine,
	"mixin": itemDefine,

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
