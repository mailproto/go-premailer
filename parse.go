package premailer

import (
	"bytes"
	"strings"

	"golang.org/x/net/html"
)

// HTML parsing and serialization matched to htmlparser2 in non-XML mode, which
// is what juice parses with (cheerio is loaded with decodeEntities:false).
//
// This deliberately skips HTML5 tree construction: no implied <tbody>, no
// <html>/<head>/<body> synthesis, no foster parenting, no reconstruction of
// active formatting elements. x/net/html's own Parse would do all of those and
// would decode entities besides, and a mail client can see the difference.
//
// The tree is built out of *html.Node so cascadia can match against it
// directly, but it is populated by hand and differs from what html.Parse
// produces in two ways worth knowing:
//
//   - Attr[i].Val and text hold RAW, UNDECODED source bytes. "&nbsp;" stays
//     "&nbsp;". This is also what cheerio matches against, so selectors behave
//     the same.
//   - Comment and Doctype nodes keep their full source text in Data,
//     delimiters included, because bogus constructs like <!decl> and <?pi?>
//     have to round-trip verbatim.
//
// Namespace is set to "svg" or "math" on foreign content, which the serializer
// uses to decide self-closing.

var voidElements = map[string]bool{
	"area": true, "base": true, "basefont": true, "br": true, "col": true,
	"command": true, "embed": true, "frame": true, "hr": true, "img": true,
	"input": true, "isindex": true, "keygen": true, "link": true, "meta": true,
	"param": true, "source": true, "track": true, "wbr": true,
}

// openImpliesClose mirrors htmlparser2's table: opening the key tag pops the
// stack while the element on top is in the value set. It tests only the top of
// the stack, which is why <ul><li>a<ul><li>b</ul> nests rather than closing
// the outer <li>.
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

func init() {
	for _, t := range []string{"select", "input", "output", "button", "datalist", "textarea"} {
		openImpliesClose[t] = formTags
	}
	pTag := map[string]bool{"p": true}
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
// htmlparser2 parses as markup, so their contents get a second pass.
var reparseRaw = map[string]bool{
	"noscript": true, "iframe": true, "noembed": true, "noframes": true,
}

func isForeign(tag string) bool { return tag == "svg" || tag == "math" }

// parseDocument builds an htmlparser2-shaped tree. The returned root is a
// synthetic container whose children are the document's top-level nodes.
func parseDocument(src []byte) *html.Node {
	root := &html.Node{Type: html.DocumentNode}
	cur := root
	foreign := 0
	z := html.NewTokenizer(bytes.NewReader(src))

	for {
		tt := z.Next()
		if tt == html.ErrorToken {
			break
		}
		// z.Raw() points into the tokenizer's buffer, which it compacts and
		// reallocates as it reads, so spans must be copied out now.
		raw := string(z.Raw())

		switch tt {
		case html.TextToken:
			cur.AppendChild(&html.Node{Type: html.TextNode, Data: raw})

		case html.CommentToken:
			cur.AppendChild(&html.Node{Type: html.CommentNode, Data: raw})

		case html.DoctypeToken:
			cur.AppendChild(&html.Node{Type: html.DoctypeNode, Data: raw})

		case html.StartTagToken, html.SelfClosingTagToken:
			name, attrs := scanStartTag(raw)
			if name == "" {
				continue
			}
			if set := openImpliesClose[name]; set != nil {
				for cur != root && set[cur.Data] {
					cur = cur.Parent
				}
			}
			n := &html.Node{Type: html.ElementNode, Data: name, Attr: attrs}
			if foreign > 0 || isForeign(name) {
				n.Namespace = "svg"
			}
			cur.AppendChild(n)
			if voidElements[name] {
				continue
			}
			// Outside foreign content htmlparser2 ignores the solidus, so
			// <div/> opens a div rather than closing it.
			if tt == html.SelfClosingTagToken && n.Namespace != "" {
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
			for p := cur; p != root; p = p.Parent {
				if p.Data == name {
					open = true
					break
				}
			}
			if open {
				for cur.Data != name {
					if isForeign(cur.Data) {
						foreign--
					}
					cur = cur.Parent
				}
				if isForeign(cur.Data) {
					foreign--
				}
				if reparseRaw[cur.Data] {
					reparseChildren(cur)
				}
				cur = cur.Parent
			} else if name == "br" || name == "p" {
				// htmlparser2 turns an unmatched </br> into <br> and an
				// unmatched </p> into an empty <p></p>.
				cur.AppendChild(&html.Node{Type: html.ElementNode, Data: name})
			}
			// Any other unmatched end tag is dropped.
		}
	}
	// Anything still open at EOF stays open; the serializer closes it.
	return root
}

// reparseChildren re-parses an element the tokenizer treated as raw text but
// htmlparser2 would have parsed as markup.
func reparseChildren(n *html.Node) {
	if n.FirstChild == nil || n.FirstChild != n.LastChild || n.FirstChild.Type != html.TextNode {
		return
	}
	text := n.FirstChild.Data
	n.RemoveChild(n.FirstChild)
	inner := parseDocument([]byte(text))
	for c := inner.FirstChild; c != nil; {
		next := c.NextSibling
		inner.RemoveChild(c)
		n.AppendChild(c)
		c = next
	}
}

func isSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\f'
}

func lowerASCII(b []byte) string {
	for _, c := range b {
		if 'A' <= c && c <= 'Z' {
			out := make([]byte, len(b))
			for i, c := range b {
				if 'A' <= c && c <= 'Z' {
					c += 'a' - 'A'
				}
				out[i] = c
			}
			return string(out)
		}
	}
	return string(b)
}

// scanStartTag reads a tag name and attributes from raw start-tag bytes,
// keeping values exactly as written. x/net/html's TagAttr decodes entities,
// which loses the difference between "&amp;" and a literal "&".
func scanStartTag(raw string) (string, []html.Attribute) {
	b := []byte(raw)
	i := 1 // skip '<'
	start := i
	for i < len(b) && !isSpace(b[i]) && b[i] != '>' && b[i] != '/' {
		i++
	}
	name := lowerASCII(b[start:i])
	if name == "" {
		return "", nil
	}

	var attrs []html.Attribute
	for i < len(b) {
		for i < len(b) && (isSpace(b[i]) || b[i] == '/') {
			i++
		}
		if i >= len(b) || b[i] == '>' {
			break
		}
		ns := i
		for i < len(b) && !isSpace(b[i]) && b[i] != '=' && b[i] != '>' && b[i] != '/' {
			i++
		}
		an := lowerASCII(b[ns:i])
		for i < len(b) && isSpace(b[i]) {
			i++
		}
		var val string
		if i < len(b) && b[i] == '=' {
			i++
			for i < len(b) && isSpace(b[i]) {
				i++
			}
			if i < len(b) && (b[i] == '"' || b[i] == '\'') {
				q := b[i]
				i++
				vs := i
				for i < len(b) && b[i] != q {
					i++
				}
				val = string(b[vs:i])
				if i < len(b) {
					i++
				}
			} else {
				vs := i
				for i < len(b) && !isSpace(b[i]) && b[i] != '>' {
					i++
				}
				val = string(b[vs:i])
			}
		}
		if an == "" {
			continue
		}
		// htmlparser2 keeps the first occurrence of a repeated attribute.
		dup := false
		for k := range attrs {
			if attrs[k].Key == an {
				dup = true
				break
			}
		}
		if !dup {
			attrs = append(attrs, html.Attribute{Key: an, Val: val})
		}
	}
	return name, attrs
}

func scanEndTag(raw string) string {
	b := []byte(raw)
	if len(b) < 3 {
		return ""
	}
	i := 2 // skip '</'
	start := i
	for i < len(b) && !isSpace(b[i]) && b[i] != '>' {
		i++
	}
	return lowerASCII(b[start:i])
}

// render serializes the tree the way dom-serializer does with
// decodeEntities:false: text is written untouched and only the double quote is
// escaped inside attribute values.
func render(buf *bytes.Buffer, n *html.Node) {
	switch n.Type {
	case html.TextNode, html.DoctypeNode:
		buf.WriteString(n.Data)
		return
	case html.CommentNode:
		// htmlparser2 has no CDATA section outside XML mode, so it reports one
		// as a bogus comment and serializes it as a comment. Other bogus
		// constructs (<!decl>, <?pi?>) pass through untouched.
		if strings.HasPrefix(n.Data, "<![CDATA[") && strings.HasSuffix(n.Data, ">") {
			buf.WriteString("<!--")
			buf.WriteString(n.Data[2 : len(n.Data)-1])
			buf.WriteString("-->")
			return
		}
		buf.WriteString(n.Data)
		return
	}

	if n.Type == html.ElementNode {
		buf.WriteByte('<')
		buf.WriteString(n.Data)
		for _, a := range n.Attr {
			buf.WriteByte(' ')
			buf.WriteString(a.Key)
			if a.Val != "" {
				buf.WriteString(`="`)
				writeAttrValue(buf, a.Val)
				buf.WriteByte('"')
			}
		}
		// Childless elements in foreign content (svg, math) close themselves.
		if n.Namespace != "" && n.FirstChild == nil {
			buf.WriteString("/>")
			return
		}
		buf.WriteByte('>')
		if voidElements[n.Data] {
			return
		}
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		render(buf, c)
	}

	if n.Type == html.ElementNode {
		buf.WriteString("</")
		buf.WriteString(n.Data)
		buf.WriteByte('>')
	}
}

func writeAttrValue(buf *bytes.Buffer, v string) {
	for {
		i := strings.IndexByte(v, '"')
		if i < 0 {
			buf.WriteString(v)
			return
		}
		buf.WriteString(v[:i])
		buf.WriteString("&quot;")
		v = v[i+1:]
	}
}

func renderDocument(n *html.Node) []byte {
	var buf bytes.Buffer
	render(&buf, n)
	return buf.Bytes()
}

// --- small helpers over html.Node ---

func getAttr(n *html.Node, name string) (string, bool) {
	for _, a := range n.Attr {
		if a.Key == name {
			return a.Val, true
		}
	}
	return "", false
}

func setAttr(n *html.Node, name, v string) {
	for i := range n.Attr {
		if n.Attr[i].Key == name {
			n.Attr[i].Val = v
			return
		}
	}
	n.Attr = append(n.Attr, html.Attribute{Key: name, Val: v})
}

func removeAttr(n *html.Node, name string) {
	for i := range n.Attr {
		if n.Attr[i].Key == name {
			n.Attr = append(n.Attr[:i], n.Attr[i+1:]...)
			return
		}
	}
}

func detach(n *html.Node) {
	if n.Parent != nil {
		n.Parent.RemoveChild(n)
	}
}

// walk visits n and its descendants in document order. Callers that remove
// nodes must collect them first: RemoveChild clears NextSibling, which would
// silently truncate the walk.
func walk(n *html.Node, fn func(*html.Node)) {
	fn(n)
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		walk(c, fn)
	}
}

func findFirst(n *html.Node, tag string) *html.Node {
	if n.Type == html.ElementNode && n.Data == tag {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if f := findFirst(c, tag); f != nil {
			return f
		}
	}
	return nil
}
