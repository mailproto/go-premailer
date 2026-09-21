package premailer

import (
	"crypto/md5"
	"encoding/hex"
)

// Document cleanup carried over from the Ruby premailer this package used to
// wrap. juice has no equivalent; these run after inlining so that removing a
// class cannot change which rules matched.
func (in *Inliner) cleanup(root *node) {
	o := &in.opts
	if !o.removeIDs && !o.removeClasses && !o.removeComments && !o.resetContentEditable {
		return
	}

	// Anchors referencing an id have to be rewritten to the same hash, so
	// collect targets before touching anything.
	targets := map[string]bool{}
	if o.removeIDs {
		walk(root, func(n *node) {
			if n.typ != nodeElement || n.tag != "a" {
				return
			}
			if href, ok := n.attr("href"); ok && len(href) > 1 && href[0] == '#' {
				targets[string(href[1:])] = true
			}
		})
	}

	var doomed []*node
	walk(root, func(n *node) {
		switch n.typ {
		case nodeComment:
			// Conditional comments carry Outlook-only markup, so they stay.
			if o.removeComments && !isConditionalComment(n.raw) {
				doomed = append(doomed, n)
			}
		case nodeElement:
			if o.removeClasses {
				n.removeAttr("class")
			}
			if o.resetContentEditable {
				n.removeAttr("contenteditable")
			}
			if o.removeIDs {
				if id, ok := n.attr("id"); ok {
					if targets[string(id)] {
						n.setAttr("id", []byte(hashID(id)))
					} else {
						n.removeAttr("id")
					}
				}
				if href, ok := n.attr("href"); ok && len(href) > 1 && href[0] == '#' {
					n.setAttr("href", []byte("#"+hashID(href[1:])))
				}
			}
		}
	})
	// Detach after the walk: removing during it would skip siblings.
	for _, n := range doomed {
		n.remove()
	}
}

func hashID(id []byte) string {
	sum := md5.Sum(id)
	return hex.EncodeToString(sum[:])
}

func isConditionalComment(raw []byte) bool {
	// <!--[if ...]> ... <![endif]-->
	const open = "<!--["
	return len(raw) > len(open) && string(raw[:len(open)]) == open
}

func walk(n *node, fn func(*node)) {
	fn(n)
	for _, k := range n.kids {
		walk(k, fn)
	}
}
