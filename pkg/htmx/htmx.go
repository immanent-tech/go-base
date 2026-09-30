// Copyright 2025 Joshua Rich <joshua.rich@gmail.com>.
// SPDX-License-Identifier: 	AGPL-3.0-or-later

package htmx

import "net/http"

// ResponseHandling configures how htmx handles different HTTP response codes. When htmx receives a response it will
// iterate in order over the htmx.config.responseHandling array and test if the code property of a given object, when
// treated as a Regular Expression, matches the current response. If an entry does match the current response code, it
// will be used to determine if and how the response will be processed.
//
// https://htmx.org/docs/#response-handling
type ResponseHandling struct {
	// Code is a String representing a regular expression that will be tested against response codes.
	Code string `json:"code"`
	// Swap is true if the response should be swapped into the DOM, false otherwise.
	Swap bool `json:"swap"`
	// Error is true if htmx should treat this response as an error.
	Error bool `json:"error,omitempty,omitzero"`
	// IgnoreTitle is true if htmx should ignore title tags in the response.
	IgnoreTitle bool `json:"ignoreTitle,omitempty,omitzero"`
	// Select is a CSS selector to use to select content from the response.
	Select string `json:"select,omitempty,omitzero"`
	// Target is a CSS selector specifying an alternative target for the response.
	Target string `json:"target,omitempty,omitzero"`
	// SwapOverride is an alternative swap mechanism for the response.
	SwapOverride string `json:"swapOverride,omitempty,omitzero"`
}

const (
	ModeCORS       Mode = "cors"
	ModeNoCORS     Mode = "no-cors"
	ModeSameOrigin Mode = "same-origin"
)

type Mode string

// Config defines the htmx config options.
//
// https://htmx.org/docs/#config
type Config struct {
	// Logall defaults to false, if set to true htmx will log all events to the console for debugging.
	LogAll *bool `json:"logAll,omitempty"`
	// Prefix defaults to "data-hx-", a secondary attribute prefix recognised alongside the always-active hx- prefix
	// (e.g. data-hx-get works by default). Set to "" to disable. Must be set via meta tag, setting this after page load
	// will not apply correctly.
	Prefix string `json:"prefix,omitzero"`
	// Transitions defaults to false, whether to use view transitions when swapping content (if browser supports it).
	Transitions *bool `json:"transitions,omitempty"`
	// History defaults to true, whether to enable history support. Set to "reload" to do a full page reload on history
	// navigation instead of an AJAX request.
	History *bool `json:"history,omitempty"`
	// Mode defaults to 'same-origin', the fetch mode for AJAX requests. Can be 'cors', 'no-cors', or 'same-origin'.
	Mode Mode `json:"mode,omitzero" validate:"omitempty,oneof=cors no-cors same-origin"`
	// DefaultSwap defaults to "innerMorph".
	DefaultSwap string `json:"defaultSwap,omitzero"`
	// IndicatorClass defaults to "htmx-indicator".
	IndicatorClass string `json:"indicatorClass,omitzero"`
	// RequestClass defaults to "htmx-request".
	RequestClass string `json:"requestClass,omitzero"`
	// IncludeIndicatorCSS defaults to true (determines if the indicator styles are loaded).
	IncludeIndicatorCSS *bool `json:"includeIndicatorCSS,omitempty"`
	// DefaultTimeout defaults to 60000 (60 seconds), the number of milliseconds a request can take before automatically
	// being terminated.
	DefaultTimeout int `json:"defaultTimeout" validate:"omitempty,gt=0"`
	// InlineScriptNonce defaults to unset, meaning that no nonce will be added to inline scripts.
	InlineScriptNonce string `json:"inlineScriptNonce,omitzero"`
	// Extensions defaults to '', a comma-separated list of extension names to load (e.g., 'preload,pending').
	Extensions string `json:"extensions,omitzero"`
	// MorphIgnore defaults to ["data-htmx-powered"], array of attribute name prefixes to preserve when morphing
	// elements.
	MorphIgnore string `json:"morphIgnore,omitzero"`
	// MorphScanLimit limits the number of nodes scanned during morphing.
	MorphScanLimit int `json:"morphScanLimit,omitzero" validate:"omitempty,gt=0"`
	// MorphSkip defaults to '[hx-morph-skip]', CSS selector for elements to completely skip during morphing (they stay
	// frozen).
	MorphSkip string `json:"morphSkip,omitzero"`
	// MorphSkipChildren defaults to '[hx-morph-skip-children]', CSS selector for elements whose attributes update but
	// children are preserved during morphing.
	MorphSkipChildren string `json:"morphSkipChildren,omitzero"`
	// NoSwap defaults to [204, 304], array of HTTP status codes that should not trigger a swap.
	NoSwap []int `json:"noSwap,omitzero"`
	// AllowEmptySwapAfterOOB defaults to false, whether the main swap still runs when a response contained only
	// out-of-band elements.
	AllowEmptySwapAfterOOB *bool `json:"allowEmptySwapAfterOOB,omitempty"`
	// ImplicitInheritance defaults to false, if set to true attributes will be inherited from parent elements
	// automatically without requiring the :inherited modifier.
	ImplicitInheritance *bool `json:"implicitInheritance,omitempty"`
	// DefaultFocusScroll defaults to false, whether to scroll focused elements into view after swap.
	DefaultFocusScroll *bool `json:"defaultFocusScroll,omitempty"`
	// DefaultSettleDelay defaults to 1 (ms), delay between swap and settle phases.
	DefaultSettleDelay int `json:"defaultSettleDelay,omitzero" validate:"omitempty,gt=0"`
	// MetaCharacter defaults to undefined, allows you to use a custom character instead of : for attribute modifiers
	// (e.g., - to use hx-get-inherited instead of hx-get:inherited).
	MetaCharacter string `json:"metaCharacter,omitzero"`
}

// HXLocationRequest defines the value of the HX-Location header.
//
// https://htmx.org/headers/hx-location/
type HXLocationRequest struct {
	// The URL path.
	Path string `json:"path"`
	//  The source element of the request.
	Source string `json:"source,omitzero"`
	// An event that “triggered” the request.
	Event string `json:"event,omitzero"`
	// A JS callback that will handle the response HTML.
	Handler string `json:"handler,omitzero"`
	// The target to swap the response into.
	Target string `json:"target,omitzero"`
	// How the response will be swapped in relative to the target.
	Swap string `json:"swap,omitzero"`
	// Values to submit with the request.
	Values any `json:"values,omitzero"`
	// Headers to submit with the request.
	Headers map[string]string `json:"headers,omitzero"`
	// Allows you to select the content you want swapped from a response.
	Select string `json:"select,omitzero"`
	// Set to 'false' or a path string to prevent or override the URL pushed to browser location history
	Push string `json:"push,omitzero"`
	// A path string to replace the URL in the browser location history
	Replace string `json:"replace,omitzero"`
}

// The following functions are copied from github.com/angelofallars/htmx-go.

// IsHTMX returns true if the given request
// was made by HTMX.
//
// This can be used to add special logic for HTMX requests.
//
// Checks if header 'HX-Request' is 'true'.
func IsHTMX(r *http.Request) bool {
	return r.Header.Get(HeaderRequest) == "true"
}

// IsBoosted returns true if the given request
// was made via an element using 'hx-boost'.
//
// This can be used to add special logic for boosted requests.
//
// Checks if header 'HX-Boosted' is 'true'.
//
// For more info, see https://htmx.org/attributes/hx-boost/
func IsBoosted(r *http.Request) bool {
	return r.Header.Get(HeaderBoosted) == "true"
}

// IsHistoryRestoreRequest returns true if the given request
// is for history restoration after a miss in the local history cache.
//
// Checks if header 'HX-History-Restore-Request' is 'true'.
func IsHistoryRestoreRequest(r *http.Request) bool {
	return r.Header.Get(HeaderHistoryRestoreRequest) == "true"
}

// GetCurrentURL returns the current URL that HTMX made this request from.
//
// Returns false if header 'HX-Current-URL' does not exist.
func GetCurrentURL(r *http.Request) (string, bool) {
	if _, ok := r.Header[http.CanonicalHeaderKey(HeaderCurrentURL)]; !ok {
		return "", false
	}
	return r.Header.Get(HeaderCurrentURL), true
}

// GetTarget returns the ID of the target element if it exists from a given request.
//
// Returns false if header 'HX-Target' does not exist.
//
// For more info, see https://htmx.org/attributes/hx-target/
func GetTarget(r *http.Request) (string, bool) {
	if _, ok := r.Header[http.CanonicalHeaderKey(HeaderTarget)]; !ok {
		return "", false
	}
	return r.Header.Get(HeaderTarget), true
}
