// Copyright 2026 Joshua Rich <joshua.rich@gmail.com>.
// SPDX-License-Identifier: 	AGPL-3.0-or-later

package htmlx

import (
	"strings"

	"golang.org/x/net/html"
)

// CheckMediumSignals walks the HTML tree and counts Medium-specific markers.
func CheckMediumSignals(node *html.Node) int {
	count := 0

	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.ElementNode {
			switch node.Data {
			case "meta":
				name := attrVal(node, "name")
				content := attrVal(node, "content")
				property := attrVal(node, "property")

				// Primary signal: Medium's generator tag
				if strings.EqualFold(name, "generator") &&
					strings.EqualFold(content, "Medium") {
					count += 10 // strong signal
				}

				// Medium's Apollo client state hint
				if strings.EqualFold(name, "medium-lite-url") {
					count += 5
				}

				// Open Graph site name often set to "Medium"
				if strings.EqualFold(property, "og:site_name") &&
					strings.EqualFold(content, "Medium") {
					count++
				}

			case "link":
				// Medium serves assets from miro.medium.com
				if href := attrVal(node, "href"); strings.Contains(href, "miro.medium.com") ||
					strings.Contains(href, "medium.com") {
					count++
				}

			case "script":
				// Medium's JS bundles come from medium.com
				if src := attrVal(node, "src"); strings.Contains(src, "medium.com") ||
					strings.Contains(src, "miro.medium.com") {
					count++
				}
			}
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(node)
	return count
}

func attrVal(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if strings.EqualFold(a.Key, key) {
			return a.Val
		}
	}
	return ""
}
