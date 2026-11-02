/*
 * Copyright (c) 2026 Immanent Tech
 * SPDX-License-Identifier: AGPL-3.0-or-later
 */

package htmlx

import (
	"fmt"
	"io"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

// Attributes whose value is a single URL.
var urlAttrs = map[string]bool{
	"href":       true,
	"src":        true,
	"action":     true,
	"formaction": true,
	"poster":     true,
	"data":       true,
	"cite":       true,
	"background": true,
	"longdesc":   true,
	"manifest":   true,
}

// Attributes whose value is a comma-separated list of "url [descriptor]".
var srcsetAttrs = map[string]bool{
	"srcset": true,
}

// MakeLinksAbsolute takes a full or partial HTML document and a base URL, and
// returns the document with every relative URL replaced by an absolute one.
//
// It uses a streaming tokenizer, so partial documents are not wrapped in
// <html>/<body>, and any tag that doesn't need changing is emitted
// byte-for-byte as it was in the input.
//
// Left untouched: URLs that already have a scheme (http:, https:, mailto:,
// tel:, data:, javascript:, ...), empty values, and fragment-only links ("#top").
func MakeLinksAbsolute(htmlDoc, baseURL string) (string, error) {
	base, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil {
		return "", fmt.Errorf("invalid base url: %w", err)
	}
	if !base.IsAbs() {
		return "", fmt.Errorf("base url %q must be absolute (include a scheme)", baseURL)
	}

	z := html.NewTokenizer(strings.NewReader(htmlDoc))
	var out strings.Builder
	out.Grow(len(htmlDoc))

	for {
		tt := z.Next()
		if tt == html.ErrorToken {
			if err := z.Err(); err != io.EOF {
				return "", err
			}
			return out.String(), nil
		}

		// Raw's backing slice is invalidated by later tokenizer calls, so use
		// it (or copy it) before calling Token().
		raw := string(z.Raw())

		if tt != html.StartTagToken && tt != html.SelfClosingTagToken {
			out.WriteString(raw)
			continue
		}

		tok := z.Token()
		changed := false
		for i, a := range tok.Attr {
			// Skip namespaced attributes (e.g. xlink:href is handled below).
			key := strings.ToLower(a.Key)
			if a.Namespace != "" {
				key = strings.ToLower(a.Namespace + ":" + a.Key)
			}

			var newVal string
			switch {
			case urlAttrs[key], key == "xlink:href":
				newVal = resolve(base, a.Val)
			case srcsetAttrs[key]:
				newVal = resolveSrcset(base, a.Val)
			default:
				continue
			}
			if newVal != a.Val {
				tok.Attr[i].Val = newVal
				changed = true
			}
		}

		if !changed {
			out.WriteString(raw)
			continue
		}
		writeTag(&out, tok)
	}
}

// writeTag re-serializes a start/self-closing tag with (possibly modified) attributes.
func writeTag(sb *strings.Builder, t html.Token) {
	sb.WriteByte('<')
	sb.WriteString(t.Data)
	for _, a := range t.Attr {
		sb.WriteByte(' ')
		if a.Namespace != "" {
			sb.WriteString(a.Namespace)
			sb.WriteByte(':')
		}
		sb.WriteString(a.Key)
		sb.WriteString(`="`)
		sb.WriteString(html.EscapeString(a.Val))
		sb.WriteByte('"')
	}
	if t.Type == html.SelfClosingTagToken {
		sb.WriteString("/>")
	} else {
		sb.WriteByte('>')
	}
}
