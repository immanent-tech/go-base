/*
 * Copyright (c) 2026 Immanent Tech
 * SPDX-License-Identifier: AGPL-3.0-or-later
 */

package redirect

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// To performs route redirection for routes that have moved. It will redirect the request to the given target with the
// given status code. URL parameters are preserved.
func To(target string, code int) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		dest := target
		if req.URL.RawQuery != "" {
			dest += "?" + req.URL.RawQuery
		}
		http.Redirect(res, req, dest, code)
	}
}

// Param performs route redirection for routes with parameters that have moved. Param specifies the original parameter
// while targetTmpl specifies a new path with as a fmt specifier that will substitute the parameter in.
//
// For example:
//
//	Param("foo","/bar/%s",303)
//
// Will read the the value of the URL parameter `foo` and substitute append it to `/bar/`. So if `foo=articles`, this
// would redirect to `/bar/articles` with a 303 status.
func Param(paramName, targetTmpl string, code int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		val := chi.URLParam(r, paramName)
		dest := fmt.Sprintf(targetTmpl, val)
		if r.URL.RawQuery != "" {
			dest += "?" + r.URL.RawQuery
		}
		http.Redirect(w, r, dest, code)
	}
}
