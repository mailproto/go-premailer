package premailer

import (
	"bytes"

	"github.com/tdewolff/parse/v2"
	"github.com/tdewolff/parse/v2/css"
)

// decl is one declaration from a stylesheet.
type decl struct {
	prop      string
	value     []byte // sliced from the source, so original spacing survives
	important bool
	ord       uint32 // position in the stylesheet; breaks specificity ties
}

// preserved is the source span of an at-rule that cannot be inlined and is
// re-emitted into a surviving <style> element.
type preserved struct {
	kind string // "media", "font-face", "keyframes", ...
	text []byte
}

// parseStylesheet splits src into inlinable rules and at-rules to preserve.
//
// At-rules never become rules: that is how juice excludes @media, @font-face
// and @keyframes from inlining, rather than by filtering them later.
//
// Values are sliced out of src rather than rebuilt from tokens, because
// tdewolff normalizes whitespace inside a value ("rgba(0 , 0,0,.5)" becomes
// "rgba(0,0,0,.5)") and juice emits it exactly as written.
func parseStylesheet(src []byte, opts *options, ord uint32) ([]rule, []preserved, uint32) {
	var rules []rule
	var keep []preserved

	p := css.NewParser(parse.NewInputBytes(src), false)
	prev := 0
	atDepth := 0
	atStart := 0
	atKind := ""

	var sel []byte
	var decls []decl

	for {
		gt, _, data := p.Next()
		if gt == css.ErrorGrammar {
			break
		}
		start, end := prev, p.Offset()
		prev = end

		switch gt {
		case css.BeginAtRuleGrammar:
			if atDepth == 0 {
				atStart = start
				atKind = atRuleKind(data)
			}
			atDepth++

		case css.AtRuleGrammar:
			// A bodyless at-rule such as `@import url(x);` or `@layer a;`.
			// The terminating semicolon is part of how it is re-emitted.
			if atDepth == 0 && preserveKind(atRuleKind(data), opts) {
				text := append(append([]byte{}, trimCSS(src[start:end])...), ';')
				keep = append(keep, preserved{atRuleKind(data), text})
			}

		case css.EndAtRuleGrammar:
			atDepth--
			if atDepth == 0 && preserveKind(atKind, opts) {
				keep = append(keep, preserved{atKind, trimCSS(src[atStart:end])})
			}

		case css.BeginRulesetGrammar:
			sel = selectorText(src, start, end)
			decls = decls[:0]

		case css.DeclarationGrammar:
			if atDepth > 0 {
				continue
			}
			v, imp := declValue(src[start:end])
			if len(v) == 0 {
				// juice drops empty values, preserving mensch behaviour.
				continue
			}
			if imp && !opts.preserveImportant {
				// Scored as important either way; only the text is dropped.
			} else if imp {
				v = append(append([]byte{}, v...), " !important"...)
			}
			decls = append(decls, decl{prop: string(data), value: v, important: imp, ord: ord})
			ord++

		case css.CustomPropertyGrammar:
			if atDepth > 0 {
				continue
			}
			v, imp := declValue(src[start:end])
			decls = append(decls, decl{prop: string(data), value: v, important: imp, ord: ord})
			ord++

		case css.EndRulesetGrammar:
			if atDepth > 0 || len(decls) == 0 {
				continue
			}
			shared := append([]decl(nil), decls...)
			anyIgnored := false
			for _, arm := range splitSelector(sel) {
				r, ok := compileRule(arm, shared)
				if !ok {
					// juice swallows selectors its engine cannot parse.
					continue
				}
				if r.ignored {
					anyIgnored = true
					continue
				}
				rules = append(rules, r)
			}
			// juice preserves the rule as written, selector list and all, so
			// a:hover keeps its :hover arm in <style> while the plain arm is
			// still inlined.
			if anyIgnored && opts.preservePseudos {
				keep = append(keep, preserved{"pseudo", ruleText(sel, shared)})
			}
		}
	}
	return rules, keep, ord
}

// ruleText re-emits a rule that could not be inlined, formatted the way
// juice's stringifier does: two-space indent, one declaration per line.
func ruleText(sel []byte, decls []decl) []byte {
	var b bytes.Buffer
	b.Write(sel)
	b.WriteString(" {")
	for _, d := range decls {
		b.WriteString("\n  ")
		b.WriteString(d.prop)
		b.WriteString(": ")
		b.Write(d.value)
		b.WriteByte(';')
	}
	b.WriteString("\n}")
	return b.Bytes()
}

func atRuleKind(data []byte) string {
	k := string(bytes.TrimLeft(bytes.ToLower(data), "@"))
	if i := bytes.IndexByte([]byte(k), ' '); i >= 0 {
		k = k[:i]
	}
	// Vendor-prefixed keyframes still count as keyframes.
	for _, p := range []string{"-webkit-", "-moz-", "-ms-", "-o-"} {
		if len(k) > len(p) && k[:len(p)] == p {
			k = k[len(p):]
		}
	}
	return k
}

func preserveKind(kind string, o *options) bool {
	switch kind {
	case "media":
		return o.preserveMediaQueries
	case "font-face":
		return o.preserveFontFaces
	case "keyframes":
		return o.preserveKeyFrames
	}
	return false
}

// selectorText recovers the selector as written. The span runs to just past
// the opening brace.
func selectorText(src []byte, start, end int) []byte {
	s := src[start:end]
	if i := bytes.LastIndexByte(s, '{'); i >= 0 {
		s = s[:i]
	}
	return trimCSS(s)
}

// declValue pulls the value out of a declaration span, which may carry a
// leading separator from the previous declaration.
func declValue(span []byte) ([]byte, bool) {
	i := bytes.IndexByte(span, ':')
	if i < 0 {
		return nil, false
	}
	v := span[i+1:]
	// The last declaration of a ruleset runs up to and past the closing
	// brace. Trim it before looking for !important, which would otherwise
	// never be recognised there.
	if j := bytes.LastIndexByte(v, '}'); j >= 0 {
		v = v[:j]
	}
	v = trimCSS(v)
	if j := importantSuffix(v); j >= 0 {
		return trimCSS(v[:j]), true
	}
	return v, false
}

// importantSuffix reports where a trailing !important begins, or -1.
func importantSuffix(v []byte) int {
	const kw = "!important"
	end := len(v)
	for end > 0 && isCSSSpace(v[end-1]) {
		end--
	}
	if end < len(kw) {
		return -1
	}
	if !bytes.EqualFold(v[end-len(kw):end], []byte(kw)) {
		return -1
	}
	return end - len(kw)
}

func isCSSSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\f'
}

func trimCSS(b []byte) []byte {
	for len(b) > 0 && (isCSSSpace(b[0]) || b[0] == ';') {
		b = b[1:]
	}
	for len(b) > 0 && (isCSSSpace(b[len(b)-1]) || b[len(b)-1] == ';') {
		b = b[:len(b)-1]
	}
	return b
}
