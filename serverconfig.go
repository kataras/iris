package iris

// DefaultPort is the default port number of the ServerConfig,  which a net listener is listening to
const DefaultPort = 8080

// ServerConfig contains just a port number and a host string, used on the server
type ServerConfig struct {
	Port int    // default is 8080
	Host string `default:":"`
}

// NewServerConfig creates and returns new server config with a host and port given as parameters
func NewServerConfig(host string, port int) *ServerConfig {
	return &ServerConfig{Port: port, Host: host}
}

// DefaultServerConfig creates and returns new server config with the default host(localhost) and port(8080)
func DefaultServerConfig() *ServerConfig {
	return &ServerConfig{Port: DefaultPort, Host: "0.0.0.0"}
}
