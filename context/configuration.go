package context

import "github.com/kataras/iris/v12/core/netutil"

// ConfigurationReadOnly can be implemented
// by Configuration, it's being used inside the Context.
// All methods that it contains should be "safe" to be called by the context
// at "serve time". A configuration field may be missing when it's not
// safe or its useless to be called from a request handler.
type ConfigurationReadOnly interface {
	// GetVHost returns the non-exported vhost config field.
	GetVHost() string
	// GetLogLevel returns the LogLevel field.
	GetLogLevel() string
	// GetDisablePathCorrection returns the DisablePathCorrection field
	GetDisablePathCorrection() bool
	// GetDisablePathCorrectionRedirection returns the DisablePathCorrectionRedirection field.
	GetDisablePathCorrectionRedirection() bool
	// GetEnablePathIntelligence returns the EnablePathIntelligence field.
	GetEnablePathIntelligence() bool
	// GetEnablePathEscape returns the EnablePathEscape field.
	GetEnablePathEscape() bool
	// GetForceLowercaseRouting returns the ForceLowercaseRouting field.
	GetForceLowercaseRouting() bool
	// GetFireMethodNotAllowed returns the FireMethodNotAllowed field.
	GetFireMethodNotAllowed() bool
	// GetDisableAutoFireStatusCode returns the DisableAutoFireStatusCode field.
	GetDisableAutoFireStatusCode() bool
	// ResetOnFireErrorCode retruns the ResetOnFireErrorCode field.
	GetResetOnFireErrorCode() bool

	// GetEnableOptimizations returns the EnableOptimizations field.
	GetEnableOptimizations() bool
	// GetDisableBodyConsumptionOnUnmarshal returns the DisableBodyConsumptionOnUnmarshal field.
	GetDisableBodyConsumptionOnUnmarshal() bool
	// GetFireEmptyFormError returns the FireEmptyFormError field.
	GetFireEmptyFormError() bool

	// GetTimeFormat returns the TimeFormat field.
	GetTimeFormat() string
	// GetCharset returns the Charset field.
	GetCharset() string
	// GetPostMaxMemory returns the PostMaxMemory field.
	GetPostMaxMemory() int64

	// GetTranslateLanguageContextKey returns the LocaleContextKey field.
	GetLocaleContextKey() string
	// GetLanguageContextKey returns the LanguageContextKey field.
	GetLanguageContextKey() string
	// GetVersionContextKey returns the VersionContextKey field.
	GetVersionContextKey() string
	// GetViewLayoutContextKey returns the ViewLayoutContextKey field.
	GetViewLayoutContextKey() string
	// GetViewDataContextKey returns the ViewDataContextKey field.
	GetViewDataContextKey() string

	// GetRemoteAddrHeaders returns RemoteAddrHeaders field.
	GetRemoteAddrHeaders() map[string]bool
	// GetRemoteAddrPrivateSubnets returns the RemoteAddrPrivateSubnets field.
	GetRemoteAddrPrivateSubnets() []netutil.IPRange
	// GetSSLProxyHeaders returns the SSLProxyHeaders field.
	GetSSLProxyHeaders() map[string]string

	// GetOther returns the Other field.
	GetOther() map[string]interface{}
}
