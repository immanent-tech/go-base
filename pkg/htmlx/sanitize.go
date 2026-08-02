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
