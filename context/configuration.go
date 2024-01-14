package context

import (
	"time"

	"github.com/kataras/iris/v12/core/netutil"
)

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
	// GetSocketSharding returns the SocketSharding field.
	GetSocketSharding() bool
	// GetKeepAlive returns the KeepAlive field.
	GetKeepAlive() time.Duration
	// GetTimeout returns the Timeout field.
	GetTimeout() time.Duration
	// GetTimeoutMessage returns the TimeoutMessage field.
	GetTimeoutMessage() string
	// GetNonBlocking returns the NonBlocking field.
	GetNonBlocking() bool
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
	// GetEnableOptimizations returns the EnableDynamicHandler field.
	GetEnableDynamicHandler() bool
	// GetFireMethodNotAllowed returns the FireMethodNotAllowed field.
	GetFireMethodNotAllowed() bool
	// GetDisableAutoFireStatusCode returns the DisableAutoFireStatusCode field.
	GetDisableAutoFireStatusCode() bool
	// GetResetOnFireErrorCode returns the ResetOnFireErrorCode field.
	GetResetOnFireErrorCode() bool
	// GetURLParamSeparator returns URLParamSeparator field.
	GetURLParamSeparator() *string
	// GetEnableOptimizations returns the EnableOptimizations field.
	GetEnableOptimizations() bool
	// GetEnableProtoJSON returns the EnableProtoJSON field.
	GetEnableProtoJSON() bool
	// GetEnableEasyJSON returns the EnableEasyJSON field.
	GetEnableEasyJSON() bool

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

	// GetLocaleContextKey returns the LocaleContextKey field.
	GetLocaleContextKey() string
	// GetLanguageContextKey returns the LanguageContextKey field.
	GetLanguageContextKey() string
	// GetLanguageInputContextKey returns the LanguageInputContextKey field.
	GetLanguageInputContextKey() string
	// GetVersionContextKey returns the VersionContextKey field.
	GetVersionContextKey() string
	// GetVersionAliasesContextKey returns the VersionAliasesContextKey field.
	GetVersionAliasesContextKey() string

	// GetViewEngineContextKey returns the ViewEngineContextKey field.
	GetViewEngineContextKey() string
	// GetViewLayoutContextKey returns the ViewLayoutContextKey field.
	GetViewLayoutContextKey() string
	// GetViewDataContextKey returns the ViewDataContextKey field.
	GetViewDataContextKey() string
	// GetFallbackViewContextKey returns the FallbackViewContextKey field.
	GetFallbackViewContextKey() string

	// GetRemoteAddrHeaders returns RemoteAddrHeaders field.
	GetRemoteAddrHeaders() []string
	// GetRemoteAddrHeadersForce returns RemoteAddrHeadersForce field.
	GetRemoteAddrHeadersForce() bool
	// GetRemoteAddrPrivateSubnets returns the RemoteAddrPrivateSubnets field.
	GetRemoteAddrPrivateSubnets() []netutil.IPRange
	// GetSSLProxyHeaders returns the SSLProxyHeaders field.
	GetSSLProxyHeaders() map[string]string
	// GetHostProxyHeaders returns the HostProxyHeaders field.
	GetHostProxyHeaders() map[string]bool
	// GetOther returns the Other field.
	GetOther() map[string]interface{}
}
