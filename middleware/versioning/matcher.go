package versioning

import (
	"strconv"
	"strings"

	"github.com/blang/semver/v4"
)

type matcher struct {
	expressions []*expression
}

func newMatcher(expressions []*expression) *matcher {
	return &matcher{
		expressions: expressions,
	}
}

// Match matches the request version and path against the expressions.
// If the version is not expected, it returns false.
// Otherwise, it returns true and the target path.
// The target path may contain placeholders like "$1", "$2", "$3", "$4".
// - "$1" is replaced with the major version.
// - "$2" is replaced with the minor version.
// - "$3" is replaced with the patch version.
// - "$4" is replaced with the request path.
// The target path may be empty, in which case the request path is returned.
// The target path is a relative path, in which case it is appended to the request path.
// The third return value is true if the version is deprecated.
func (m *matcher) Match(requestVersion string, requestPath string) (string, semver.Version, bool, bool) {
	parsedVersion, err := parseVersion(requestVersion)
	if err != nil {
		return "", zeroVersion, false, false
	}

	requestPath = strings.TrimPrefix(requestPath, "/") // remove the 1st slash if any.
	for _, expr := range m.expressions {
		if !expr.match(parsedVersion) {
			continue
		}

		targetPath := expr.version.Path

		if placeholder := "$1"; strings.Contains(targetPath, placeholder) {
			majorString := strconv.FormatUint(parsedVersion.Major, 10)
			targetPath = strings.ReplaceAll(targetPath, placeholder, majorString)
		}

		if placeholder := "$2"; strings.Contains(targetPath, placeholder) {
			minorString := strconv.FormatUint(parsedVersion.Minor, 10)
			targetPath = strings.ReplaceAll(targetPath, placeholder, minorString)
		}

		if placeholder := "$3"; strings.Contains(targetPath, placeholder) {
			patchString := strconv.FormatUint(parsedVersion.Patch, 10)
			targetPath = strings.ReplaceAll(targetPath, placeholder, patchString)
		}

		if placeholder := "$4"; strings.Contains(targetPath, placeholder) {
			targetPath = strings.ReplaceAll(targetPath, placeholder, requestPath)
		}

		return targetPath, parsedVersion, expr.version.Deprecated, true
	}

	return "", parsedVersion, false, false
}

var zeroVersion semver.Version

func parseVersion(requestVersion string) (semver.Version, error) {
	parsedVersion, err := semver.Parse(requestVersion)
	if err != nil {
		if err.Error() == "No Major.Minor.Patch elements found" {
			parsedVersion, err = semver.ParseTolerant(requestVersion)
			if err != nil {
				return zeroVersion, err
			}
		} else {
			return zeroVersion, err
		}
	}

	return parsedVersion, nil
}
