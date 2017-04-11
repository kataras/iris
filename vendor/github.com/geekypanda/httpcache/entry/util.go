package entry

import (
	"regexp"
	"strconv"
)

var maxAgeExp = regexp.MustCompile(`maxage=(\d+)`)

// ParseMaxAge parses the max age from the receiver parameter, "cache-control" header
// returns seconds as int64
// if header not found or parse failed then it returns -1
func ParseMaxAge(header string) int64 {
	if header == "" {
		return -1
	}
	m := maxAgeExp.FindStringSubmatch(header)
	if len(m) == 2 {
		if v, err := strconv.Atoi(m[1]); err == nil {
			return int64(v)
		}
	}
	return -1
}
