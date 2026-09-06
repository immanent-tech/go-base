// Copyright 2026 Joshua Rich <joshua.rich@gmail.com>.
// SPDX-License-Identifier: 	AGPL-3.0-or-later

package htmx

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"slices"

	slogctx "github.com/veqryn/slog-context"
)

// LocationOption is a functional option applied to a HX-Location request.
type LocationOption func(*HXLocationRequest)

// WithLocationPath option sets the path to which the request will be sent.
func WithLocationPath(path string) LocationOption {
	return func(hr *HXLocationRequest) {
		hr.Path = path
	}
}

// WithLocationTarget option sets the target to swap the response into.
func WithLocationTarget(target string) LocationOption {
	return func(hr *HXLocationRequest) {
		hr.Target = target
	}
}

// WithLocationSwap option sets how the response will be swapped in relative to the target.
func WithLocationSwap(swap string) LocationOption {
	return func(hr *HXLocationRequest) {
		hr.Swap = swap
	}
}

// WithLocationValues option sets values to submit with the request.
func WithLocationValues(values any) LocationOption {
	return func(hr *HXLocationRequest) {
		hr.Values = values
	}
}

// WithLocationHeaders option sets headers to submit with the request.
func WithLocationHeaders(headers map[string]string) LocationOption {
	return func(hr *HXLocationRequest) {
		hr.Headers = headers
	}
}

// WithLocationSource option sets the source element of the request.
func WithLocationSource(source string) LocationOption {
	return func(hr *HXLocationRequest) {
		hr.Source = source
	}
}

// WithLocationEvent option sets an event that “triggered” the request.
func WithLocationEvent(source string) LocationOption {
	return func(hr *HXLocationRequest) {
		hr.Source = source
	}
}

// WithLocationSelect option allows you to select the content you want swapped from a response.
func WithLocationSelect(sel string) LocationOption {
	return func(hr *HXLocationRequest) {
		hr.Select = sel
	}
}

// WithLocationPush option if set to 'false' or a path string to prevent or override the URL pushed to browser location
// history.
func WithLocationPush(push string) LocationOption {
	return func(hr *HXLocationRequest) {
		hr.Push = push
	}
}

// WithLocationReplace option sets a path string to replace the URL in the browser location history.
func WithLocationReplace(replace string) LocationOption {
	return func(hr *HXLocationRequest) {
		hr.Replace = replace
	}
}

// WithLocationHandler option sets a callback that will handle the response HTML.
func WithLocationHandler(handler string) LocationOption {
	return func(hr *HXLocationRequest) {
		hr.Handler = handler
	}
}

// LocationResponse creates and sets the HX-Location header on the response as per the given options.
func LocationResponse(options ...LocationOption) http.HandlerFunc {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		request := &HXLocationRequest{}
		for option := range slices.Values(options) {
			option(request)
		}
		requestJSON, err := json.Marshal(request)
		if err != nil {
			slogctx.Error(req.Context(), "Unable to marshal HX-Location request.",
				slog.Any("error", err))
			http.Error(res, "HX-Location Request failed", http.StatusInternalServerError)
			return
		}
		res.Header().Set(HeaderLocation, string(requestJSON))
	})
}
