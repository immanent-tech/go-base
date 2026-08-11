// Copyright 2026 Joshua Rich <joshua.rich@gmail.com>.
// SPDX-License-Identifier: 	AGPL-3.0-or-later

package htmlx

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"sync"

	"github.com/immanent-tech/go-base/client"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

var (
	MaxHeaderSizeBytes       = 256 * 1024
	MaxBodySizeBytes   int64 = 10 * 1024 * 1024 // 10 MB limit
)

// Common HTML tags.
var htmlTags = []string{
	// Document structure
	"html", "head", "body", "doctype",
	// Metadata
	"title", "meta", "link", "style", "script",
	// Sectioning
	"header", "footer", "main", "nav", "section", "article", "aside",
	// Block elements
	"div", "p", "h1", "h2", "h3", "h4", "h5", "h6",
	"ul", "ol", "li", "dl", "dt", "dd",
	"table", "thead", "tbody", "tfoot", "tr", "th", "td",
	"form", "fieldset", "legend", "label",
	"blockquote", "pre", "figure", "figcaption", "hr",
	// Inline elements
	"a", "span", "strong", "em", "b", "i", "u", "s",
	"img", "input", "button", "select", "textarea", "option",
	"code", "kbd", "samp", "var", "abbr", "cite",
	"br", "small", "sub", "sup",
	// Media / embedded
	"video", "audio", "canvas", "iframe", "svg",
}

var (
	// Matches a DOCTYPE declaration.
	doctypeRe = regexp.MustCompile(`(?i)<!DOCTYPE\s+html`)
	// Matches HTML comments.
	commentRe = regexp.MustCompile(`<!--[\s\S]*?-->`)
	// Matches any opening or closing HTML tag from our known list,
	// with optional attributes. e.g. <div>, <a href="...">, </p>.
	tagPattern = buildTagPattern()
)

var (
	ErrNotFound  = errors.New("not found")
	ErrParseURL  = errors.New("could not parse URL")
	ErrParseHTML = errors.New("could not parse HTML")
)

func buildTagPattern() *regexp.Regexp {
	joined := strings.Join(htmlTags, "|")
	// Match opening tags (with optional attrs) or closing tags
	pattern := `(?i)<(/?)(?:` + joined + `)(?:\s[^>]*)?>`
	return regexp.MustCompile(pattern)
}

// HeadReader wraps a reader and stops after the </head> tag or a byte limit. This avoids downloading the entire page
// body.
type HeadReader struct {
	r       io.Reader
	buf     []byte
	done    bool
	total   int
	maxRead int
}

func NewHeadReader(r io.Reader, maxBytes int) *HeadReader {
	return &HeadReader{r: r, maxRead: maxBytes}
}

func (h *HeadReader) Read(page []byte) (int, error) {
	if h.done {
		return 0, io.EOF
	}
	if h.total >= h.maxRead {
		return 0, io.EOF
	}
	n, err := h.r.Read(page)
	if err != nil {
		return n, fmt.Errorf("read header: %w", err)
	}
	h.total += n
	// Look for </head> in what we just read to stop early
	chunk := strings.ToLower(string(page[:n]))
	if idx := strings.Index(chunk, "</head>"); idx != -1 {
		h.done = true
		return idx + len("</head>"), io.EOF
	}
	return n, nil
}

type Response struct {
	Status  int
	Message string
}

func (e *Response) Error() string { return fmt.Sprintf("%d: %s", e.Status, e.Message) }
func (e *Response) Unwrap() error { return fmt.Errorf("%d: %s", e.Status, e.Message) }

// HTTPStatus returns the status code of the API error.
func (e *Response) HTTPStatus() int { return e.Status }

// GetHTML will fetch the HTML source from the page at the given URL.
func GetHTML(ctx context.Context, strURL string) (*bytes.Buffer, error) {
	sourceURL, err := url.Parse(strURL)
	if err != nil {
		return nil, &Response{
			Status:  http.StatusBadRequest,
			Message: fmt.Sprintf("parse URL %s: %s", strURL, err.Error()),
		}
	}

	// Create a buffer for the feed data.
	var pageBuf bytes.Buffer

	client, err := client.Load()
	if err != nil {
		return nil, &Response{
			Status:  http.StatusInternalServerError,
			Message: fmt.Sprintf("load client: %s", err.Error()),
		}
	}

	resp, err := client.R().
		SetContext(ctx).
		SetDoNotParseResponse(true).
		Get(sourceURL.String())
	if err != nil {
		return nil, &Response{
			Status:  http.StatusInternalServerError,
			Message: fmt.Sprintf("fetch URL %s: %s", sourceURL.String(), err.Error()),
		}
	}
	if resp.IsError() || resp.StatusCode() == http.StatusNoContent {
		return nil, &Response{
			Status:  resp.StatusCode(),
			Message: fmt.Sprintf("fetch URL %s: %s", sourceURL.String(), resp.Status()),
		}
	}
	defer resp.RawBody().Close()
	if resp.Header().Get("Content-Encoding") == "gzip" {
		// For gzipped response, uncompress first.
		reader, err := gzip.NewReader(resp.RawBody())
		if err != nil {
			return nil, &Response{
				Status:  http.StatusUnprocessableEntity,
				Message: fmt.Sprintf("read gzip response: %s", err.Error()),
			}
		}
		defer reader.Close()
		limitReader := io.LimitReader(reader, MaxBodySizeBytes)
		if _, err := io.Copy(&pageBuf, limitReader); err != nil {
			return nil, &Response{
				Status:  http.StatusUnprocessableEntity,
				Message: fmt.Sprintf("read response: %s", err.Error()),
			}
		}
	} else {
		// Read response directly.
		if _, err := io.Copy(&pageBuf, resp.RawBody()); err != nil {
			return nil, &Response{
				Status:  http.StatusUnprocessableEntity,
				Message: fmt.Sprintf("read response: %s", err.Error()),
			}
		}
	}

	return &pageBuf, nil
}

// IsHTML returns a boolean indicating whether the given string contains HTML. It can detect both a full HTML document
// or partial HTML content.
func IsHTML(s string) bool {
	score := 0

	trimmed := strings.TrimSpace(s)
	if len(trimmed) == 0 {
		return false
	}

	lower := strings.ToLower(trimmed)

	// Signal 1: DOCTYPE declaration — very strong indicator
	if doctypeRe.MatchString(trimmed) {
		score += 40
	}

	// Signal 2: <html> tag present
	if strings.Contains(lower, "<html") {
		score += 30
	}

	// Signal 3: <head> + <body> structure

	if hasHead, hasBody := strings.Contains(lower, "<head"), strings.Contains(lower, "<body"); hasHead && hasBody {
		score += 20
	} else if hasHead || hasBody {
		score += 10
	}

	// Signal 4: Count known HTML tag matches
	matches := tagPattern.FindAllString(trimmed, -1)
	if tagCount := len(matches); tagCount >= 3 {
		switch {
		case tagCount >= 10:
			score += 30
		case tagCount >= 5:
			score += 20
		default:
			score += 10
		}
	} else if tagCount > 0 {
		score += 5
	}

	// Signal 5: HTML comment syntax
	if commentRe.MatchString(trimmed) {
		score += 10
	}

	// Signal 6: Common HTML attribute patterns (href, src, class, id, style)
	if regexp.MustCompile(`(?i)\s(?:href|src|class|id|style|alt|type|name|value|placeholder)\s*=\s*["']`).
		MatchString(trimmed) {
		score += 10
	}

	// Signal 7: Self-closing tags like <br/>, <img/>, <input/>
	if regexp.MustCompile(`(?i)<(?:br|hr|img|input|meta|link)\b[^>]*?/?>`).MatchString(trimmed) {
		score += 5
	}

	// Signal 8: Starts with a tag (strong partial HTML indicator)
	if strings.HasPrefix(trimmed, "<") && tagPattern.MatchString(trimmed[:min(50, len(trimmed))]) {
		score += 10
	}

	// Normalise score to a 0–1 confidence value (cap at 100 before dividing)
	if score > 100 {
		score = 100
	}
	confidence := float64(score) / 100.0
	return confidence >= 0.10 // low threshold — we want to catch partials
}

// IsHTMLElement returns a boolean indicating whether the given string is the given HTML element.
func IsHTMLElement(str, tag string) bool {

	switch pattern1, pattern2, trimmed := regexp.MustCompile(`(?i)<(/?)(?:`+tag+`)(?:\s[^>]*)?>`), regexp.MustCompile(`(?i)<(?:`+tag+`)\b[^>]*?/?>`), strings.TrimSpace(str); {
	case len(trimmed) == 0:
		return false
	case strings.HasPrefix(trimmed, "<") && pattern1.MatchString(trimmed[:min(50, len(trimmed))]):
		return true
	case pattern2.MatchString(trimmed):
		return true
	default:
		return false
	}
}

// FindHTMLNode does a depth-first search for the first node matching the tag.
func FindHTMLNode(n *html.Node, tag string) *html.Node {
	if n.Type == html.ElementNode && n.Data == tag {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if result := FindHTMLNode(c, tag); result != nil {
			return result
		}
	}
	return nil
}

// FindAllHTMLNodes returns all nodes matching the tag within n.
func FindAllHTMLNodes(n *html.Node, tag string) []*html.Node {
	var results []*html.Node
	if n.Type == html.ElementNode && n.Data == tag {
		results = append(results, n)
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		results = append(results, FindAllHTMLNodes(c, tag)...)
	}
	return results
}

// Sanitize performs sanitization of HTML. With no options specified, it ensures the input string parses correctly and
// returns well-formed HTML as a string. Additional options can be specified to perform additional sanitization steps,
// such as stripping all attributes from all elements and removing whitespace-only elements.
func Sanitize(input string, opts ...SanitizeOption) (string, error) {
	var root *html.Node

	// Parse as either full or partial HTML document.
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

var whitespaceRe = regexp.MustCompile(`\s+`)

// ToPlainText converts a HTML encoded string to plain text.
func ToPlainText(data []byte) (string, error) {
	tokenizer := html.NewTokenizer(bytes.NewReader(data))

	buf, ok := bufPool.Get().(*bytes.Buffer)
	if !ok {
		return "", errors.New("unable to retrieve buffer")
	}
	buf.Reset()
	defer bufPool.Put(buf)

	skipDepth := 0

	for {
		switch tt := tokenizer.Next(); tt {
		case html.ErrorToken:
			return whitespaceRe.ReplaceAllString(string(bytes.TrimSpace(buf.Bytes())), " "), nil

		case html.StartTagToken, html.SelfClosingTagToken:
			tok := tokenizer.Token()
			if skipContentTags[tok.Data] {
				skipDepth++
			}
			if blockTags[tok.Data] {
				buf.WriteString("\n\n")
			}
			writeAttrText(buf, tok, skipDepth)

		case html.EndTagToken:
			tok := tokenizer.Token()
			if skipContentTags[tok.Data] && skipDepth > 0 {
				skipDepth--
			}
			if blockTags[tok.Data] {
				buf.WriteString("\n\n")
			}

		case html.TextToken:
			if skipDepth == 0 {
				// tokenizer.Text() already decodes entities (&amp; -> &).
				buf.Write(tokenizer.Text())
				buf.WriteByte(' ')
			}
		}
	}
}

// skipContentTags holds elements whose text content should never be embedded (scripts, styles, and non-visible
// metadata).
var skipContentTags = map[string]bool{
	"script": true, "style": true, "noscript": true,
	"head": true, "title": true, "meta": true, "link": true,
}

// blockTags forces a paragraph break so ChunkBytes's "\n\n" boundary detection still finds sensible break points in the
// extracted text.
var blockTags = map[string]bool{
	"p": true, "div": true, "br": true, "li": true, "tr": true,
	"h1": true, "h2": true, "h3": true, "h4": true, "h5": true, "h6": true,
	"blockquote": true, "section": true, "article": true, "table": true,
}

// attrTextKeys lists attributes whose values carry human-readable content worth embedding, rather than machine-facing
// metadata (id, class, href, etc).
var attrTextKeys = map[string]bool{
	"alt": true,
	// "title":      true,
	"aria-label": true,
}

// writeAttrText pulls alt/title/aria-label off a tag and appends them to buf as plain text. These are written
// regardless of skipDepth for alt/ aria-label (an <img> inside a <div> isn't "script content"), but honoring skipDepth
// keeps them out of genuinely non-visible containers like <head>.
func writeAttrText(buf *bytes.Buffer, tok html.Token, skipDepth int) {
	if skipDepth > 0 {
		return
	}
	for _, attr := range tok.Attr {
		key := strings.ToLower(attr.Key)
		val := strings.TrimSpace(attr.Val)
		if attrTextKeys[key] && val != "" {
			buf.WriteString(val)
			buf.WriteString(". ")
		}
	}
}

var bufPool = sync.Pool{
	New: func() any {
		return new(bytes.Buffer)
	},
}
