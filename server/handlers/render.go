// Copyright 2026 Joshua Rich <joshua.rich@gmail.com>.
// SPDX-License-Identifier: 	AGPL-3.0-or-later

//nolint:iface // duplication is more for readability than simplicity.
package handlers

import (
	"net/http"

	"github.com/a-h/templ"
	slogctx "github.com/veqryn/slog-context"

	"github.com/immanent-tech/go-base/pkg/htmx"
	"github.com/immanent-tech/go-base/pkg/templx"
)

// PartialResponseHandler is a handler that handles partial responses.
type PartialResponseHandler interface {
	PartialResponse(w http.ResponseWriter, r *http.Request)
}

// FullResponseHandler is a handler that handles full responses.
type FullResponseHandler interface {
	FullResponse(w http.ResponseWriter, r *http.Request)
}

// InternalPage is a handler for internal pages. Internal pages support rendering either full or partial responses.
type InternalPage interface {
	PartialResponseHandler
	FullResponseHandler
}

// ExternalPage is a handler for external pages. External pages only support rendering full responses.
type ExternalPage interface {
	FullResponseHandler
}

// handleNilContent logs and writes an appropriate error response when a handler was invoked with nil content. This
// should not happen in normal operation — it indicates a route was wired up without a backing handler — so it's treated
// as a server error rather than a legitimate "nothing to return" case, and logged at Error level to make the
// misconfiguration visible.
//
// Returns true if it wrote a response (i.e. content was nil) so callers can short-circuit.
func handleNilContent(res http.ResponseWriter, req *http.Request, content any) bool {
	if content != nil {
		return false
	}
	slogctx.FromCtx(req.Context()).Error("Render handler invoked with nil content; check route wiring.")
	http.Error(res, "internal server error", http.StatusInternalServerError)
	return true
}

// wantsFullPage reports whether a request should receive a full-page render instead of an HTMX partial.
//
// Non-HTMX requests always get the full page, since there's no HTMX runtime on the other end to swap a fragment into.
// History-restore requests also get the full page even though they originate from HTMX: when the browser restores a
// page from bfcache, it needs the complete document to repopulate correctly rather than a fragment with nothing to swap
// into.
func wantsFullPage(req *http.Request) bool {
	return !htmx.IsHTMX(req) || htmx.IsHistoryRestoreRequest(req)
}

// RenderInternalPage is a handler that chooses the appropriate rendering for an internal page (full or partial), based
// on the request.
func RenderInternalPage(content InternalPage) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		if handleNilContent(res, req, content) {
			return
		}
		if wantsFullPage(req) {
			content.FullResponse(res, req)
			return
		}
		content.PartialResponse(res, req)
	}
}

// RenderExternalPage is a handler that renders a full external page.
func RenderExternalPage(content ExternalPage) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		if handleNilContent(res, req, content) {
			return
		}
		content.FullResponse(res, req)
	}
}

// RenderPartial is a handler that renders a partial response only.
//
// Unlike RenderInternalPage, this has no full-page fallback: it's intended for routes that only ever make sense as an
// HTMX fragment (e.g. out-of-band swap targets, polling endpoints). A non-HTMX request against one of these routes is
// treated as a client error rather than silently rendered as a full page.
func RenderPartial(content PartialResponseHandler) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		if handleNilContent(res, req, content) {
			return
		}
		if !htmx.IsHTMX(req) {
			res.WriteHeader(http.StatusNotAcceptable)
			return
		}
		content.PartialResponse(res, req)
	}
}

// renderFragmentAware renders a templ.Component, honoring any fragment keys present on the request context (e.g. for
// HTMX OOB / targeted fragment rendering). Shared by any type that needs to render a partial from a plain
// templ.Component.
func renderFragmentAware(component templ.Component, res http.ResponseWriter, req *http.Request) {
	if fragments := templx.FragmentKeysFromCtx(req.Context()); len(fragments) > 0 {
		templ.Handler(component, templ.WithFragments(fragments)).ServeHTTP(res, req)
		return
	}
	templ.Handler(component).ServeHTTP(res, req)
}

// PartialTemplate is a template that only supports being rendered in a partial response.
type PartialTemplate struct {
	template templ.Component
}

// PartialResponse renders the template.
func (t *PartialTemplate) PartialResponse(res http.ResponseWriter, req *http.Request) {
	renderFragmentAware(t.template, res, req)
}

// Page composes a partial view with a layout to satisfy InternalPage, so most pages that are just "this fragment,
// optionally wrapped in the app layout" don't need a bespoke type.
//
//	RenderInternalPage(&Page{Partial: dashboardView(), Layout: layout.App})
type Page struct {
	// Partial is the inner content, rendered as-is for HTMX requests.
	Partial templ.Component
	// Layout wraps Partial for full-page (non-HTMX) requests.
	Layout func(content templ.Component) templ.Component
}

// PartialResponse renders Partial directly, honoring fragment keys.
func (p *Page) PartialResponse(res http.ResponseWriter, req *http.Request) {
	renderFragmentAware(p.Partial, res, req)
}

// FullResponse renders Partial wrapped in Layout.
func (p *Page) FullResponse(res http.ResponseWriter, req *http.Request) {
	templ.Handler(p.Layout(p.Partial)).ServeHTTP(res, req)
}
