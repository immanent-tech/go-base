/*
 * Copyright (c) 2026 Immanent Tech
 * SPDX-License-Identifier: AGPL-3.0-or-later
 */

// Package breadcrumbs maintains a per-session trail of the pages a user has navigated through on a site, backed by
// github.com/alexedwards/scs.
//
// A chi middleware inspects the Referer header on every request:
//
//   - If the Referer is missing, unparseable, or points at a different host
//     (i.e. the user arrived from somewhere else entirely), the trail is
//     reset. This marks the start of a fresh visit.
//   - If the Referer is relative (no host component) or points at the same
//     host as the current request (i.e. the user navigated here from
//     another page on this site), it is appended to the trail as the most
//     recent breadcrumb.
//
// The trail can then be queried for the previous page, or for any page in the trail by index.package breadcrumbs
package breadcrumbs

import (
	"context"
	"encoding/gob"
	"net/http"
	"net/url"
	"strings"
)

// sessionKey is the key under which the breadcrumb trail is stored in the scs session.
const sessionKey = "breadcrumbs:trail"

func init() {
	gob.Register([]url.URL{})
}

type SessionManager interface {
	Get(ctx context.Context, key string) any
	Put(ctx context.Context, key string, value any)
}

// Manager maintains a breadcrumb trail of page URLs, backed by an scs session store. It is safe for concurrent use,
// since all state lives in the (per-request) session, not on the Manager itself.
type Manager struct {
	Session SessionManager

	// HostFunc determines the "local" host for a given request. It is used to decide whether a Referer header points at
	// this site (local) or somewhere else (non-local). Defaults to defaultHostFunc, which uses r.Host and falls back to
	// the X-Forwarded-Host header when present (useful behind a reverse proxy).
	HostFunc func(r *http.Request) string
}

// New creates a breadcrumbs Manager backed by the given scs session manager.
func New(session SessionManager) *Manager {
	return &Manager{
		Session:  session,
		HostFunc: defaultHostFunc,
	}
}

func defaultHostFunc(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-Host"); fwd != "" {
		return fwd
	}
	return r.Host
}

// Recorder returns chi-compatible middleware that records the current request's Referer header into the breadcrumb
// trail (or resets the trail, per the rules described in the package doc).
func (m *Manager) Recorder(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m.record(r)
		next.ServeHTTP(w, r)
	})
}

// record inspects r's Referer header and updates the trail accordingly.
func (m *Manager) record(r *http.Request) {
	ctx := r.Context()
	raw := strings.TrimSpace(r.Header.Get("Referer"))

	if raw == "" {
		// No referrer at all: direct navigation, bookmark, new tab, etc. Treat as the start of a fresh trail.
		m.Reset(ctx)
		return
	}

	ref, err := url.Parse(raw)
	if err != nil {
		// Unparseable referrer - don't trust it, and don't carry over a stale trail either.
		m.Reset(ctx)
		return
	}

	local := ref.Host == "" || strings.EqualFold(ref.Host, m.HostFunc(r))
	if !local {
		// Referer points at a different host - the user arrived from elsewhere on the web, so start a new trail.
		m.Reset(ctx)
		return
	}

	m.push(ctx, ref)
}

// Reset clears the breadcrumb trail. It's called automatically whenever a non-local (or missing) Referer is seen, but
// can also be called manually, e.g. from a "start over" or logout handler.
func (m *Manager) Reset(ctx context.Context) {
	m.Session.Put(ctx, sessionKey, []url.URL{})
}

// push appends a page to the trail.
func (m *Manager) push(ctx context.Context, page *url.URL) {
	trail := append(m.All(ctx), *page)
	m.Session.Put(ctx, sessionKey, trail)
}

// All returns the full breadcrumb trail, oldest first. It never returns nil - an empty trail is an empty (non-nil)
// slice.
func (m *Manager) All(ctx context.Context) []url.URL {
	trail, ok := m.Session.Get(ctx, sessionKey).([]url.URL)
	if !ok || trail == nil {
		return []url.URL{}
	}
	return trail
}

// Len reports how many breadcrumbs are currently stored.
func (m *Manager) Len(ctx context.Context) int {
	return len(m.All(ctx))
}

// Previous returns the most recently recorded breadcrumb - i.e. the page the user navigated from to reach the current
// request - and reports whether one was available. This is equivalent to At(ctx, Len(ctx)-1).
func (m *Manager) Previous(ctx context.Context) (*url.URL, bool) {
	trail := m.All(ctx)
	if len(trail) == 0 {
		return nil, false
	}
	return &trail[len(trail)-1], true
}

// At returns the breadcrumb at the given index within the current trail. Index 0 is the oldest breadcrumb since the
// trail was last reset (i.e. the first local page the user came from after arriving on the site), and Len(ctx)-1 is the
// most recent (equivalent to Previous). It reports false if index is out of range.
func (m *Manager) At(ctx context.Context, index int) (*url.URL, bool) {
	trail := m.All(ctx)
	if index < 0 || index >= len(trail) {
		return nil, false
	}
	return &trail[index], true
}
