package premailer

import (
	"bytes"
	"strings"

	"github.com/aymerick/douceur/inliner"
	"golang.org/x/net/html"
)

func Douceur(raw string) (*html.Node, error) {
	inlined, err := inliner.Inline(raw)
	if err != nil {
		return nil, err
	}

	doc, err := html.Parse(bytes.NewBuffer([]byte(inlined)))
	if err != nil {
		return nil, err
	}

	// Apply some corrections to match github.com/premailer/premailer behaviour
	eachElement(doc, func(n *html.Node) bool {

		var attrs []html.Attribute
		for _, attr := range n.Attr {
			if attr.Key == "style" {

				// skip style elements on children of the `head` tag (and the head tag itself)
				if hasParent(n, "head") {
					continue
				}

				// remove trailing semicolons from style tags
				if attr.Val[len(attr.Val)-1] == ';' {
					attr.Val = strings.TrimRight(attr.Val, ";")
				}
			}
			attrs = append(attrs, attr)
		}
		n.Attr = attrs
		return true
	})

	return doc, err
}
