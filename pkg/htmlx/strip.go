// Copyright 2026 Joshua Rich <joshua.rich@gmail.com>.
// SPDX-License-Identifier: 	AGPL-3.0-or-later

package htmlx

import (
	"bytes"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
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

// StripAttributesFragment parses the given HTML fragment and removes all attributes from every element. An optional
// list of tags to preserve can be supplied, otherwise a default list will be used.
func StripAttributesFragment(input string, preserveTags map[string]bool) (string, error) {
	trimmed := strings.TrimSpace(input)
	lower := strings.ToLower(trimmed)

	if isFullDocument := strings.HasPrefix(lower, "<!doctype") ||
		strings.Contains(lower, "<html"); isFullDocument {
		return stripFullDocument(input, preserveTags)
	}
	return stripFragment(input, preserveTags)
}

func stripFullDocument(input string, preserveTags map[string]bool) (string, error) {
	doc, err := html.Parse(strings.NewReader(input))
	if err != nil {
		return "", fmt.Errorf("parse html: %w", err)
	}
	stripWalk(doc, preserveTags)

	// Create a buffer for sanitized html.
	buf, ok := bufPool.Get().(*bytes.Buffer)
	if !ok {
		return "", errors.New("unable to retrieve buffer")
	}
	buf.Reset()
	defer bufPool.Put(buf)

	if err := html.Render(buf, doc); err != nil {
		return "", fmt.Errorf("render html: %w", err)
	}
	return buf.String(), nil
}

func stripFragment(input string, preserveTags map[string]bool) (string, error) {
	nodes, err := html.ParseFragment(strings.NewReader(input), &html.Node{
		Type:     html.ElementNode,
		Data:     "body",
		DataAtom: atom.Body,
	})
	if err != nil {
		return "", fmt.Errorf("parse html fragment: %w", err)
	}

	// Create a buffer for sanitized html.
	buf, ok := bufPool.Get().(*bytes.Buffer)
	if !ok {
		return "", errors.New("unable to retrieve buffer")
	}
	buf.Reset()
	defer bufPool.Put(buf)

	for _, n := range nodes {
		stripWalk(n, preserveTags)
		if err := html.Render(buf, n); err != nil {
			return "", fmt.Errorf("render html fragment: %w", err)
		}
	}
	return buf.String(), nil
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

// StripWhitespaceOnlyElements walks the tree post-order (children first), removing any element node whose full text
// content is blank. The root node itself is never removed.
func StripWhitespaceOnlyElements(input string) (string, error) {
	trimmed := strings.TrimSpace(input)
	lower := strings.ToLower(trimmed)

	var root *html.Node

	if isFullDocument := strings.HasPrefix(lower, "<!doctype") ||
		strings.Contains(lower, "<html"); isFullDocument {
		var err error
		root, err = html.Parse(strings.NewReader(input))
		if err != nil {
			return "", fmt.Errorf("parse html: %w", err)
		}
	} else {
		nodes, err := html.ParseFragment(strings.NewReader(input), &html.Node{
			Type:     html.ElementNode,
			Data:     "body",
			DataAtom: atom.Body,
		})
		if err != nil {
			return "", fmt.Errorf("parse html fragment: %w", err)
		}
		wrapper := &html.Node{Type: html.ElementNode, Data: "body"}
		for n := range slices.Values(nodes) {
			wrapper.AppendChild(n)
		}
		root = wrapper
	}

	stripWhitespaceFromNode(root)

	var buf bytes.Buffer
	for c := root.FirstChild; c != nil; c = c.NextSibling {
		if err := html.Render(&buf, c); err != nil {
			return "", fmt.Errorf("render html: %w", err)
		}
	}
	return buf.String(), nil
}

func stripWhitespaceFromNode(node *html.Node) {
	// Snapshot children first since removal mutates sibling pointers
	var children []*html.Node
	for c := node.FirstChild; c != nil; c = c.NextSibling {
		children = append(children, c)
	}

	for child := range slices.Values(children) {
		if child.Type == html.ElementNode {
			StripWhitespaceOnlyElements(child.Data) // recurse first (post-order)
			if isBlank(textContent(child)) {
				removeNode(child)
			}
		}
	}
}
