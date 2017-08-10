package netutil

import (
	"net"
	"net/http"
	"time"

	"github.com/kataras/golog"
)

// Client returns a new http.Client using
// the "timeout" for open connection and read-write operations.
func Client(timeout time.Duration) *http.Client {
	transport := http.Transport{
		Dial: func(network string, addr string) (net.Conn, error) {
			conn, err := net.DialTimeout(network, addr, timeout)
			if err != nil {
				golog.Debugf("%v", err)
				return nil, err
			}
			if err = conn.SetDeadline(time.Now().Add(timeout)); err != nil {
				golog.Debugf("%v", err)
			}
			return conn, err
		},
	}

	client := &http.Client{
		Transport: &transport,
	}

	return client
}
