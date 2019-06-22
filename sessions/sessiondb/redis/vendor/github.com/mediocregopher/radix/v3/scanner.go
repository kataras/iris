package radix

import (
	"bufio"
	"errors"
	"strconv"
	"strings"

	"github.com/mediocregopher/radix/v3/resp/resp2"
)

// Scanner is used to iterate through the results of a SCAN call (or HSCAN,
// SSCAN, etc...)
//
// Once created, repeatedly call Next() on it to fill the passed in string
// pointer with the next result. Next will return false if there's no more
// results to retrieve or if an error occurred, at which point Close should be
// called to retrieve any error.
type Scanner interface {
	Next(*string) bool
	Close() error
}

// ScanOpts are various parameters which can be passed into ScanWithOpts. Some
// fields are required depending on which type of scan is being done.
type ScanOpts struct {
	// The scan command to do, e.g. "SCAN", "HSCAN", etc...
	Command string

	// The key to perform the scan on. Only necessary when Command isn't "SCAN"
	Key string

	// An optional pattern to filter returned keys by
	Pattern string

	// An optional count hint to send to redis to indicate number of keys to
	// return per call. This does not affect the actual results of the scan
	// command, but it may be useful for optimizing certain datasets
	Count int
}

func (o ScanOpts) cmd(rcv interface{}, cursor string) CmdAction {
	cmdStr := strings.ToUpper(o.Command)
	args := make([]string, 0, 6)
	if cmdStr != "SCAN" {
		args = append(args, o.Key)
	}

	args = append(args, cursor)
	if o.Pattern != "" {
		args = append(args, "MATCH", o.Pattern)
	}
	if o.Count > 0 {
		args = append(args, "COUNT", strconv.Itoa(o.Count))
	}

	return Cmd(rcv, cmdStr, args...)
}

// ScanAllKeys is a shortcut ScanOpts which can be used to scan all keys
var ScanAllKeys = ScanOpts{
	Command: "SCAN",
}

type scanner struct {
	Client
	ScanOpts
	res    scanResult
	resIdx int
	err    error
}

// NewScanner creates a new Scanner instance which will iterate over the redis
// instance's Client using the ScanOpts.
//
// NOTE if Client is a *Cluster this will not work correctly, use the NewScanner
// method on Cluster instead.
func NewScanner(c Client, o ScanOpts) Scanner {
	return &scanner{
		Client:   c,
		ScanOpts: o,
		res: scanResult{
			cur: "0",
		},
	}
}

func (s *scanner) Next(res *string) bool {
	for {
		if s.err != nil {
			return false
		}

		for s.resIdx < len(s.res.keys) {
			*res = s.res.keys[s.resIdx]
			s.resIdx++
			if *res != "" {
				return true
			}
		}

		if s.res.cur == "0" && s.res.keys != nil {
			return false
		}

		s.err = s.Client.Do(s.cmd(&s.res, s.res.cur))
		s.resIdx = 0
	}
}

func (s *scanner) Close() error {
	return s.err
}

type scanResult struct {
	cur  string
	keys []string
}

func (s *scanResult) UnmarshalRESP(br *bufio.Reader) error {
	var ah resp2.ArrayHeader
	if err := ah.UnmarshalRESP(br); err != nil {
		return err
	} else if ah.N != 2 {
		return errors.New("not enough parts returned")
	}

	var c resp2.BulkString
	if err := c.UnmarshalRESP(br); err != nil {
		return err
	}

	s.cur = c.S
	s.keys = s.keys[:0]

	return (resp2.Any{I: &s.keys}).UnmarshalRESP(br)
}
