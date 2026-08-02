// Copyright 2026 Joshua Rich <joshua.rich@gmail.com>.
// SPDX-License-Identifier: 	AGPL-3.0-or-later

package htmlx

import (
	"bytes"
	"regexp"
	"slices"
	"strings"

	"golang.org/x/net/html"
)

// preserveTags is the set of elements allowed to keep their attributes.
var defaultPreserveTags = map[string]bool{
	"img":     true,
	"audio":   true,
	"video":   true,
	"figure":  true,
	"svg":     true,
	"source":  true,
	"track":   true,
	"a":       true,
	"area":    true,
	"iframe":  true,
	"canvas":  true,
	"picture": true,
	"time":    true,
	"data":    true,
}

func StripAttributesOption(node *html.Node) {
	stripWalk(node, defaultPreserveTags)
}

func stripWalk(node *html.Node, preserveTags map[string]bool) {
	if preserveTags == nil {
		preserveTags = defaultPreserveTags
	}
	if node.Type == html.ElementNode && !preserveTags[node.Data] {
		// Preserve any id attribute, strip all others.
		if idx := slices.IndexFunc(node.Attr, func(a html.Attribute) bool {
			return a.Key == "id"
		}); idx != -1 {
			node.Attr = []html.Attribute{node.Attr[idx]}
		} else {
			node.Attr = nil
		}
	}
	for c := node.FirstChild; c != nil; c = c.NextSibling {
		stripWalk(c, preserveTags)
	}
}

var preheaderPadding = regexp.MustCompile(`[\x{00A0}\x{200C}\s]{10,}`)

func StripEmailPreheaderPadding(html string) string {
	return preheaderPadding.ReplaceAllString(html, "")
}

func StripWhitespaceOnlyNodesOption(node *html.Node) {
	// Snapshot children first since removal mutates sibling pointers
	var children []*html.Node
	for c := node.FirstChild; c != nil; c = c.NextSibling {
		children = append(children, c)
	}

	for child := range slices.Values(children) {
		if child.Type == html.ElementNode {
			StripWhitespaceOnlyNodesOption(child) // recurse first (post-order)
			if isBlank(textContent(child)) {
				removeNode(child)
			}
		}
	}
}

// textContent recursively collects all text within a node (like DOM's textContent).
func textContent(node *html.Node) string {
	var buf bytes.Buffer
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			buf.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(node)
	return buf.String()
}

// isBlank treats regular whitespace and \u00A0 (&nbsp;) as empty.
func isBlank(s string) bool {
	return strings.TrimFunc(s, func(r rune) bool {
		return r == ' ' || r == '\t' || r == '\n' || r == '\r' || r == '\u00A0'
	}) == ""
}

// removeNode detaches n from its parent.
func removeNode(n *html.Node) {
	if n.Parent != nil {
		n.Parent.RemoveChild(n)
	}
}
