package context

import "net/http"

// ClientErrorCodes holds the 4xx Client errors.
var (
	ClientErrorCodes = []int{
		http.StatusBadRequest,
		http.StatusUnauthorized,
		http.StatusPaymentRequired,
		http.StatusForbidden,
		http.StatusNotFound,
		http.StatusMethodNotAllowed,
		http.StatusNotAcceptable,
		http.StatusProxyAuthRequired,
		http.StatusRequestTimeout,
		http.StatusConflict,
		http.StatusGone,
		http.StatusLengthRequired,
		http.StatusPreconditionFailed,
		http.StatusRequestEntityTooLarge,
		http.StatusRequestURITooLong,
		http.StatusUnsupportedMediaType,
		http.StatusRequestedRangeNotSatisfiable,
		http.StatusExpectationFailed,
		http.StatusTeapot,
		http.StatusMisdirectedRequest,
		http.StatusUnprocessableEntity,
		http.StatusLocked,
		http.StatusFailedDependency,
		http.StatusTooEarly,
		http.StatusUpgradeRequired,
		http.StatusPreconditionRequired,
		http.StatusTooManyRequests,
		http.StatusRequestHeaderFieldsTooLarge,
		http.StatusUnavailableForLegalReasons,
		// Unofficial.
		StatusPageExpired,
		StatusBlockedByWindowsParentalControls,
		StatusInvalidToken,
		StatusTokenRequired,
	}
	// ServerErrorCodes holds the 5xx Server errors.
	ServerErrorCodes = []int{
		http.StatusInternalServerError,
		http.StatusNotImplemented,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout,
		http.StatusHTTPVersionNotSupported,
		http.StatusVariantAlsoNegotiates,
		http.StatusInsufficientStorage,
		http.StatusLoopDetected,
		http.StatusNotExtended,
		http.StatusNetworkAuthenticationRequired,
		// Unofficial.
		StatusBandwidthLimitExceeded,
		StatusInvalidSSLCertificate,
		StatusSiteOverloaded,
		StatusSiteFrozen,
		StatusNetworkReadTimeout,
	}

	// ClientAndServerErrorCodes is the static list of all client and server error codes.
	ClientAndServerErrorCodes = append(ClientErrorCodes, ServerErrorCodes...)
)

// Unofficial status error codes.
const (
	// 4xx
	StatusPageExpired                      = 419
	StatusBlockedByWindowsParentalControls = 450
	StatusInvalidToken                     = 498
	StatusTokenRequired                    = 499
	// 5xx
	StatusBandwidthLimitExceeded = 509
	StatusInvalidSSLCertificate  = 526
	StatusSiteOverloaded         = 529
	StatusSiteFrozen             = 530
	StatusNetworkReadTimeout     = 598
)

var unofficialStatusText = map[int]string{
	StatusPageExpired:                      "Page Expired",
	StatusBlockedByWindowsParentalControls: "Blocked by Windows Parental Controls",
	StatusInvalidToken:                     "Invalid Token",
	StatusTokenRequired:                    "Token Required",
	StatusBandwidthLimitExceeded:           "Bandwidth Limit Exceeded",
	StatusInvalidSSLCertificate:            "Invalid SSL Certificate",
	StatusSiteOverloaded:                   "Site is overloaded",
	StatusSiteFrozen:                       "Site is frozen",
	StatusNetworkReadTimeout:               "Network read timeout error",
}

// StatusText returns a text for the HTTP status code. It returns the empty
// string if the code is unknown.
func StatusText(code int) string {
	text := http.StatusText(code)
	if text == "" {
		text = unofficialStatusText[code]
	}

	return text
}

// StatusCodeNotSuccessful defines if a specific "statusCode" is not
// a valid status code for a successful response.
// By default if the status code is lower than 400 then it is not a failure one,
// otherwise it is considered as an error code.
//
// Read more at `iris/Configuration#DisableAutoFireStatusCode` and
// `iris/core/router/Party#OnAnyErrorCode` for relative information.
//
// Modify this variable when your Iris server or/and client
// not follows the RFC: https://www.w3.org/Protocols/rfc2616/rfc2616-sec10.html
var StatusCodeNotSuccessful = func(statusCode int) bool { return statusCode >= 400 }
