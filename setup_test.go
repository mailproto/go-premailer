package premailer_test

import (
	"strings"
	"testing"

	"golang.org/x/net/html"

	premailer "github.com/mailproto/go-premailer"
)

func checkConvertToText(t *testing.T, expect, html string) {
	if plain := premailer.ConvertToText(html, premailer.DefaultLineLength); strings.TrimSpace(plain) != expect {
		t.Errorf("Wrong conversion of `%v`, want: %v got: %v", html, expect, plain)
	}
}

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
func eachElement(root *html.Node, callback func(parent, n *html.Node) bool) {
	if root == nil {
		return
	}
	for c := root; c != nil; c = c.NextSibling {
		if !callback(root, c) {
			return
		}
		if c.FirstChild != nil {
			eachElement(c.FirstChild, callback)
		}
	}

}

func findElement(root *html.Node, element string) *html.Node {
	var found *html.Node
	eachElement(root, func(parent, n *html.Node) bool {
		if n.Type == html.ElementNode && n.Data == element {
			found = n
			return false
		}
		return true
	})

	return found
}
