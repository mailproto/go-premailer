// Package premailer inlines CSS into HTML email, matching the behaviour of
// the JavaScript library juice (https://github.com/Automattic/juice) v12.
//
// Output is intended to be byte-identical to juice's, which is why the HTML
// parser and serializer in parse.go deliberately do not follow the HTML5 tree
// construction spec: juice parses with htmlparser2, and matching a mail
// client's view of the message means matching that.
package premailer

// Inliner holds resolved options and is safe for concurrent use. Reuse one
// across documents rather than constructing per call.
type Inliner struct {
	opts options
}

type options struct {
	extraCSS                    string
	applyStyleTags              bool
	removeStyleTags             bool
	preserveMediaQueries        bool
	preserveFontFaces           bool
	preserveKeyFrames           bool
	preservePseudos             bool
	preserveImportant           bool
	applyWidthAttributes        bool
	applyHeightAttributes       bool
	applyAttributesTableElement bool
	resolveCSSVariables         bool
	inlinePseudoElements        bool
	styleAttributeName          string
	codeBlocks                  []codeBlock

	// Document cleanup, carried over from the Ruby premailer this package
	// used to wrap. juice has no equivalent.
	removeIDs            bool
	removeClasses        bool
	removeComments       bool
	resetContentEditable bool
}

// defaults mirror juice's getDefaultOptions. Note that preserveImportant and
// inlinePseudoElements are absent from that table and so default to false:
// !important is stripped from the emitted value even though it still wins the
// cascade.
func defaults() options {
	return options{
		applyStyleTags:              true,
		removeStyleTags:             true,
		preserveMediaQueries:        true,
		preserveFontFaces:           true,
		preserveKeyFrames:           true,
		preservePseudos:             true,
		applyWidthAttributes:        true,
		applyHeightAttributes:       true,
		applyAttributesTableElement: true,
		resolveCSSVariables:         true,
		styleAttributeName:          "style",
		codeBlocks:                  defaultCodeBlocks(),
	}
}

// Option configures an Inliner.
type Option func(*options)

// ExtraCSS is appended to the CSS collected from the document.
func ExtraCSS(css string) Option { return func(o *options) { o.extraCSS = css } }

// ApplyStyleTags controls whether CSS is collected from <style> elements.
func ApplyStyleTags(v bool) Option { return func(o *options) { o.applyStyleTags = v } }

// RemoveStyleTags controls whether <style> elements are removed after
// inlining. Rules that cannot be inlined are kept regardless.
func RemoveStyleTags(v bool) Option { return func(o *options) { o.removeStyleTags = v } }

// PreserveMediaQueries keeps @media blocks in a surviving <style> element.
func PreserveMediaQueries(v bool) Option { return func(o *options) { o.preserveMediaQueries = v } }

// PreserveFontFaces keeps @font-face blocks in a surviving <style> element.
func PreserveFontFaces(v bool) Option { return func(o *options) { o.preserveFontFaces = v } }

// PreserveKeyFrames keeps @keyframes blocks in a surviving <style> element.
func PreserveKeyFrames(v bool) Option { return func(o *options) { o.preserveKeyFrames = v } }

// PreservePseudos keeps rules using :hover and friends in a surviving <style>.
func PreservePseudos(v bool) Option { return func(o *options) { o.preservePseudos = v } }

// PreserveImportant keeps the literal !important suffix on inlined values.
func PreserveImportant(v bool) Option { return func(o *options) { o.preserveImportant = v } }

// ApplyWidthAttributes emits width="" from a CSS width on table cells and images.
func ApplyWidthAttributes(v bool) Option { return func(o *options) { o.applyWidthAttributes = v } }

// ApplyHeightAttributes emits height="" from a CSS height on table cells and images.
func ApplyHeightAttributes(v bool) Option { return func(o *options) { o.applyHeightAttributes = v } }

// ApplyAttributesTableElements emits bgcolor, background, align and valign on
// table elements, which Outlook honours where it ignores the CSS.
func ApplyAttributesTableElements(v bool) Option {
	return func(o *options) { o.applyAttributesTableElement = v }
}

// ResolveCSSVariables substitutes var() references and drops custom properties.
func ResolveCSSVariables(v bool) Option { return func(o *options) { o.resolveCSSVariables = v } }

// InlinePseudoElements materializes ::before and ::after as real elements.
func InlinePseudoElements(v bool) Option { return func(o *options) { o.inlinePseudoElements = v } }

// StyleAttributeName writes inlined declarations to an attribute other than style.
func StyleAttributeName(name string) Option {
	return func(o *options) { o.styleAttributeName = name }
}

// CodeBlocks replaces the template delimiters protected from the HTML parser.
// The default set covers Liquid, Handlebars and EJS.
func CodeBlocks(b []codeBlock) Option { return func(o *options) { o.codeBlocks = b } }

// RemoveIDs replaces id attributes with a hash, rewriting internal anchors.
func RemoveIDs(v bool) Option { return func(o *options) { o.removeIDs = v } }

// RemoveClasses strips class attributes after inlining.
func RemoveClasses(v bool) Option { return func(o *options) { o.removeClasses = v } }

// RemoveComments strips HTML comments. Conditional comments are kept.
func RemoveComments(v bool) Option { return func(o *options) { o.removeComments = v } }

// ResetContentEditable strips contenteditable attributes.
func ResetContentEditable(v bool) Option { return func(o *options) { o.resetContentEditable = v } }

// New returns an Inliner configured by opts.
func New(opts ...Option) *Inliner {
	o := defaults()
	for _, fn := range opts {
		fn(&o)
	}
	return &Inliner{opts: o}
}

// Inline inlines the document's CSS into style attributes.
func (in *Inliner) Inline(doc string) (string, error) {
	out, err := in.InlineBytes([]byte(doc))
	return string(out), err
}

// InlineBytes is Inline over a byte slice.
func (in *Inliner) InlineBytes(doc []byte) ([]byte, error) {
	src, blocks := encodeCodeBlocks(doc, in.opts.codeBlocks)
	root := parseDocument(src)
	if err := in.process(root); err != nil {
		return nil, err
	}
	return decodeCodeBlocks(renderDocument(root), blocks), nil
}

// Inline inlines doc's CSS using a default Inliner.
func Inline(doc string, opts ...Option) (string, error) {
	return New(opts...).Inline(doc)
}

// process runs the inlining pipeline over an already-parsed tree.
func (in *Inliner) process(root *node) error {
	in.cleanup(root)
	return nil
}
