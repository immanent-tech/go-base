// Adapted from github.com/angelofallars/htmx-go

package htmx

const (
	// HeaderRequest is always “true” for HTMX requests.
	HeaderRequest = "HX-Request"
	// HeaderRequestType indicates which type of HTMX request this is, "full" or "partial".
	HeaderRequestType = "HX-Request-Type"
	// HeaderCurrentURL contains the current URL of the browser.
	HeaderCurrentURL = "HX-Current-URL"
	// HeaderSource request header contains the element that triggered the request. Format is tag#id like button#submit.
	HeaderSource = "HX-Source"
	// HeaderTarget is the id of the target element if it exists.
	HeaderTarget = "HX-Target"
	// HeaderBoosted is "true" if the request was made from an element using 'hx-boost'.
	HeaderBoosted = "HX-Boosted"
	// HeaderHistoryRestoreRequest is “true” if the request is for history restoration after a miss in the local history cache.
	HeaderHistoryRestoreRequest = "HX-History-Restore-Request"
)

// RequestHeaders contains all valid HTMX request headers.
//
// https://htmx.org/reference/#request_headers
var RequestHeaders = []string{
	HeaderRequest,
	HeaderRequestType,
	HeaderCurrentURL,
	HeaderSource,
	HeaderTarget,
	HeaderBoosted,
	HeaderHistoryRestoreRequest,
}

const (
	// HeaderTrigger triggers client-side events after the swap has completed.
	HeaderTrigger = "HX-Trigger"
	// HeaderLocation allows you to do a client-side redirect that does not do a full page reload.
	HeaderLocation = "HX-Location"
	// HeaderRedirect can be used to do a client-side redirect to a new location.
	HeaderRedirect = "HX-Redirect"
	// HeaderRefresh if set to “true” the client-side will do a full refresh of the page.
	HeaderRefresh = "HX-Refresh"
	// HeaderRetarget uses a CSS selector that updates the target of the content update to a different element on the
	// page.
	HeaderRetarget = "HX-Retarget"
	// HeaderReswap allows you to specify how the response will be swapped.
	HeaderReswap = "HX-Reswap"
	// HeaderReselect uses a CSS selector that allows you to choose which part of the response is used to be swapped in.
	// Overrides an existing hx-select on the triggering element.
	HeaderReselect = "HX-Reselect"
	// HeaderReplaceUrl replaces the current URL in the location bar.
	HeaderReplaceUrl = "HX-Replace-Url"
	// HeaderPushURL pushes a new url into the history stack.
	HeaderPushURL = "HX-Push-Url"
)

// ResponseHeaders contains all valid HTMX response headers.
//
// https://htmx.org/reference/#response_headers
var ResponseHeaders = []string{
	HeaderLocation,
	HeaderPushURL,
	HeaderRedirect,
	HeaderRefresh,
	HeaderReplaceUrl,
	HeaderReswap,
	HeaderRetarget,
	HeaderReselect,
	HeaderTrigger,
}

// StatusStopPolling is a HTTP status code that tells HTMX to stop polling from a server response.
//
// For more info, see https://htmx.org/docs/#load_polling
const StatusStopPolling int = 286
