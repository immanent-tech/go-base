/*
 * Copyright (c) 2026 Immanent Tech
 * SPDX-License-Identifier: AGPL-3.0-or-later
 */

package comparisons

import "github.com/immanent-tech/go-base/pkg/htmlx"

// SameURLs reports whether a and b contain the same set of URLs, ignoring query strings, scheme/host case, trailing
// slashes, and the order/duplicates of entries.
func SameURLs(a, b []string) (bool, error) {
	setA, err := htmlx.NormalizeURLs(a)
	if err != nil {
		return false, err
	}
	setB, err := htmlx.NormalizeURLs(b)
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
