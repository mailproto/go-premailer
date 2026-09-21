package premailer

import (
	"bytes"

	"golang.org/x/net/html"
)

// HTML parsing and serialization matched to htmlparser2 in non-XML mode, which
// is what juice parses with (cheerio is loaded with decodeEntities:false).
//
// This deliberately skips HTML5 tree construction: no implied <tbody>, no
// <html>/<head>/<body> synthesis, no foster parenting, no reconstruction of
// active formatting elements. Text and attribute values are kept as raw,
// undecoded bytes so the serializer can write them back byte for byte --
// x/net/html's Parse+Render would turn &nbsp; into U+00A0 and re-escape & in
// URLs, both of which a mail client can see.

type nodeType uint8

const (
	nodeElement nodeType = iota
	nodeText
	nodeComment
	nodeDoctype
)

type attribute struct {
	name  string
	value []byte // raw and undecoded; empty serializes as a bare attribute
}

type node struct {
	typ    nodeType
	tag    string // lowercased
	attrs  []attribute
	raw    []byte // text, comment and doctype content, including delimiters
	parent *node
	kids   []*node
	// foreign marks svg/math and their descendants. dom-serializer closes any
	// childless element in foreign content, so <g></g> renders as <g/>.
	foreign bool
}

func (n *node) add(c *node) {
	c.parent = n
	n.kids = append(n.kids, c)
}

func (n *node) attr(name string) ([]byte, bool) {
	for i := range n.attrs {
		if n.attrs[i].name == name {
			return n.attrs[i].value, true
		}
	}
	return nil, false
}

func (n *node) setAttr(name string, v []byte) {
	for i := range n.attrs {
		if n.attrs[i].name == name {
			n.attrs[i].value = v
			return
		}
	}
	n.attrs = append(n.attrs, attribute{name: name, value: v})
}

func (n *node) removeAttr(name string) {
	for i := range n.attrs {
		if n.attrs[i].name == name {
			n.attrs = append(n.attrs[:i], n.attrs[i+1:]...)
			return
		}
	}
}

// remove detaches n from its parent. Safe to call while ranging over a
// separately collected slice; never while walking kids directly.
func (n *node) remove() {
	p := n.parent
	if p == nil {
		return
	}
	for i, k := range p.kids {
		if k == n {
			p.kids = append(p.kids[:i], p.kids[i+1:]...)
			n.parent = nil
			return
		}
	}
}

var voidElements = map[string]bool{
	"area": true, "base": true, "basefont": true, "br": true, "col": true,
	"command": true, "embed": true, "frame": true, "hr": true, "img": true,
	"input": true, "isindex": true, "keygen": true, "link": true, "meta": true,
	"param": true, "source": true, "track": true, "wbr": true,
}

// openImpliesClose mirrors htmlparser2's table: opening the key tag pops the
// stack while the element on top is in the value set. Note this tests only the
// top of the stack, which is why <ul><li>a<ul><li>b</ul> nests rather than
// closing the outer <li>.
var openImpliesClose = map[string]map[string]bool{
	"tr":       {"tr": true, "th": true, "td": true},
	"th":       {"th": true},
	"td":       {"thead": true, "th": true, "td": true},
	"body":     {"head": true, "link": true, "script": true},
	"li":       {"li": true},
	"option":   {"option": true},
	"optgroup": {"optgroup": true, "option": true},
	"dd":       {"dt": true, "dd": true},
	"dt":       {"dt": true, "dd": true},
}

var formTags = map[string]bool{
	"input": true, "option": true, "optgroup": true, "select": true,
	"button": true, "datalist": true, "textarea": true,
}

var pTag = map[string]bool{"p": true}

func init() {
	for _, t := range []string{"select", "input", "output", "button", "datalist", "textarea"} {
		openImpliesClose[t] = formTags
	}
	for _, t := range []string{
		"p", "h1", "h2", "h3", "h4", "h5", "h6",
		"address", "article", "aside", "blockquote", "details", "div", "dl",
		"fieldset", "figcaption", "figure", "footer", "form", "header",
		"hgroup", "hr", "main", "menu", "nav", "ol", "pre", "section",
		"table", "ul",
	} {
		openImpliesClose[t] = pTag
	}
}

// reparseRaw are tags x/net/html's tokenizer reports as raw text but
// htmlparser2 parses as markup. Their contents get a second pass.
var reparseRaw = map[string]bool{
	"noscript": true, "iframe": true, "noembed": true, "noframes": true,
}

func isForeign(tag string) bool { return tag == "svg" || tag == "math" }

// parseDocument builds an htmlparser2-shaped tree. The returned root is a
// synthetic container; its kids are the document's top-level nodes.
func parseDocument(src []byte) *node {
	root := &node{typ: nodeElement, tag: "#document"}
	// One arena for every raw span. Appends may reallocate, but earlier bytes
	// are never mutated, so slices handed out before a growth stay valid.
	arena := make([]byte, 0, len(src))
	keep := func(b []byte) []byte {
		n := len(arena)
		arena = append(arena, b...)
		return arena[n:]
	}

	cur := root
	foreign := 0
	z := html.NewTokenizer(bytes.NewReader(src))
	for {
		tt := z.Next()
		if tt == html.ErrorToken {
			break
		}
		// z.Raw() points into the tokenizer's buffer, which it compacts and
		// reallocates as it reads, so every span must be copied out now.
		raw := keep(z.Raw())

		switch tt {
		case html.TextToken:
			cur.add(&node{typ: nodeText, raw: raw})

		case html.CommentToken:
			cur.add(&node{typ: nodeComment, raw: raw})

		case html.DoctypeToken:
			cur.add(&node{typ: nodeDoctype, raw: raw})

		case html.StartTagToken, html.SelfClosingTagToken:
			name, attrs := scanStartTag(raw)
			if name == "" {
				continue
			}
			if set := openImpliesClose[name]; set != nil {
				for cur != root && set[cur.tag] {
					cur = cur.parent
				}
			}
			n := &node{typ: nodeElement, tag: name, attrs: attrs,
				foreign: foreign > 0 || isForeign(name)}
			cur.add(n)
			if voidElements[name] {
				continue
			}
			// In non-XML mode htmlparser2 ignores the solidus outside foreign
			// content, so <div/> opens a div rather than closing it.
			if tt == html.SelfClosingTagToken && n.foreign {
				continue
			}
			cur = n
			if isForeign(name) {
				foreign++
			}

		case html.EndTagToken:
			name := scanEndTag(raw)
			if name == "" {
				continue
			}
			open := false
			for p := cur; p != root; p = p.parent {
				if p.tag == name {
					open = true
					break
				}
			}
			if open {
				for cur.tag != name {
					if isForeign(cur.tag) {
						foreign--
					}
					cur = cur.parent
				}
				if isForeign(cur.tag) {
					foreign--
				}
				if reparseRaw[cur.tag] {
					reparseChildren(cur)
				}
				cur = cur.parent
			} else if name == "br" || name == "p" {
				// htmlparser2 turns an unmatched </br> into <br> and an
				// unmatched </p> into an empty <p></p>.
				cur.add(&node{typ: nodeElement, tag: name})
			}
			// Any other unmatched end tag is dropped.
		}
	}
	// Anything still open at EOF stays open; the serializer closes it.
	return root
}

// reparseChildren re-parses the raw text of an element the tokenizer treated as
// raw text but htmlparser2 would have parsed as markup.
func reparseChildren(n *node) {
	if len(n.kids) != 1 || n.kids[0].typ != nodeText {
		return
	}
	inner := parseDocument(n.kids[0].raw)
	n.kids = n.kids[:0]
	for _, k := range inner.kids {
		n.add(k)
	}
}

func isSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\f'
}

func lowerASCII(b []byte) string {
	needs := false
	for _, c := range b {
		if 'A' <= c && c <= 'Z' {
			needs = true
			break
		}
	}
	if !needs {
		return string(b)
	}
	out := make([]byte, len(b))
	for i, c := range b {
		if 'A' <= c && c <= 'Z' {
			c += 'a' - 'A'
		}
		out[i] = c
	}
	return string(out)
}

// scanStartTag reads a tag name and attributes out of raw start-tag bytes,
// keeping values exactly as written. x/net/html's TagAttr would decode
// entities, which loses the distinction between &amp; and a literal &.
func scanStartTag(raw []byte) (string, []attribute) {
	i := 1 // skip '<'
	start := i
	for i < len(raw) && !isSpace(raw[i]) && raw[i] != '>' && raw[i] != '/' {
		i++
	}
	name := lowerASCII(raw[start:i])
	if name == "" {
		return "", nil
	}

	var attrs []attribute
	for i < len(raw) {
		for i < len(raw) && (isSpace(raw[i]) || raw[i] == '/') {
			i++
		}
		if i >= len(raw) || raw[i] == '>' {
			break
		}
		ns := i
		for i < len(raw) && !isSpace(raw[i]) && raw[i] != '=' && raw[i] != '>' && raw[i] != '/' {
			i++
		}
		an := lowerASCII(raw[ns:i])
		for i < len(raw) && isSpace(raw[i]) {
			i++
		}
		var val []byte
		if i < len(raw) && raw[i] == '=' {
			i++
			for i < len(raw) && isSpace(raw[i]) {
				i++
			}
			if i < len(raw) && (raw[i] == '"' || raw[i] == '\'') {
				q := raw[i]
				i++
				vs := i
				for i < len(raw) && raw[i] != q {
					i++
				}
				val = raw[vs:i]
				if i < len(raw) {
					i++
				}
			} else {
				vs := i
				for i < len(raw) && !isSpace(raw[i]) && raw[i] != '>' {
					i++
				}
				val = raw[vs:i]
			}
		}
		if an == "" {
			continue
		}
		// htmlparser2 keeps the first occurrence of a repeated attribute.
		dup := false
		for k := range attrs {
			if attrs[k].name == an {
				dup = true
				break
			}
		}
		if !dup {
			attrs = append(attrs, attribute{name: an, value: val})
		}
	}
	return name, attrs
}

func scanEndTag(raw []byte) string {
	i := 2 // skip '</'
	if i > len(raw) {
		return ""
	}
	start := i
	for i < len(raw) && !isSpace(raw[i]) && raw[i] != '>' {
		i++
	}
	return lowerASCII(raw[start:i])
}

// render serializes the tree the way dom-serializer does with
// decodeEntities:false: text is written untouched and only the double quote is
// escaped inside attribute values.
func render(buf *bytes.Buffer, n *node) {
	switch n.typ {
	case nodeText, nodeDoctype:
		buf.Write(n.raw)
		return
	case nodeComment:
		// htmlparser2 has no CDATA section outside XML mode, so it reports
		// one as a bogus comment and serializes it as a comment. Other bogus
		// constructs (<!decl>, <?pi?>) pass through untouched.
		if bytes.HasPrefix(n.raw, []byte("<![CDATA[")) && bytes.HasSuffix(n.raw, []byte(">")) {
			buf.WriteString("<!--")
			buf.Write(n.raw[2 : len(n.raw)-1])
			buf.WriteString("-->")
			return
		}
		buf.Write(n.raw)
		return
	}

	if n.tag != "#document" {
		buf.WriteByte('<')
		buf.WriteString(n.tag)
		for i := range n.attrs {
			buf.WriteByte(' ')
			buf.WriteString(n.attrs[i].name)
			if v := n.attrs[i].value; len(v) > 0 {
				buf.WriteString(`="`)
				writeAttrValue(buf, v)
				buf.WriteByte('"')
			}
		}
		// Empty elements in foreign content (svg, math) close themselves.
		if n.foreign && len(n.kids) == 0 {
			buf.WriteString("/>")
			return
		}
		buf.WriteByte('>')
		if voidElements[n.tag] {
			return
		}
	}

	for _, k := range n.kids {
		render(buf, k)
	}

	if n.tag != "#document" {
		buf.WriteString("</")
		buf.WriteString(n.tag)
		buf.WriteByte('>')
	}
}

func writeAttrValue(buf *bytes.Buffer, v []byte) {
	for {
		i := bytes.IndexByte(v, '"')
		if i < 0 {
			buf.Write(v)
			return
		}
		buf.Write(v[:i])
		buf.WriteString("&quot;")
		v = v[i+1:]
	}
}

func renderDocument(n *node) []byte {
	var buf bytes.Buffer
	render(&buf, n)
	return buf.Bytes()
}
