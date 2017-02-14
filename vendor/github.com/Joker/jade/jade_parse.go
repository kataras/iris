package jade

import (
	"strings"
)

func (t *tree) parse(treeSet map[string]*tree) (next node) {
	token := t.next()
	t.Root = t.newList(token.pos)

	for token.typ != itemEOF {

		switch token.typ {
		case itemError:
			t.errorf("%s", token.val)
		case itemDoctype:
			t.Root.append(t.newDoctype(token.pos, token.val))
		case itemHTMLTag:
			t.Root.append(t.newLine(token.pos, token.val, token.typ, 0, 0))

		case itemTag, itemDiv, itemInlineTag, itemAction, itemActionEnd, itemDefine, itemBlock, itemComment:
			nest := t.newNest(token.pos, token.val, token.typ, 0, 0)
			if token.typ < itemComment {
				t.parseAttr(nest)
			}
			t.Root.append(nest)
			t.parseInside(nest)

		case itemBlank:
			nest := t.newNest(token.pos, token.val, token.typ, 0, 0)
			t.parseInside(nest)
		}

		token = t.next()

	}
	return nil
}

func (t *tree) parseInside(outTag *nestNode) int {
	indentCount := 0
	token := t.next()

	for token.typ != itemEOF {
		switch token.typ {
		case itemError:
			t.errorf("%s", token.val)

		case itemEndL:
			indentCount = 0
		case itemIdentSpace:
			indentCount++
		case itemIdentTab:
			indentCount += TabSize
		case itemParentIdent:
			indentCount = outTag.Indent + 1 // for  "tag: tag: tag"
		case itemChildIdent:
			return indentCount // for  "]"

		case itemDoctype:
			t.backup()
			return indentCount
		case itemInlineText, itemInlineAction:
			outTag.append(t.newLine(token.pos, token.val, token.typ, indentCount, outTag.Nesting+1))

		case itemHTMLTag, itemText:
			if indentCount > outTag.Indent {
				outTag.append(t.newLine(token.pos, token.val, token.typ, indentCount, outTag.Nesting+1))
			} else {
				t.backup()
				return indentCount
			}

		case itemVoidTag, itemInlineVoidTag:
			if indentCount > outTag.Indent {
				nest := t.newNest(token.pos, token.val, token.typ, indentCount, outTag.Nesting+1)
				t.parseAttr(nest)
				outTag.append(nest)
			} else {
				t.backup()
				return indentCount
			}

		case itemTag, itemDiv, itemInlineTag:
			if indentCount > outTag.Indent {
				nest := t.newNest(token.pos, token.val, token.typ, indentCount, outTag.Nesting+1)
				if t.parseAttr(nest) {
					outTag.append(nest)
					indentCount = t.parseInside(nest)
				} else {
					nest.typ = itemVoidTag
					outTag.append(nest)
				}
			} else {
				t.backup()
				return indentCount
			}

		case itemAction, itemActionEnd, itemTemplate, itemDefine, itemBlock:
			if indentCount > outTag.Indent {
				nest := t.newNest(token.pos, token.val, token.typ, indentCount, outTag.Nesting+1)
				outTag.append(nest)
				indentCount = t.parseInside(nest)
				if strings.HasPrefix(nest.Tag, "if") || strings.HasPrefix(nest.Tag, "with") {
					action := t.next()
					if strings.HasPrefix(action.val, "else") {
						nest.typ = itemAction
					}
					t.backup()
				}
			} else {
				t.backup()
				return indentCount
			}

		case itemBlank, itemComment:
			if indentCount > outTag.Indent {
				nest := t.newNest(token.pos, token.val, token.typ, indentCount, outTag.Nesting+1)
				if token.typ == itemComment {
					outTag.append(nest)
				}
				indentCount = t.parseInside(nest)
			} else {
				t.backup()
				return indentCount
			}

		}
		token = t.next()
	}
	t.backup()
	return indentCount
}

func (t *tree) parseAttr(currentTag *nestNode) bool {
	for {
		attr := t.next()
		switch attr.typ {
		case itemError:
			t.errorf("%s", attr.val)
		case itemID:
			if len(currentTag.id) > 0 {
				t.errorf("unexpected second id \"%s\" ", attr.val)
			}
			currentTag.id = attr.val
		case itemClass:
			currentTag.class = append(currentTag.class, attr.val)
		case itemAttr, itemAttrN, itemAttrName, itemAttrVoid:
			currentTag.append(t.newAttr(attr.pos, attr.val, attr.typ))
		case itemEndTag:
			currentTag.append(t.newAttr(attr.pos, "/>", itemEndTag))
			return false
		default:
			t.backup()
			currentTag.append(t.newAttr(attr.pos, ">", itemEndAttr))
			return true
		}
	}
}
