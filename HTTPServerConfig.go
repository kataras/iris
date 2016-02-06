package gapi

var DefaultPort = 8080

type HTTPServerConfig struct {
	Port int    // default is 8080
	Host string `default:":"`
}

func NewHTTPServerConfig(host string, port int) *HTTPServerConfig {
	return &HTTPServerConfig{Port: port, Host: host}
}

func DefaultHttpConfig() *HTTPServerConfig {
	return &HTTPServerConfig{Port: DefaultPort, Host: ":"}
}
