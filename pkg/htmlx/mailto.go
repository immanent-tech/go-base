// Copyright 2026 Joshua Rich <joshua.rich@gmail.com>.
// SPDX-License-Identifier: 	AGPL-3.0-or-later

package htmlx

import (
	"net/url"
	"slices"
	"strings"
)

// MailTo represents a link that will open the user's mail client, with optionall pre-filled details).
type MailTo struct {
	parts []string
}

// NewMailTo builds a new `mailto:` link with the given options.
func NewMailTo(to string, options ...MailToOption) string {
	mtl := &MailTo{}
	for option := range slices.Values(options) {
		option(mtl)
	}
	var builder strings.Builder
	builder.WriteString("mailto:")
	builder.WriteString(to)
	if len(mtl.parts) > 0 {
		builder.WriteString("?")
		builder.WriteString(strings.Join(mtl.parts, "&"))
	}

	return builder.String()
}

// MailToOption is a functional option to apply to a mailto: link object.
type MailToOption func(*MailTo)

// WithMailToSubject option adds a subject to the mailto: link.
func WithMailToSubject(subject string) MailToOption {
	return func(mtl *MailTo) {
		mtl.parts = append(mtl.parts, "subject="+url.QueryEscape(subject))
	}
}

// WithMailToBody option adds body text to the mailto: link.
func WithMailToBody(body string) MailToOption {
	return func(mtl *MailTo) {
		mtl.parts = append(mtl.parts, "body="+url.QueryEscape(body))
	}
}
