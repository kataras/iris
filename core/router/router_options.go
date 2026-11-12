package router

// RouterOptions is the configuration for the Router.
//
// The json keys are snake_case, matching the ini ones and context.ContextOptions;
// the yaml and toml keys are the Go field names.
//
// Note: root-domain to "www" redirection is not a field here; the "rewrite"
// middleware's PrimarySubdomain option does that and more.
type RouterOptions struct {
	// Host is the server's host, including the port (not required if 80 or 443).
	// If empty then this field is set to Server's options Host by default.
	Host string `ini:"host" json:"host,omitempty" yaml:"Host" toml:"Host"`
	// DisablePathCorrection disables the correcting
	// and redirecting or executing directly the handler of
	// the requested path to the registered path
	// for example, if /home/ path is requested but no handler for this Route found,
	// then the Router checks if /home handler exists, if yes,
	// (permanent)redirects the client to the correct path /home.
	//
	// See `DisablePathCorrectionRedirection` to enable direct handler execution instead of redirection.
	//
	// Defaults to false.
	DisablePathCorrection bool `ini:"disable_path_correction" json:"disable_path_correction,omitzero" yaml:"DisablePathCorrection" toml:"DisablePathCorrection"`
	// DisablePathCorrectionRedirection works whenever configuration.DisablePathCorrection is set to false
	// and if DisablePathCorrectionRedirection set to true then it will fire the handler of the matching route without
	// the trailing slash ("/") instead of send a redirection status.
	//
	// Defaults to false.
	DisablePathCorrectionRedirection bool `ini:"disable_path_correction_redirection" json:"disable_path_correction_redirection,omitzero" yaml:"DisablePathCorrectionRedirection" toml:"DisablePathCorrectionRedirection"`
	// EnablePathIntelligence if set to true,
	// the router will redirect HTTP "GET" not found pages to the most closest one path(if any). For example
	// you register a route at "/contact" path -
	// a client tries to reach it by "/cont", the path will be automatic fixed
	// and the client will be redirected to the "/contact" path
	// instead of getting a 404 not found response back.
	//
	// Defaults to false.
	EnablePathIntelligence bool `ini:"enable_path_intelligence" json:"enable_path_intelligence,omitzero" yaml:"EnablePathIntelligence" toml:"EnablePathIntelligence"`
	// EnablePathEscape enables path and parameter escaping. Disable it if parameters with slashes are expected.
	// Example: A request to http://localhost:8080/details/Project%2FDelta will retain the encoded slash.
	// Defaults to false.
	EnablePathEscape bool `ini:"enable_path_escape" json:"enable_path_escape,omitzero" yaml:"EnablePathEscape" toml:"EnablePathEscape"`
	// ForceLowercaseRouting if enabled, converts all registered routes paths to lowercase
	// and it does lowercase the request path too for matching.
	//
	// Defaults to false.
	ForceLowercaseRouting bool `ini:"force_lowercase_routing" json:"force_lowercase_routing,omitzero" yaml:"ForceLowercaseRouting" toml:"ForceLowercaseRouting"`
	// AllowQuerySemicolons if true adds a router wrapper that serves requests by converting any
	// unescaped semicolons(;) in the URL query to ampersands(&).
	//
	// This restores the pre-Go 1.17 behavior of splitting query parameters on both semicolons and ampersands.
	// See golang.org/issue/25192 and https://github.com/kataras/iris/issues/1875.
	// Note that this behavior doesn't match that of many proxies,
	// and the mismatch can lead to security issues.
	AllowQuerySemicolons bool `ini:"allow_query_semicolons" json:"allow_query_semicolons,omitzero" yaml:"AllowQuerySemicolons,omitempty" toml:"AllowQuerySemicolons,omitempty"`

	// EnableDynamicHandler enables the dynamic request handler.
	// It gives the router the feature to add routes while in serve-time,
	// when `BuildRouter` is called again.
	// If this setting is set to true, the request handler will use a mutex for data(trie routing) protection,
	// hence the performance cost.
	//
	// Defaults to false.
	EnableDynamicHandler bool `ini:"enable_dynamic_handler" json:"enable_dynamic_handler,omitzero" yaml:"EnableDynamicHandler" toml:"EnableDynamicHandler"`
	// FireMethodNotAllowed if it's true router checks for StatusMethodNotAllowed(405) and
	//  fires the 405 error instead of 404
	// Defaults to false.
	FireMethodNotAllowed bool `ini:"fire_method_not_allowed" json:"fire_method_not_allowed,omitzero" yaml:"FireMethodNotAllowed" toml:"FireMethodNotAllowed"`
}
