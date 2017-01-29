package premailer_test

import (
	"bytes"
	"testing"

	"golang.org/x/net/html"

	premailer "github.com/mailproto/go-premailer"
)

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
