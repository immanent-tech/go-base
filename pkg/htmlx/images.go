// Copyright 2026 Joshua Rich <joshua.rich@gmail.com>.
// SPDX-License-Identifier: 	AGPL-3.0-or-later

package htmlx

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"slices"
	"strings"

	"codeberg.org/readeck/go-readability/v2"
	"github.com/immanent-tech/go-base/client"
	slogctx "github.com/veqryn/slog-context"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

var imageExtensions = []string{"jpg", "jpeg", "png", "webp", "gif", "avif", "svg", "tiff", "bmp"}

// ExtractImage searches the given HTML string and returns the URL and alt tag of the first image it finds.
func ExtractImage(input, pageURL string) (string, string, error) {
	if !IsHTML(input) {
		return "", "", fmt.Errorf("%w: content is not HTML", ErrParseHTML)
	}
	var root *html.Node

	// Parse as either full or partial HTML document.
	trimmed := strings.TrimSpace(input)
	lower := strings.ToLower(trimmed)
	if isFullDocument := strings.HasPrefix(lower, "<!doctype") ||
		strings.Contains(lower, "<html"); isFullDocument {
		var err error
		root, err = html.Parse(strings.NewReader(input))
		if err != nil {
			return "", "", fmt.Errorf("parse html: %w", err)
		}
	} else {
		nodes, err := html.ParseFragment(strings.NewReader(input), &html.Node{
			Type:     html.ElementNode,
			Data:     "body",
			DataAtom: atom.Body,
		})
		if err != nil {
			return "", "", fmt.Errorf("parse html fragment: %w", err)
		}
		wrapper := &html.Node{Type: html.ElementNode, Data: "body"}
		for n := range slices.Values(nodes) {
			wrapper.AppendChild(n)
		}
		root = wrapper
	}

	var rawURL, alt string

	for n := range root.Descendants() {
		if n.Type == html.ElementNode && n.DataAtom == atom.Img {
			for a := range slices.Values(n.Attr) {
				switch a.Key {
				case "src":
					rawURL = a.Val
				case "alt":
					alt = a.Val
				}
			}

			if rawURL != "" {
				imgURL, err := url.Parse(rawURL)
				if err != nil {
					return "", "", fmt.Errorf("parse image URL: %w", err)
				}
				// If it is not an absolute URL, resolve it relative to the page URL.
				if !imgURL.IsAbs() {
					absURL, _ := url.Parse(pageURL)
					return absURL.ResolveReference(imgURL).String(), alt, nil
				}
			}
		}
	}

	return "", "", fmt.Errorf("%w: no image found", ErrParseHTML)
}

// FindMainImage will attempt to extract a URL to what is likely the "main" image of a page (i.e., typically used on
// article/post pages).
func FindMainImage(ctx context.Context, rawURL string) (string, error) {
	pageBuf, err := GetHTML(ctx, rawURL)
	if err != nil {
		return "", fmt.Errorf("get %s: %w", rawURL, err)
	}

	var foundURL string

	// Try to parse opengraph data out of the page content.
	if og, err := DecodeOpengraph(bytes.NewReader(pageBuf.Bytes())); err != nil {
		slogctx.FromCtx(ctx).Debug("Could not parse opengraph data for URL.",
			slog.String("url", rawURL),
			slog.Any("error", err))
	} else {
		foundURL = og.Image
	}

	// Try to find the "main" image in the page content.
	if foundURL == "" {
		foundURL, _ = findMainImage(pageBuf.Bytes(), rawURL)
	}

	// Parse the found URL.
	imgURL, err := url.Parse(foundURL)
	if err != nil {
		return foundURL, fmt.Errorf("parse image URL %q: %w", foundURL, err)
	}

	// Check it points to an actual image.
	if !slices.ContainsFunc(imageExtensions, func(ext string) bool {
		return strings.HasSuffix(imgURL.Path, ext)
	}) {
		return "", errors.New("invalid image extension")
	}

	// If it is not an absolute URL, resolve it relative to the page URL.
	if !imgURL.IsAbs() {
		sourceURL, _ := url.Parse(rawURL)
		return sourceURL.ResolveReference(imgURL).String(), nil
	}

	return imgURL.String(), nil
}

// findMainImage tries to find a "main" image for the page, using the readability parser.
func findMainImage(page []byte, rawURL string) (string, error) {
	pageURL, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("parse url: %w", err)
	}

	node, err := html.Parse(bytes.NewReader(page))
	if err != nil {
		return "", fmt.Errorf("parse html: %w", err)
	}
	// Parse using readability to find main content details.
	rdData, err := readability.FromDocument(node, pageURL)
	if err != nil {
		return "", fmt.Errorf("find image: %w", err)
	}
	if rdData.ImageURL() == "" {
		return "", errors.New("no main image found")
	}
	return rdData.ImageURL(), nil
}

// FindFavicon will attempt to extract a URL to what is likely the favicon of a page.
func FindFavicon(ctx context.Context, rawURL string) (string, error) {
	pageBuf, err := GetHTML(ctx, rawURL)
	if err != nil {
		return "", fmt.Errorf("get %s: %w", rawURL, err)
	}

	_, faviconURL, _, err := findBestFavicon(pageBuf.Bytes(), rawURL)
	if err != nil {
		return "", fmt.Errorf("find favicon: %w", err)
	}

	// Parse the found URL.
	imgURL, err := url.Parse(faviconURL)
	if err != nil {
		return faviconURL, fmt.Errorf("parse favicon URL %q: %w", faviconURL, err)
	}

	// If it is not an absolute URL, resolve it relative to the page URL.
	if !imgURL.IsAbs() {
		sourceURL, _ := url.Parse(rawURL)
		return sourceURL.ResolveReference(imgURL).String(), nil
	}

	return imgURL.String(), nil
}

// Favicon is a favicon link found in <head>.
type Favicon struct {
	href string
	rel  string // e.g. "icon", "apple-touch-icon", "shortcut icon"
	typ  string // e.g. "image/png"
	size string // e.g. "32x32"
}

// findFaviconCandidates fetches the page and parses <link> tags in <head> that look like favicon declarations, plus
// synthesises the conventional /favicon.ico path.
func findFaviconCandidates(page []byte) []Favicon {
	limited := NewHeadReader(bytes.NewReader(page), MaxHeaderSizeBytes)

	var candidates []Favicon
	for tokenizer := html.NewTokenizer(limited); ; {
		tt := tokenizer.Next()
		if tt == html.ErrorToken {
			break
		}
		if tt != html.StartTagToken && tt != html.SelfClosingTagToken {
			continue
		}
		tok := tokenizer.Token()
		if tok.Data == "body" {
			break
		}
		if tok.Data != "link" {
			continue
		}

		var rel, href, typ, size string
		for attr := range slices.Values(tok.Attr) {
			switch strings.ToLower(attr.Key) {
			case "rel":
				rel = strings.ToLower(attr.Val)
			case "href":
				href = attr.Val
			case "type":
				typ = attr.Val
			case "sizes":
				size = attr.Val
			}
		}

		if href == "" {
			continue
		}
		// Accept any rel that contains "icon"
		if !strings.Contains(rel, "icon") {
			continue
		}
		candidates = append(candidates, Favicon{href: href, rel: rel, typ: typ, size: size})
	}

	// Always append the conventional fallback last.
	candidates = append(candidates, Favicon{href: "/favicon.ico", rel: "conventional"})
	return candidates
}

// resolve turns a possibly relative href into an absolute URL based on the page origin.
func resolve(pageURL, href string) (string, error) {
	base, err := url.Parse(pageURL)
	if err != nil {
		return "", fmt.Errorf("parse url %s: %w", pageURL, err)
	}
	ref, err := url.Parse(href)
	if err != nil {
		return "", fmt.Errorf("parse url %s: %w", href, err)
	}
	return base.ResolveReference(ref).String(), nil
}

// findBestFaviconCandidate tries each candidate in order and returns the first one that responds with a 2xx status and
// a non-empty body.
func findBestFavicon(
	page []byte,
	pageURL string,
) ([]byte, string, Favicon, error) {
	// Find all favicon candidate URLs in the page.
	candidates := findFaviconCandidates(page)
	if len(candidates) == 0 {
		return nil, "", Favicon{}, errors.New("no favicon candidates found")
	}

	// Loop over favicon candidate URLs and return the first one found.
	for cand := range slices.Values(candidates) {
		abs, err := resolve(pageURL, cand.href)
		if err != nil {
			continue
		}
		client, err := client.Load()
		if err != nil {
			return nil, "", Favicon{}, fmt.Errorf("load client: %w", err)
		}
		resp, err := client.R().Get(abs)
		if err != nil {
			continue
		}
		if resp.StatusCode() < 200 || resp.StatusCode() >= 300 || len(resp.Body()) == 0 {
			continue
		}
		return resp.Body(), abs, cand, nil
	}
	return nil, "", Favicon{}, errors.New("no reachable favicon found")
}
