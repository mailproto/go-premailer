package premailer

import (
	"bytes"

	"golang.org/x/net/html"
)

type Premailer struct {
	LineLength int

	doc  *html.Node
	orig []byte
}

func New(body []byte) (*Premailer, error) {

	doc, err := html.Parse(bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	return &Premailer{
		LineLength: DefaultLineLength,
		doc:        doc,
		orig:       body,
	}, nil
}

func findElement(n *html.Node, element string) *html.Node {
	for c := n; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == element {
			return c
		} else if c.FirstChild != nil {
			if n := findElement(c.FirstChild, element); n != nil {
				return n
			}
		}
	}
	return nil
}

// ToPlainText converts the input document to the plaintext version
func (p Premailer) ToPlainText() (string, error) {
	baseElement := findElement(p.doc, "body")
	if baseElement == nil {
		baseElement = p.doc
	}

	var buf bytes.Buffer
	err := html.Render(&buf, baseElement)
	if err != nil {
		return "", err
	}

	return ConvertToText(string(buf.Bytes()), p.LineLength, DefaultCharset), nil
}
