/*
 * Copyright (c) 2026 Immanent Tech
 * SPDX-License-Identifier: AGPL-3.0-or-later
 */

package breadcrumbs

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

type SessionManagerTest map[string]any

func (m SessionManagerTest) Get(ctx context.Context, key string) any {
	val, found := m[key]
	if found {
		return val
	}
	return nil
}

func (m SessionManagerTest) Put(ctx context.Context, key string, value any) {
	m[key] = value
}

func newTestManager() *Manager {
	sm := make(SessionManagerTest)
	return New(sm)
}

// run wraps a request through session LoadAndSave + the breadcrumbs
// middleware, then hands the (now session-bearing) request to fn so
// assertions can run against Manager methods.
func run(t *testing.T, mgr *Manager, referer string, fn func(r *http.Request)) {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "http://example.com/current", nil)
	if referer != "" {
		req.Header.Set("Referer", referer)
	}
	rw := httptest.NewRecorder()

	handler := mgr.Recorder(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fn(r)
	}))
	handler.ServeHTTP(rw, req)
}

func TestLocalReferrerIsRecorded(t *testing.T) {
	mgr := newTestManager()

	run(t, mgr, "http://example.com/page-a", func(r *http.Request) {
		got, ok := mgr.Previous(r.Context())
		if !ok || got != "http://example.com/page-a" {
			t.Fatalf("expected page-a recorded, got %q ok=%v", got, ok)
		}
	})
}

func TestRelativeReferrerIsRecorded(t *testing.T) {
	mgr := newTestManager()

	run(t, mgr, "/page-a", func(r *http.Request) {
		got, ok := mgr.Previous(r.Context())
		if !ok || got != "/page-a" {
			t.Fatalf("expected /page-a recorded, got %q ok=%v", got, ok)
		}
	})
}

func TestExternalReferrerResetsTrail(t *testing.T) {
	mgr := newTestManager()

	run(t, mgr, "https://google.com/search?q=x", func(r *http.Request) {
		if n := mgr.Len(r.Context()); n != 0 {
			t.Fatalf("expected empty trail after external referrer, got len %d", n)
		}
	})
}

func TestMissingReferrerResetsTrail(t *testing.T) {
	mgr := newTestManager()

	run(t, mgr, "", func(r *http.Request) {
		if n := mgr.Len(r.Context()); n != 0 {
			t.Fatalf("expected empty trail with no referrer, got len %d", n)
		}
	})
}

func TestAtIndexOutOfRange(t *testing.T) {
	mgr := newTestManager()

	run(t, mgr, "/page-a", func(r *http.Request) {
		if _, ok := mgr.At(r.Context(), 5); ok {
			t.Fatalf("expected ok=false for out-of-range index")
		}
		if _, ok := mgr.At(r.Context(), -1); ok {
			t.Fatalf("expected ok=false for negative index")
		}
	})
}
