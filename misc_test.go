package premailer_test

import (
	"bytes"
	"testing"

	"golang.org/x/net/html"

	premailer "github.com/mailproto/go-premailer"
)

func inlineCSSNode(t *testing.T, body string) *html.Node {
	pre, err := premailer.New([]byte(body))
	if err != nil {
		t.Error("couldn't parse for premailer", err)
	}

	inline, err := pre.ToInlineCSS()
	if err != nil {
		t.Error("couldn't convert to inline css", err)
	}

	doc, err := html.Parse(bytes.NewBuffer([]byte(inline)))
	if err != nil {
		t.Error("Resulting content was not html", err)
	}

	return doc
}

func checkElementStyle(t *testing.T, doc *html.Node, element, expectStyle string) {

	eachElement(doc, func(n *html.Node) bool {

		// Skip all non-matching elements
		if n.Data != element {
			return true
		}

		var hasStyle bool
		for _, attr := range n.Attr {
			if attr.Key == "style" {

				if expectStyle == "" {
					t.Errorf("Expected no style attribute, got: `%v`", attr.Val)
					continue
				}

				if attr.Val != expectStyle {
					t.Errorf("Wrong inlined style, want: `%v`, got: `%v`", expectStyle, attr.Val)
				}
				hasStyle = true
			}
		}

		if expectStyle != "" && !hasStyle {
			t.Error("style didn't get inlined properly")
		}
		return true
	})
}

func TestStylesInTheBody(t *testing.T) {

	doc := inlineCSSNode(t, `<html>
    <body>
    <style type="text/css"> p { color: red; } </style>
		<p>Test</p>
		</body>
    </html>`)

	checkElementStyle(t, doc, "p", "color: red")

}

// XXX: not sure why commented out styles should still be applied, but premailer does this
func TestCommentedOutStylesInTheBody(t *testing.T) {
	doc := inlineCSSNode(t, ` <html>
    <body>
    <style type="text/css"> <!-- p { color: red; } --> </style>
		<p>Test</p>
		</body>
		</html>`)

	checkElementStyle(t, doc, "p", "color: red")
}

func TestNotApplyingStylesToTheHead(t *testing.T) {
	doc := inlineCSSNode(t, `<html>
    <head>
    <title>Title</title>
    <style type="text/css"> * { color: red; } </style>
    </head>
    <body>
		<p><a>Test</a></p>
		</body>
		</html>`)

	checkElementStyle(t, doc, "head", "")
	checkElementStyle(t, doc, "title", "")
}

func TestMultipleIdenticalIDs(t *testing.T) {
	doc := inlineCSSNode(t, `<html>
    <head>
    <style type="text/css"> #the_id { color: red; } </style>
    </head>
    <body>
		<p id="the_id">Test</p>
		<p id="the_id">Test</p>
		</body>
		</html>`)

	checkElementStyle(t, doc, "p", "color: red")
}

func TestRemovingScripts(t *testing.T) {
	p, err := premailer.New([]byte(`<html>
    <head>
      <script>script to be removed</script>
    </head>
    <body>
      content
    </body>
    </html>`))
	if err != nil {
		t.Error("error creating premailer", err)
	}

	p.RemoveScripts = true

	inline, err := p.ToInlineCSS()
	if err != nil {
		t.Error("error inlining css", err)
	}

	doc, err := html.Parse(bytes.NewBuffer([]byte(inline)))
	if err != nil {
		t.Error("Resulting content was not html", err)
	}

	if e := findElement(doc, "script"); e != nil {
		t.Errorf("Shouldn't have found a script tag, got: %v", e)
	}
}

func TestDontRemoveScripts(t *testing.T) {
	p, err := premailer.New([]byte(`<html>
    <head>
      <script>script to be removed</script>
    </head>
    <body>
      content
    </body>
    </html>`))
	if err != nil {
		t.Error("error creating premailer", err)
	}

	p.RemoveScripts = false

	inline, err := p.ToInlineCSS()
	if err != nil {
		t.Error("error inlining css", err)
	}

	doc, err := html.Parse(bytes.NewBuffer([]byte(inline)))
	if err != nil {
		t.Error("Resulting content was not html", err)
	}

	if e := findElement(doc, "script"); e == nil {
		t.Errorf("Should have found a script tag, but didn't")
	}
}
