package versioning

import (
	"fmt"
	"strings"

	"github.com/blang/semver/v4"
)

var defaultPathPattern = "/v$1/$4" // default path pattern. "/v$VERSION_MAJOR/$REQUEST_PATH" pattern.

func parseExpressions(versions []Version) []*expression {
	exprs := make([]*expression, len(versions))
	for i := range versions {
		expr := parseExpression(versions[i])
		exprs[i] = expr
	}

	return exprs
}

// expression is a versioning expression.
type expression struct {
	expected semver.Range
	version  *Version
}

// parseExpression parses the version and target path pattern into an expression.
func parseExpression(version Version) *expression {
	validate, err := semver.ParseRange(version.Range)
	if err != nil {
		panic(fmt.Errorf("parseExpression: %s:%s: %w", version.Range, version.Path, err))
	}

	targetPath := version.Path
	if !strings.Contains(targetPath, "$") { // if no contains a pattern.
		if len(targetPath) == 1 { // contains only slash. Use the "/v$VERSION_MAJOR/$REQUEST_PATH" pattern.
			targetPath = defaultPathPattern
		} else {
			// it's a /v1 or /v2 or /v3 etc. Use the /$TARGET_PATH/$REQUEST_PATH pattern.
			targetPath = strings.TrimSuffix(targetPath, "/") + "/$4"
		}

		version.Path = targetPath
	}

	return &expression{
		expected: validate,
		version:  &version,
	}
}

func (expr *expression) match(parsedVersion semver.Version) bool {
	return expr.expected(parsedVersion)
}
