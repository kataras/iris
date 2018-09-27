package jade

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

func (t *Tree) topParse() {
	t.Root = t.newList(t.peek().pos)
	var (
		ext   bool
		token = t.nextNonSpace()
	)
	if token.typ == itemExtends {
		ext = true
		t.Root.append(t.parseSubFile(token.val))
		token = t.nextNonSpace()
	}
	for {
		switch token.typ {
		case itemInclude:
			t.Root.append(t.parseInclude(token))
		case itemBlock, itemBlockPrepend, itemBlockAppend:
			if ext {
				t.parseBlock(token)
			} else {
				t.Root.append(t.parseBlock(token))
			}
		case itemMixin:
			t.mixin[token.val] = t.parseMixin(token)
		case itemEOF:
			return
		case itemExtends:
			t.errorf(`Declaration of template inheritance ("extends") should be the first thing in the file. There can only be one extends statement per file.`)
		case itemError:
			t.errorf("%s line: %d\n", token.val, token.line)
		default:
			if ext {
				t.errorf(`Only import, named blocks and mixins can appear at the top level of an extending template`)
			}
			t.Root.append(t.hub(token))
		}
		token = t.nextNonSpace()
	}
}

func (t *Tree) hub(token item) (n Node) {
	for {
		switch token.typ {
		case itemDiv:
			token.val = "div"
			fallthrough
		case itemTag, itemTagInline, itemTagVoid, itemTagVoidInline:
			return t.parseTag(token)
		case itemText, itemComment, itemHTMLTag:
			return t.newText(token.pos, token.val, token.typ)
		case itemCode, itemCodeBuffered, itemCodeUnescaped, itemMixinBlock:
			return t.newCode(token.pos, token.val, token.typ)
		case itemIf, itemUnless:
			return t.parseIf(token)
		case itemFor, itemEach, itemWhile:
			return t.parseFor(token)
		case itemCase:
			return t.parseCase(token)
		case itemBlock, itemBlockPrepend, itemBlockAppend:
			return t.parseBlock(token)
		case itemMixinCall:
			return t.parseMixinUse(token)
		case itemInclude:
			return t.parseInclude(token)
		case itemDoctype:
			return t.newDoctype(token.pos, token.val)
		case itemFilter, itemFilterText:
			return t.parseFilter(token)
		case itemError:
			t.errorf("Error lex: %s line: %d\n", token.val, token.line)
		default:
			t.errorf(`Error hub(): unexpected token  "%s"  type  "%s"`, token.val, token.typ)
		}
	}
}

func (t *Tree) parseFilter(tk item) Node {
	// TODO add golang filters
	return t.newList(tk.pos)
}

func (t *Tree) parseTag(tk item) Node {
	var (
		deep = tk.depth
		tag  = t.newTag(tk.pos, tk.val, tk.typ)
	)
Loop:
	for {
		switch token := t.nextNonSpace(); {
		case token.depth > deep:
			if tag.tagType == itemTagVoid || tag.tagType == itemTagVoidInline {
				break Loop
			}
			t.tab++
			tag.append(t.hub(token))
			t.tab--
		case token.depth == deep:
			switch token.typ {
			case itemClass:
				tag.attr("class", `"`+token.val+`"`)
			case itemID:
				tag.attr("id", `"`+token.val+`"`)
			case itemAttrStart:
				t.parseAttributes(tag)
			case itemTagEnd:
				tag.tagType = itemTagVoid
				return tag
			default:
				break Loop
			}
		default:
			break Loop
		}
	}
	t.backup()
	return tag
}

type pAttr interface {
	attr(string, string)
}

func (t *Tree) parseAttributes(tag pAttr) {
	var (
		aname string
		equal bool
		unesc bool
		stack = make([]string, 0, 4)
	)
	for {
		switch token := t.next(); token.typ {
		case itemAttrSpace:
			// skip
		case itemAttr:
			switch {
			case aname == "":
				aname = token.val
			case aname != "" && !equal:
				tag.attr(aname, `"`+aname+`"`)
				aname = token.val
			case aname != "" && equal:
				if unesc {
					stack = append(stack, "ߐ"+token.val)
					unesc = false
				} else {
					stack = append(stack, token.val)
				}
			}
		case itemAttrEqualUn:
			unesc = true
			fallthrough
		case itemAttrEqual:
			equal = true
			switch len_stack := len(stack); {
			case len_stack == 0 && aname != "":
				// skip
			case len_stack > 1 && aname != "":
				tag.attr(aname, strings.Join(stack[:len(stack)-1], " "))

				aname = stack[len(stack)-1]
				stack = stack[:0]
			case len_stack == 1 && aname == "":
				aname = stack[0]
				stack = stack[:0]
			default:
				t.errorf("unexpected '='")
			}
		case itemAttrComma:
			equal = false
			switch len_stack := len(stack); {
			case len_stack > 0 && aname != "":
				tag.attr(aname, strings.Join(stack, " "))
				aname = ""
				stack = stack[:0]
			case len_stack == 0 && aname != "":
				tag.attr(aname, `"`+aname+`"`)
				aname = ""
			}
		case itemAttrEnd:
			switch len_stack := len(stack); {
			case len_stack > 0 && aname != "":
				tag.attr(aname, strings.Join(stack, " "))
			case len_stack > 0 && aname == "":
				for _, a := range stack {
					tag.attr(a, a)
				}
			case len_stack == 0 && aname != "":
				tag.attr(aname, `"`+aname+`"`)
			}
			return
		default:
			t.errorf("unexpected %s", token.val)
		}
	}
}

func (t *Tree) parseIf(tk item) Node {
	var (
		deep = tk.depth
		cond = t.newCond(tk.pos, tk.val, tk.typ)
	)
Loop:
	for {
		switch token := t.nextNonSpace(); {
		case token.depth > deep:
			t.tab++
			cond.append(t.hub(token))
			t.tab--
		case token.depth == deep:
			switch token.typ {
			case itemElse:
				ni := t.peek()
				if ni.typ == itemIf {
					token = t.next()
					cond.append(t.newCode(token.pos, token.val, itemElseIf))
				} else {
					cond.append(t.newCode(token.pos, token.val, token.typ))
				}
			default:
				break Loop
			}
		default:
			break Loop
		}
	}
	t.backup()
	return cond
}

func (t *Tree) parseFor(tk item) Node {
	var (
		deep = tk.depth
		cond = t.newCond(tk.pos, tk.val, tk.typ)
	)
Loop:
	for {
		switch token := t.nextNonSpace(); {
		case token.depth > deep:
			t.tab++
			cond.append(t.hub(token))
			t.tab--
		case token.depth == deep:
			if token.typ == itemElse {
				cond.condType = itemForIfNotContain
				cond.append(t.newCode(token.pos, token.val, itemForElse))
			} else {
				break Loop
			}
		default:
			break Loop
		}
	}
	t.backup()
	return cond
}

func (t *Tree) parseCase(tk item) Node {
	var (
		deep   = tk.depth
		_case_ = t.newCond(tk.pos, tk.val, tk.typ)
	)
	for {
		if token := t.nextNonSpace(); token.depth > deep {
			switch token.typ {
			case itemCaseWhen, itemCaseDefault:
				_case_.append(t.newCode(token.pos, token.val, token.typ))
			default:
				t.tab++
				_case_.append(t.hub(token))
				t.tab--
			}
		} else {
			break
		}
	}
	t.backup()
	return _case_
}

func (t *Tree) parseMixin(tk item) *MixinNode {
	var (
		deep  = tk.depth
		mixin = t.newMixin(tk.pos)
	)
Loop:
	for {
		switch token := t.nextNonSpace(); {
		case token.depth > deep:
			t.tab++
			mixin.append(t.hub(token))
			t.tab--
		case token.depth == deep:
			if token.typ == itemAttrStart {
				t.parseAttributes(mixin)
			} else {
				break Loop
			}
		default:
			break Loop
		}
	}
	t.backup()
	return mixin
}

func (t *Tree) parseMixinUse(tk item) Node {
	tMix, ok := t.mixin[tk.val]
	if !ok {
		t.errorf(`Mixin "%s" must be declared before use.`, tk.val)
	}
	var (
		deep  = tk.depth
		mixin = tMix.CopyMixin()
	)
Loop:
	for {
		switch token := t.nextNonSpace(); {
		case token.depth > deep:
			t.tab++
			mixin.append(t.hub(token))
			t.tab--
		case token.depth == deep:
			if token.typ == itemAttrStart {
				t.parseAttributes(mixin)
			} else {
				break Loop
			}
		default:
			break Loop
		}
	}
	t.backup()

	use := len(mixin.AttrName)
	tpl := len(tMix.AttrName)
	switch {
	case use < tpl:
		i := 0
		diff := tpl - use
		mixin.AttrCode = append(mixin.AttrCode, make([]string, diff)...) // Extend slice
		for index := 0; index < diff; index++ {
			i = tpl - index - 1
			if tMix.AttrName[i] != tMix.AttrCode[i] {
				mixin.AttrCode[i] = tMix.AttrCode[i]
			} else {
				mixin.AttrCode[i] = `""`
			}
		}
		mixin.AttrName = tMix.AttrName
	case use > tpl:
		if tpl <= 0 {
			break
		}
		if strings.HasPrefix(tMix.AttrName[tpl-1], "...") {
			mixin.AttrRest = mixin.AttrCode[tpl-1:]
		}
		mixin.AttrCode = mixin.AttrCode[:tpl]
		mixin.AttrName = tMix.AttrName
	case use == tpl:
		mixin.AttrName = tMix.AttrName
	}
	return mixin
}

func (t *Tree) parseBlock(tk item) *BlockNode {
	block := t.newList(tk.pos)
	for {
		token := t.nextNonSpace()
		if token.depth > tk.depth {
			block.append(t.hub(token))
		} else {
			break
		}
	}
	t.backup()
	var suf string
	switch tk.typ {
	case itemBlockPrepend:
		suf = "_prepend"
	case itemBlockAppend:
		suf = "_append"
	}
	t.block[tk.val+suf] = block
	return t.newBlock(tk.pos, tk.val, tk.typ)
}

func (t *Tree) parseInclude(tk item) *ListNode {
	switch ext := filepath.Ext(tk.val); ext {
	case ".jade", ".pug", "":
		return t.parseSubFile(tk.val)
	case ".js", ".css", ".tpl", ".md":
		ln := t.newList(tk.pos)
		ln.append(t.newText(tk.pos, t.read(tk.val), itemText))
		return ln
	default:
		t.errorf(`file extension is not supported`)
		return nil
	}
}

func (t *Tree) parseSubFile(path string) *ListNode {
	var incTree = New(path)
	incTree.tab = t.tab
	incTree.block = t.block
	incTree.mixin = t.mixin
	_, err := incTree.Parse(t.read(path))
	if err != nil {
		t.errorf(`%s`, err)
	}
	return incTree.Root
}

func (t *Tree) read(path string) string {
	var (
		bb  []byte
		ext string
		err error
	)
	switch ext = filepath.Ext(path); ext {
	case ".jade", ".pug", ".js", ".css", ".tpl", ".md":
		bb, err = ioutil.ReadFile(path)
	case "":
		if _, err = os.Stat(path + ".jade"); os.IsNotExist(err) {
			if _, err = os.Stat(path + ".pug"); os.IsNotExist(err) {
				t.errorf(`".jade" or ".pug" file required`)
			} else {
				ext = ".pug"
			}
		} else {
			ext = ".jade"
		}
		bb, err = ioutil.ReadFile(path + ext)
	default:
		t.errorf(`file extension  %s  is not supported`, ext)
	}
	if err != nil {
		dir, _ := os.Getwd()
		t.errorf(`%s  work dir: %s `, err, dir)
	}

	return string(bb)
}
