/*
 * Copyright (c) 2026 Immanent Tech
 * SPDX-License-Identifier: AGPL-3.0-or-later
 */

package comparisons

import (
	"fmt"
	"net/url"
	"strings"
)

// SameURLs reports whether a and b contain the same set of URLs, ignoring query strings, scheme/host case, trailing
// slashes, and the order/duplicates of entries.
func SameURLs(a, b []string) (bool, error) {
	setA, err := normalizeURLs(a)
	if err != nil {
		return false, err
	}
	setB, err := normalizeURLs(b)
	if err != nil {
		return false, err
	}

	if len(setA) != len(setB) {
		return false, nil
	}
	for u := range setA {
		if _, ok := setB[u]; !ok {
			return false, nil
		}
	}
	return true, nil
}

func normalizeURLs(urls []string) (map[string]struct{}, error) {
	set := make(map[string]struct{}, len(urls))
	for _, raw := range urls {
		n, err := normalizeURL(raw)
		if err != nil {
			return nil, err
		}
		set[n] = struct{}{}
	}
	return set, nil
}

func normalizeURL(raw string) (string, error) {
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
