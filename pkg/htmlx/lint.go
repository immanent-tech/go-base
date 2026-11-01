/*
 * Copyright (c) 2026 Immanent Tech
 * SPDX-License-Identifier: AGPL-3.0-or-later
 */

package htmlx

import (
	"bytes"
	"errors"
	"fmt"
	"net/url"
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

// Lint performs sanitization of HTML. With no options specified, it ensures the input string parses correctly and
// returns well-formed HTML as a string. Additional options can be specified to perform additional sanitization steps,
// such as stripping all attributes from all elements and removing whitespace-only elements.
func Lint(input string, opts ...SanitizeOption) (string, error) {
	// Parse as either full or partial HTML document.
	var root *html.Node
	trimmed := strings.TrimSpace(input)
	lower := strings.ToLower(trimmed)
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

	// Perform any requested sanitisation options.
	for option := range slices.Values(opts) {
		option(root)
	}

	// Create a buffer for sanitized HTML.
	buf, ok := bufPool.Get().(*bytes.Buffer)
	if !ok {
		return "", errors.New("unable to retrieve buffer")
	}
	buf.Reset()
	defer bufPool.Put(buf)

	// Write out sanitized HTML.
	for c := root.FirstChild; c != nil; c = c.NextSibling {
		if err := html.Render(buf, c); err != nil {
			return "", fmt.Errorf("render html: %w", err)
		}
	}
	return buf.String(), nil
}

// SanitizeOption is a functional option for performing a specific sanitization to the given HTML node.
type SanitizeOption func(node *html.Node)

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

// directRows returns the <tr> elements that belong directly to this table, without descending into any table nested
// inside a cell.
func directRows(table *html.Node) []*html.Node {
	var rows []*html.Node
	var walk func(n *html.Node)
	walk = func(node *html.Node) {
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			if child.Type != html.ElementNode {
				continue
			}
			switch child.Data {
			case "table":
				continue // a nested table's rows aren't this table's rows
			case "tr":
				rows = append(rows, child)
			default:
				walk(child)
			}
		}
	}
	walk(table)
	return rows
}

// directCells returns the <td>/<th> children of a single <tr>.
func directCells(row *html.Node) []*html.Node {
	var cells []*html.Node
	for c := row.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && (c.Data == "td" || c.Data == "th") {
			cells = append(cells, c)
		}
	}
	return cells
}

// isDataTable is a heuristic for "this table is actual tabular data and should be kept" vs. "this table exists only for
// layout and should be unwrapped". Layout tables in email HTML are almost always one cell per row (a vertical stack);
// real data tables have multiple columns and multiple rows.
func isDataTable(table *html.Node) bool {
	rows := directRows(table)
	if len(rows) < 2 {
		return false
	}
	maxCols := 0
	for _, row := range rows {
		if n := len(directCells(row)); n > maxCols {
			maxCols = n
		}
	}
	return maxCols >= 2
}

// spliceUnwrap replaces n with its own children, in place, inside n's parent.
func spliceUnwrap(node *html.Node) {
	parent := node.Parent
	if parent == nil {
		return
	}
	for c := node.FirstChild; c != nil; {
		next := c.NextSibling
		node.RemoveChild(c)
		parent.InsertBefore(c, node)
		c = next
	}
	parent.RemoveChild(node)
}

// stripStructural removes table/thead/tbody/tfoot/tr/td/th wrapper tags found under n, without crossing into any nested
// <table> — those are resolved separately (they may be data tables we want to keep intact).
func stripStructural(node *html.Node) {
	var structuralTags = map[string]bool{
		"table": true,
		"thead": true,
		"tbody": true,
		"tfoot": true,
		"tr":    true,
		"td":    true,
		"th":    true,
	}

	var children []*html.Node
	for c := node.FirstChild; c != nil; c = c.NextSibling {
		children = append(children, c)
	}
	for child := range slices.Values(children) {
		if child.Type != html.ElementNode {
			continue
		}
		if child.Data == "table" {
			continue // handled by the outer post-order walk
		}
		stripStructural(child) // strip deeper tags first
		if structuralTags[child.Data] {
			spliceUnwrap(child)
		}
	}
}

// unwrapTable flattens a layout-only table down to its meaningful content.
func unwrapTable(table *html.Node) {
	stripStructural(table)
	spliceUnwrap(table)
}

// UnwrapLayoutTables walks the tree post-order — innermost tables first — so arbitrarily deep nesting resolves
// correctly. Any table that doesn't look like real tabular data gets unwrapped; genuine data tables are left untouched
// (including any layout tables nested *inside* them, which are still unwrapped since children are always visited before
// their parent).
func UnwrapLayoutTables(root *html.Node) {
	var visit func(n *html.Node)
	visit = func(node *html.Node) {
		var children []*html.Node
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			children = append(children, c)
		}
		for child := range slices.Values(children) {
			visit(child)
		}
		if node.Type == html.ElementNode && node.Data == "table" {
			if !isDataTable(node) {
				unwrapTable(node)
			}
		}
	}
	visit(root)
}

var preheaderPadding = regexp.MustCompile(`[\x{00A0}\x{200C}\s]{10,}`)

// StripEmailPreheaderPadding removes excess whitespace that is often used for padding in HTML emails.
func StripEmailPreheaderPadding(html string) string {
	return preheaderPadding.ReplaceAllString(html, "")
}

// NormalizeURL performs normalization on the given URLs. This includes lowercasing scheme/host and removing query
// strings and trailing slashes.
func NormalizeURL(raw string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return "", fmt.Errorf("parsing %q: %w", raw, err)
	}

	// Scheme and host are case-insensitive (the path is not).
	u.Scheme = strings.ToLower(u.Scheme)
	u.Host = strings.ToLower(u.Host)

	// Drop the query string (including a dangling "?").
	u.RawQuery = ""
	u.ForceQuery = false

	// Strip trailing slashes, so "/a/" == "/a" and "http://x.com/" == "http://x.com".
	u.Path = strings.TrimRight(u.Path, "/")
	u.RawPath = strings.TrimRight(u.RawPath, "/")

	return u.String(), nil
}

func NormalizeURLs(urls []string) (map[string]struct{}, error) {
	set := make(map[string]struct{}, len(urls))
	for _, raw := range urls {
		n, err := NormalizeURL(raw)
		if err != nil {
			return nil, err
		}
		set[n] = struct{}{}
	}
	return set, nil
}
