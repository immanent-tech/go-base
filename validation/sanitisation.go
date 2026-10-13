/*
 * Copyright (c) 2026 Immanent Tech
 * SPDX-License-Identifier: AGPL-3.0-or-later
 */

package validation

import (
	"bytes"
	"fmt"
	"html"
	"net/url"
	"strings"

	"github.com/microcosm-cc/bluemonday"
)

var safePrinter = bluemonday.UGCPolicy()

// SanitizeString attempts to "sanitize" a string value. It will strip any leading/trailing whitespace and then run the
// string through bluemonday to remove dangerous components. This should retain HTML5 content.
func SanitizeString(str string) string {
	return safePrinter.Sanitize(strings.TrimSpace(str))
}

// SanitizeInput first runs the string through SanitizeString, then unescapes any HTML encoding of the string.
func SanitizeInput(str string) string {
	return html.UnescapeString(SanitizeString(str))
}

// SanitizeURLString attempts to "sanitize" a string value representing a URL. It first runs the string through
// SanitizeString, then unescapes the any path, then parses the URL and returns its string representation.
func SanitizeURLString(str string) (string, error) {
	safeStr := SanitizeString(str)
	var err error
	safeStr, err = url.PathUnescape(safeStr)
	if err != nil {
		return "", fmt.Errorf("unescape URL path: %w", err)
	}
	parsedURL, err := url.Parse(safeStr)
	if err != nil {
		return "", fmt.Errorf("parse URL: %w", err)
	}
	return parsedURL.String(), nil
}

// SanitizeBytes attempts to "sanitize" a []byte value. It will strip any leading/trailing whitespace and then run the
// string through bluemonday to remove dangerous components. This should retain HTML5 content.
func SanitizeBytes(data []byte) []byte {
	return safePrinter.SanitizeBytes(bytes.TrimSpace(data))
}
