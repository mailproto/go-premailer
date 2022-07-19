package premailer_test

import (
	"testing"

	"golang.org/x/net/html"

	premailer "github.com/mailproto/go-premailer"
)

func checkPremailerToPlaintext(t *testing.T, expect, html string) {
	p, err := premailer.New([]byte(html))
	if err != nil {
		t.Errorf("Error creating premailer from `%v`: %v", html, err)
	}

	if plain, err := p.ToPlaintext(); err != nil {
		t.Errorf("Error generating plaintext from `%v`: %v", html, err)
	} else if plain != expect {
		t.Errorf("Wrong content from `%v`, want: `%v` got: `%v`", html, expect, plain)
	}
}

// XXX: these are copied from the main pkg
func eachElement(root *html.Node, callback func(n *html.Node) bool) {
	var iter func(*html.Node)
	iter = func(n *html.Node) {
		if n == nil {
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if !callback(c) {
				return
			}
			iter(c)
		}
	}
	iter(root)
	callback(root)
}

func findElement(root *html.Node, element string) *html.Node {
	var found *html.Node
	eachElement(root, func(n *html.Node) bool {
		if n.Type == html.ElementNode && n.Data == element {
			found = n
			return false
		}
		return true
	})

	return found
}
