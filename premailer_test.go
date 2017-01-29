package premailer_test

import (
	"bytes"
	"testing"

	"golang.org/x/net/html"

	premailer "github.com/mailproto/go-premailer"
)

func TestRemoveIDs(t *testing.T) {

	body := `<html> <head> <style type="text/css"> #remove { color:blue; } </style> </head>
    <body>
		<p id="remove"><a href="#keep">Test</a></p>
		<div id="keep">Test</div>
		</body> </html>`

	p, err := premailer.New([]byte(body))
	if err != nil {
		t.Error("couldn't parse for premailer", err)
	}

	p.RemoveIDs = true

	inline, err := p.ToInlineCSS()
	if err != nil {
		t.Error("error inlining css", err)
	}

	doc, err := html.Parse(bytes.NewBuffer([]byte(inline)))
	if err != nil {
		t.Error("Resulting content was not html", err)
	}

	remove := findElement(doc, "p")
	for _, attr := range remove.Attr {
		if attr.Key == "id" {
			t.Error("id attribute not removed", remove.Attr)
		}
	}

	link := findElement(doc, "a")
	var linkHref string
	for _, attr := range link.Attr {
		if attr.Key == "href" {
			linkHref = attr.Val
		}
	}

	var hadID bool
	keep := findElement(doc, "div")
	for _, attr := range keep.Attr {
		if attr.Key == "id" {
			if attr.Val != linkHref[1:] {
				t.Errorf("Expected ID to match href want: %v, got: %v", linkHref[1:], attr.Val)
			}
			hadID = true
		}
	}
	if !hadID {
		t.Error("Expected id value, got none on", keep)
	}
}

func TestResetContentEditable(t *testing.T) {
	body := `<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd">
    <html> <head> <style type="text/css"> #remove { color:blue; } </style> </head>
    <body>
    <div contenteditable="true" id="editable"> Test </div>
    </body> </html>`

	p, err := premailer.New([]byte(body))
	if err != nil {
		t.Error("couldn't parse for premailer", err)
	}

	p.ResetContentEditable = true

	inline, err := p.ToInlineCSS()
	if err != nil {
		t.Error("error inlining css", err)
	}

	doc, err := html.Parse(bytes.NewBuffer([]byte(inline)))
	if err != nil {
		t.Error("Resulting content was not html", err)
	}

	node := findElement(doc, "div")
	for _, attr := range node.Attr {
		if attr.Key == "contenteditable" {
			t.Error("contenteditable attribute not removed", node.Attr)
		}
	}

}

func TestRemoveComments(t *testing.T) {
	body := `<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd">
    <html> <head> <style type="text/css"> #remove { color:blue; } </style> </head>
    <body>
    <!-- Link to example.com -->
    <a href="http://example.com">Example</a>
    </body> </html>`

	p, err := premailer.New([]byte(body))
	if err != nil {
		t.Error("couldn't parse for premailer", err)
	}

	p.RemoveComments = true

	inline, err := p.ToInlineCSS()
	if err != nil {
		t.Error("error inlining css", err)
	}

	doc, err := html.Parse(bytes.NewBuffer([]byte(inline)))
	if err != nil {
		t.Error("Resulting content was not html", err)
	}

	eachElement(doc, func(_, n *html.Node) bool {
		if n.Type == html.CommentNode {
			t.Error("Should have removed all comment nodes, found", n)
		}
		return true
	})
}

func TestDontRemoveComments(t *testing.T) {
	body := `<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd">
    <html> <head> <style type="text/css"> #remove { color:blue; } </style> </head>
    <body>
    <!-- Link to example.com -->
    <a href="http://example.com">Example</a>
    </body> </html>`

	p, err := premailer.New([]byte(body))
	if err != nil {
		t.Error("couldn't parse for premailer", err)
	}

	p.RemoveComments = false

	inline, err := p.ToInlineCSS()
	if err != nil {
		t.Error("error inlining css", err)
	}

	doc, err := html.Parse(bytes.NewBuffer([]byte(inline)))
	if err != nil {
		t.Error("Resulting content was not html", err)
	}

	var gotComment bool
	eachElement(doc, func(_, n *html.Node) bool {
		if n.Type == html.CommentNode {
			gotComment = true
		}
		return true
	})
	if !gotComment {
		t.Error("Expected comment, found none")
	}
}
