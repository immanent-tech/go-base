// Copyright 2025 Joshua Rich <joshua.rich@gmail.com>.
// SPDX-License-Identifier: 	AGPL-3.0-or-later

package security

import (
	"log/slog"
	"net/http"
	"slices"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	slogchi "github.com/samber/slog-chi"
	slogctx "github.com/veqryn/slog-context"
)

// PreventCSRF middleware creates a CSRF handler middleware with the given options.
func PreventCSRF(options ...CSRFOption) func(next http.Handler) http.Handler {
	cop := http.NewCrossOriginProtection()

	WithCSRFDenyHandler(CSRFError())(cop)

	for option := range slices.Values(options) {
		option(cop)
	}

	return cop.Handler
}

// CSRFOption is a functional option applied to the CSRF middleware handler.
type CSRFOption func(*http.CrossOriginProtection)

// WithCSRFTrustedOrigin option allows all requests with an Origin header which exactly matches the given value.
func WithCSRFTrustedOrigin(origin string) CSRFOption {
	return func(cop *http.CrossOriginProtection) {
		cop.AddTrustedOrigin(origin)
	}
}

// WithCSRFInsecureBypassPattern option permits all requests that match the given pattern.
func WithCSRFInsecureBypassPattern(pattern string) CSRFOption {
	return func(cop *http.CrossOriginProtection) {
		cop.AddInsecureBypassPattern(pattern)
	}
}

// WithCSRFDenyHandler sets a handler to invoke when a request is rejected. The default if this option is not specified,
// responds with a 403 Forbidden status.
func WithCSRFDenyHandler(handler http.HandlerFunc) CSRFOption {
	return func(cop *http.CrossOriginProtection) {
		cop.SetDenyHandler(handler)
	}
}

// CSRFError handles CSRF error conditions. It will log details about the request then show an error page to the user.
func CSRFError() http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		params := make(map[string]string)
		if chi.RouteContext(req.Context()) != nil {
			if len(chi.RouteContext(req.Context()).URLParams.Keys) > 0 {
				for i, k := range chi.RouteContext(req.Context()).URLParams.Keys {
					params[k] = chi.RouteContext(req.Context()).URLParams.Values[i]
				}
			}
		}
		slogctx.FromCtx(req.Context()).Error("CSRF check failed",
			slog.String("method", req.Method),
			slog.String("host", req.Host),
			slog.String("path", req.URL.Path),
			slog.String("query", req.URL.RawQuery),
			slog.Any("params", params),
			slog.String("route", chi.RouteContext(req.Context()).RoutePattern()),
			slog.String("ip", req.RemoteAddr),
			slog.String("referer", req.Referer()),
			slog.String(slogchi.RequestIDKey, middleware.GetReqID(req.Context())),
		)
		http.Error(res, "CSRF Failed", http.StatusBadRequest)
	}
}
