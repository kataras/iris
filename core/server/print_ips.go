package server

import (
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"strings"
)

// DefaultStartupLogger is the default logger which is used by the `printIPs` function.
var DefaultStartupLogger = os.Stdout

func printIPs(dest io.Writer, serverNetworkAddr, serverURL string) {
	if dest == nil {
		dest = DefaultStartupLogger
	}

	var listeningURIs = make([]string, 0, 1)

	if host, port, err := net.SplitHostPort(serverNetworkAddr); err == nil { // Improve for the issue #2175.
		if host == "" || host == "0.0.0.0" {
			if ifaces, err := net.Interfaces(); err == nil {
				var ips []string
				for _, i := range ifaces {
					addrs, err := i.Addrs()
					if err != nil {
						continue
					}
					for _, localAddr := range addrs {
						var ip net.IP
						switch v := localAddr.(type) {
						case *net.IPNet:
							ip = v.IP
						case *net.IPAddr:
							ip = v.IP
						}
						if ip != nil && ip.To4() != nil {
							if !ip.IsPrivate() {
								// let's don't print ips that are not accessible through browser.
								continue
							}
							ips = append(ips, ip.String())
						}
					}
				}

				for _, ip := range ips {
					scheme := "http"
					if port == "http" { // if :http is given on the options address.
						port = "80"
					} else if port == "443" || port == "https" { // if :https is given on the options address.
						port = "443"
						scheme = "https"
					}
					listeningURI := scheme + "://" + ResolveHostname(net.JoinHostPort(ip, port))

					listeningURI = "> Network:  " + listeningURI
					listeningURIs = append(listeningURIs, listeningURI)
				}
			}
		}
	}

	// "0.0.0.0" is the wildcard address, a browser cannot open it.
	// Print localhost in its place, as it is printed for ":8080". See #2175.
	if u, err := url.Parse(serverURL); err == nil && u.Hostname() == "0.0.0.0" {
		u.Host = "localhost" + strings.TrimPrefix(u.Host, "0.0.0.0")
		serverURL = u.String()
	}

	if _, host, _, err := SplitURL(serverURL); err == nil {
		if IsLoopback(host) {
			listeningURIs = append(listeningURIs, "> Local:    "+serverURL)
		} else {
			listeningURIs = append(listeningURIs, "> Public:   "+serverURL)
		}
	}

	if len(listeningURIs) > 1 {
		listeningURIs[0] = "\n" + listeningURIs[0]
	} else { // only Local or Public, no need for > Local|Public: .
		listeningURIs = []string{serverURL}
	}

	/*
		Now listening on:
			> Network:  http://192.168.1.109:8080
			> Network:  http://172.25.224.1:8080
			> Local:    http://localhost:8080
		OR

		Now listening on:
			> Network:  http://192.168.1.109:8080
			> Network:  http://172.25.224.1:8080
			> Public:   https://iris-go.com

		Otherwise:
			Now listening on: http://192.168.1.109:8080
	*/
	_, _ = fmt.Fprintf(dest, "Now listening on: %s\nApplication started. Press CTRL+C to shut down.\n",
		strings.Join(listeningURIs, "\n"))
}
