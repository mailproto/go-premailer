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

	doc       *html.Node
	processed *html.Node
	orig      []byte
}

// New parses an HTML fragment []byte and returns the result
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

// ToPlainText converts the input document to the plaintext version
func (p Premailer) ToPlaintext() (string, error) {
	baseElement := findElement(p.doc, "body")
	if baseElement == nil {
		baseElement = p.doc
	}

	var buf bytes.Buffer
	err := html.Render(&buf, baseElement)
	if err != nil {
		return "", err
	}

	return ConvertToText(string(buf.Bytes()), p.LineLength), nil
}
