package premailer

import "golang.org/x/net/html"

func eachElement(root *html.Node, callback func(n *html.Node) bool) {
	var iter func(*html.Node)
	iter = func(n *html.Node) {
		if n == nil {
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if !callback(c) {
				return
			}
			iter(c)
		}
	}
	iter(root)
	callback(root)
}

func findElement(root *html.Node, element string) *html.Node {
	var found *html.Node
	eachElement(root, func(n *html.Node) bool {
		if n.Type == html.ElementNode && n.Data == element {
			found = n
			return false
		}
		return true
	})

	return found
}

func removeAllElement(parent *html.Node, element string) {
	for c := parent.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == element {
			parent.RemoveChild(c)
		}
		removeAllElement(c, element)
	}
}

func removeAttribute(node *html.Node, attribute string) {
	var attrs []html.Attribute
	for _, attr := range node.Attr {
		switch attr.Key {
		case attribute:
			continue
		default:
			attrs = append(attrs, attr)
		}
	}

	node.Attr = attrs
}

// XXX: more like hasParentOrIsElement...
func hasParent(node *html.Node, parent string) bool {
	if node == nil {
		return false
	} else if node.Data == parent {
		return true
	} else if node.Parent != nil {
		if node.Parent.Data == parent {
			return true
		}
		return hasParent(node.Parent, parent)
	}
	return false
}
