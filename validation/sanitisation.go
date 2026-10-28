/*
 * Copyright (c) 2026 Immanent Tech
 * SPDX-License-Identifier: AGPL-3.0-or-later
 */

package validation

import (
	"errors"
	"fmt"
	"html"
	"net/url"
	"slices"
	"strings"
	"sync"
	"unicode"

	"github.com/microcosm-cc/bluemonday"
)

const (
	// DefaultMaxRunes applies to text sanitization. It defines a default number of runes to truncate a string to if no value is given.
	DefaultMaxRunes = 300
	// DefaultEscapePasses defines a max number of passes to attempt to unescape a string before giving up.
	DefaultEscapePasses = 5
)

var AllowedURLSchemes = []string{"http", "https", "ftp", "mailto"}

var policy = sync.OnceValue(func() *bluemonday.Policy {
	return bluemonday.UGCPolicy().
		AllowURLSchemes(AllowedURLSchemes...). // match schemes allowed in the feed specs.
		RequireNoFollowOnLinks(true).          // add nofollow on href.
		RequireNoReferrerOnLinks(true).        // add noreferer on href.
		AllowAttrs("src").OnElements("img").   // allow src on image.
		AllowElements("svg")                   // allow svg.
})

// SanitizeHTML sanitizes strings containing HTML. It performs:
//
// bluemonday -> html unescaping -> trimspace
//
// The resulting string should still contain valid HTML, but will be much safer to use.
func SanitizeHTML(str string) string {
	var err error
	str, err = fullyUnescape(strings.TrimSpace(str), 5)
	if err != nil {
		return ""
	}
	return policy().Sanitize(str)
}

// SanitizeURLString attempts to "sanitize" a string value representing a URL. It performs:
//
// path unescaping -> url parsing -> scheme validation
//
// It does not perform other validation, such as checking for absolute vs relative, host, etc.
func SanitizeURLString(str string) (string, error) {
	var (
		safeStr string
		err     error
	)
	safeStr, err = url.PathUnescape(str)
	if err != nil {
		return "", fmt.Errorf("unescape URL path: %w", err)
	}
	parsedURL, err := url.Parse(safeStr)
	if err != nil {
		return "", fmt.Errorf("parse URL: %w", err)
	}
	if !slices.Contains(AllowedURLSchemes, parsedURL.Scheme) {
		return "", fmt.Errorf("scheme is not allowed: %s", parsedURL.Scheme)
	}
	if parsedURL.Scheme != "mailto" && (parsedURL.Host == "" || parsedURL.User != nil) {
		return "", errors.New("contains illegal elements")
	}
	return parsedURL.String(), nil
}

// SanitizeText prepares a plain-text field:
//
// strips control bidi-override and zero-width characters -> replaces newlines -> and caps the length.
//
// Output is still untrusted and must be escaped.
func SanitizeText(s string, maxRunes int) string {
	s = strings.ToValidUTF8(s, "\uFFFD")
	var b strings.Builder
	n := 0
	prevSpace := true // true at the start, so leading whitespace is dropped
	for _, r := range s {
		switch {
		case unicode.IsSpace(r): // \n \r \t, NBSP, U+2028/2029, ideographic space...
			if prevSpace {
				continue
			}
			r = ' '
			prevSpace = true
		case unicode.IsControl(r):
			continue
		case r >= 0x202A && r <= 0x202E, r >= 0x2066 && r <= 0x2069,
			r == 0x200B, r == 0x200C, r == 0x200E, r == 0x200F,
			r == 0x2060, r == 0xFEFF:
			continue // keeps U+200D (ZWJ) so emoji sequences survive
		default:
			prevSpace = false
		}
		b.WriteRune(r)
		n++
		if maxRunes > 0 {
			if n >= maxRunes {
				break
			}
		}
	}
	return strings.TrimSpace(b.String())
}

// SanitizeInput wraps SanitizeText with the DefaultMaxRunes. It should be able to sanitize most inputs that are
// short-form text.
func SanitizeInput(s string) string {
	return SanitizeText(s, DefaultMaxRunes)
}

// fullUnEscape handles nested escaped strings. It will unescape continually up to maxPasses times, or until the string
// is no longer escaped. After maxPasses, an empty string is returned.
func fullyUnescape(s string, maxPasses int) (string, error) {
	for range maxPasses {
		u := html.UnescapeString(s)
		if u == s {
			return s, nil
		}
		s = u
	}
	return "", errors.New("too many layers of escaping")
}
