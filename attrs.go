package juicer

import (
	"crypto/md5"
	"encoding/hex"
	"strings"

	"golang.org/x/net/html"
)

// Document cleanup carried over from the Ruby premailer this package used to
// wrap. juice has no equivalent; these run after inlining so that removing a
// class cannot change which rules matched.
func (in *pass) cleanup(root *html.Node) {
	o := in.o
	if !o.removeIDs && !o.removeClasses && !o.removeComments && !o.resetContentEditable {
		return
	}

	// Anchors referencing an id have to be rewritten to the same hash, so
	// collect targets before touching anything.
	targets := map[string]bool{}
	if o.removeIDs {
		walk(root, func(n *html.Node) {
			if n.Type != html.ElementNode || n.Data != "a" {
				return
			}
			if href, ok := getAttr(n, "href"); ok && strings.HasPrefix(href, "#") && len(href) > 1 {
				targets[href[1:]] = true
			}
		})
	}

	var doomed []*html.Node
	walk(root, func(n *html.Node) {
		switch n.Type {
		case html.CommentNode:
			// Conditional comments carry Outlook-only markup, so they stay.
			if o.removeComments && !isConditionalComment(n.Data) {
				doomed = append(doomed, n)
			}
		case html.ElementNode:
			if o.removeClasses {
				removeAttr(n, "class")
			}
			if o.resetContentEditable {
				removeAttr(n, "contenteditable")
			}
			if o.removeIDs {
				if id, ok := getAttr(n, "id"); ok {
					if targets[id] {
						setAttr(n, "id", hashID(id))
					} else {
						removeAttr(n, "id")
					}
				}
				if href, ok := getAttr(n, "href"); ok && strings.HasPrefix(href, "#") && len(href) > 1 {
					setAttr(n, "href", "#"+hashID(href[1:]))
				}
			}
		}
	})
	// Detach after the walk: RemoveChild clears NextSibling, so removing
	// during the walk would silently skip the rest of each sibling list.
	for _, n := range doomed {
		detach(n)
	}
}

func hashID(id string) string {
	sum := md5.Sum([]byte(id))
	return hex.EncodeToString(sum[:])
}

func isConditionalComment(raw string) bool {
	// <!--[if ...]> ... <![endif]-->
	return strings.HasPrefix(raw, "<!--[")
}

// Outlook ignores CSS width, height and background on table elements, so
// juice mirrors those declarations into presentational attributes.
//
// Two details are load-bearing. These read the resolved property map rather
// than reparsing the style attribute, so they see values after variable
// substitution. And they only look at the head of a duplicate chain.

var widthHeightElements = map[string]bool{
	"table": true, "td": true, "th": true, "img": true,
}

var tableElements = map[string]bool{
	"table": true, "th": true, "tr": true, "td": true, "caption": true,
	"colgroup": true, "col": true, "thead": true, "tbody": true, "tfoot": true,
}

var styleToAttribute = []struct{ prop, attr string }{
	{"background-color", "bgcolor"},
	{"background-image", "background"},
	{"text-align", "align"},
	{"vertical-align", "valign"},
}

func (in *pass) promoteAttributes(root *html.Node) {
	o := in.o
	for _, el := range in.resolved {
		n, m := el.node, el.props
		if o.applyWidthAttributes {
			setDimension(n, m, "width")
		}
		if o.applyHeightAttributes {
			setDimension(n, m, "height")
		}
		if o.applyAttributesTableElement && tableElements[n.Data] {
			for _, sa := range styleToAttribute {
				i := m.find(sa.prop)
				if i < 0 {
					continue
				}
				v := string(m.slots[i].value)
				// A gradient cannot go in a presentational attribute.
				if strings.Contains(v, "linear-gradient(") || strings.Contains(v, "radial-gradient(") {
					continue
				}
				if sa.prop == "background-image" {
					u, ok := extractURL(v)
					if !ok {
						continue
					}
					v = u
				}
				setAttr(n, sa.attr, v)
			}
		}
	}
}

func setDimension(n *html.Node, m *propMap, dim string) {
	if !widthHeightElements[n.Data] {
		return
	}
	i := m.find(dim)
	if i < 0 {
		return
	}
	v := string(m.slots[i].value)
	// juice tests for px or auto anywhere in the value, then strips the first
	// "px". That really does turn `height: auto` into height="auto".
	if strings.Contains(v, "px") || strings.Contains(v, "auto") {
		setAttr(n, dim, strings.Replace(v, "px", "", 1))
		return
	}
	if tableElements[n.Data] && strings.Contains(v, "%") {
		setAttr(n, dim, v)
	}
}

// extractURL unwraps url("x") to x.
func extractURL(v string) (string, bool) {
	if !strings.HasPrefix(v, "url(") || !strings.HasSuffix(v, ")") {
		return "", false
	}
	inner := strings.TrimSpace(v[4 : len(v)-1])
	if len(inner) >= 2 && (inner[0] == '"' || inner[0] == '\'') && inner[len(inner)-1] == inner[0] {
		inner = inner[1 : len(inner)-1]
	}
	if inner == "" {
		return "", false
	}
	return inner, true
}
