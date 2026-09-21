package premailer

import (
	"bytes"

	"golang.org/x/net/html"
)

// CSS custom property resolution.
//
// Elements are visited in document order, so an ancestor's variables are
// always substituted before a descendant reads them. juice instead walks its
// styleProps in accumulation order (rule order crossed with document order)
// and can therefore read a value its own later pass would have changed; that
// order dependence is a bug we do not reproduce.
//
// One juice behaviour is worth keeping and falls out for free: only elements
// some rule matched have a property map, so a custom property declared on an
// untouched element is invisible.

func (in *pass) resolveVariables() {
	for _, el := range in.resolved {
		for i := range el.props.slots {
			p := &el.props.slots[i]
			if bytes.HasPrefix([]byte(p.prop), []byte("--")) {
				continue
			}
			if !bytes.Contains(p.value, []byte("var(")) {
				continue
			}
			p.value = in.substitute(el.node, p.value, 0)
		}
	}
}

const maxVarDepth = 16

// substitute replaces every var() reference in v. An unresolvable reference
// with no fallback is left in place, as juice leaves it.
func (in *pass) substitute(n *html.Node, v []byte, depth int) []byte {
	if depth > maxVarDepth {
		return v
	}
	out := v
	// from is where the next scan starts, so a reference we leave in place
	// cannot be found again on the following pass.
	from := 0
	for from < len(out) {
		rel := bytes.Index(out[from:], []byte("var("))
		if rel < 0 {
			return out
		}
		at := from + rel
		end := parenEnd(out, at+3)
		// parenEnd falls back to the end of input when the parenthesis is
		// never closed, which is not a usable span.
		if end < at+5 || out[end-1] != ')' {
			return out
		}
		name, fallback := splitVarArgs(out[at+4 : end-1])

		var repl []byte
		switch val, ok := in.lookupVar(n, string(bytes.TrimSpace(name))); {
		case ok:
			repl = in.substitute(n, val, depth+1)
		case fallback != nil:
			repl = in.substitute(n, bytes.TrimSpace(fallback), depth+1)
		default:
			from = end
			continue
		}

		merged := make([]byte, 0, len(out)-(end-at)+len(repl))
		merged = append(merged, out[:at]...)
		merged = append(merged, repl...)
		merged = append(merged, out[end:]...)
		out = merged
		from = at + len(repl)
	}
	return out
}

// lookupVar finds a custom property on the element or its nearest ancestor
// that a rule matched.
func (in *pass) lookupVar(n *html.Node, name string) ([]byte, bool) {
	for e := n; e != nil; e = e.Parent {
		m, ok := in.byNode[e]
		if !ok {
			continue
		}
		if i := m.find(name); i >= 0 {
			return m.slots[i].value, true
		}
	}
	return nil, false
}

// splitVarArgs divides a var() argument list into the name and its optional
// fallback, at the first top-level comma.
func splitVarArgs(arg []byte) (name, fallback []byte) {
	depth := 0
	for i := 0; i < len(arg); i++ {
		switch arg[i] {
		case '(':
			depth++
		case ')':
			depth--
		case ',':
			if depth == 0 {
				return arg[:i], arg[i+1:]
			}
		}
	}
	return arg, nil
}
