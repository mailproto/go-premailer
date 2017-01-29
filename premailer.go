package premailer

import (
	"bytes"

	"golang.org/x/net/html"
)

type Premailer struct {
	LineLength           int
	RemoveScripts        bool
	ResetContentEditable bool
	RemoveIDs            bool
	RemoveClasses        bool
	RemoveComments       bool

	processed *html.Node
	orig      []byte
}

// New parses an HTML fragment []byte and returns the result
func New(body []byte) (*Premailer, error) {
	return &Premailer{
		LineLength: DefaultLineLength,
		orig:       body,
	}, nil
}

// ToPlaintext converts the input document to the plaintext version
func (p Premailer) ToPlaintext() (string, error) {
	if p.processed == nil {
		_, err := p.ToInlineCSS()
		if err != nil {
			return "", err
		}
	}
	baseElement := findElement(p.processed, "body")
	if baseElement == nil {
		baseElement = p.processed
	}

	var buf bytes.Buffer
	err := html.Render(&buf, baseElement)
	if err != nil {
		return "", err
	}

	return ConvertToText(string(buf.Bytes()), p.LineLength), nil
}
