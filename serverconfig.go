package gapi

var DefaultPort = 8080

type ServerConfig struct {
	Port int    // default is 8080
	Host string `default:":"`
}

func NewServerConfig(host string, port int) *ServerConfig {
	return &ServerConfig{Port: port, Host: host}
}

func DefaultServerConfig() *ServerConfig {
	return &ServerConfig{Port: DefaultPort, Host: "0.0.0.0"}
}
