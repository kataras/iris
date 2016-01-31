package server

import (
	. "github.com/kataras/gapi/router"
)

var DefaultPort = 8080

type HttpServerConfig struct {
	Port   int    // default is 8080
	Host   string `default:":"`
	Router *HttpRouter
}

func NewHttpServerConfig(host string, port int) *HttpServerConfig {
	return &HttpServerConfig{Port: port, Host: host}
}

func DefaultHttpConfig() *HttpServerConfig {
	return &HttpServerConfig{Port: DefaultPort, Host: ":"}
}
