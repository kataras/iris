package server

import "os"

// Config used inside server for listening
// all options are exported.
type Config struct {
	// ListenningAddr the addr that server listens to
	ListeningAddr string
	CertFile      string
	KeyFile       string
	// Mode this is for unix only
	Mode os.FileMode
}
