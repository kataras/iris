package funcer

import (
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/kataras/iris/v14/core/pkg/buildinfo"
	"github.com/kataras/iris/v14/core/pkg/reflex"
)

// NameResolver is a type that can be used to register custom names for functions.
type NameResolver struct {
	names map[*NameExpr]string
	mu    sync.RWMutex
}

// NewNameResolver returns a new NameResolver.
func NewNameResolver() *NameResolver {
	return &NameResolver{
		names: make(map[*NameExpr]string),
	}
}

// The regular expression to match the module name and the version of the module.
var trimFileModuleNameRegex = regexp.MustCompile(`^[\w.]+/(kataras|iris-contrib)/|/v\d+|\.\*`)

// SetName registers a custom name for a function.
// The "replacement" should be the custom, human-text of that function name.
//
// If the name starts with "iris" then it replaces that string with the
// full Iris module package name.
//
// The "original" pattern must be a valid regular expression (a plain
// function name is a valid regexp as long as it carries no metacharacters
// like "(*Type)"; escape those). An invalid pattern panics: every SetName
// call is an init-time constant, and a swallowed compile error used to
// silently degrade the pattern to an exact-literal match.
func (ns *NameResolver) SetName(original string, replacement string) {
	if after, ok := strings.CutPrefix(original, "iris"); ok {
		original = rootPackageName + after
	}

	regex, err := regexp.Compile(original)
	if err != nil {
		panic(fmt.Sprintf("funcer: SetName: invalid pattern %q for replacement %q: %v", original, replacement, err))
	}

	expr := &NameExpr{
		literal:    original,
		normalized: strings.ToLower(trimFileModuleNameRegex.ReplaceAllString(original, "")),
		regex:      regex,
	}

	ns.mu.Lock()
	ns.names[expr] = replacement
	ns.mu.Unlock()
}

// GetCustomName returns the custom name of a function or/and filename.
func (ns *NameResolver) GetCustomName(funcName, filename string) string {
	name, _ := ns.GetCustomNameMatched(funcName, filename)
	return name
}

// GetCustomNameMatched returns the custom name of a function or/and filename
// and reports whether a registered SetName mapping matched. A matched name is
// trusted as the function's identity (see FuncInfo.TrustedFuncName).
func (ns *NameResolver) GetCustomNameMatched(funcName, filename string) (string, bool) {
	filename = strings.ToLower(filename)
	matched := false

	ns.mu.RLock()
	for expr, newName := range ns.names {
		if expr.MatchString(funcName) || expr.MatchFilename(filename) {
			funcName = newName
			matched = true
			break
		}
	}
	ns.mu.RUnlock()

	return trimFuncName(funcName), matched
}

// GetName returns the custom name of a function.
func (ns *NameResolver) GetName(fn any) string {
	funcName, funcFile, _ := reflex.FuncPtrNameAndFileLine(reflex.ValueOf(fn).Pointer())
	return ns.GetCustomName(funcName, funcFile)
}

// GetNameAndFileAndLine returns the custom name of a function and the file and line of that function.
func (ns *NameResolver) GetNameAndFileAndLine(fn any) (name, file string, line int) {
	funcName, funcFile, funcLine := reflex.FuncPtrNameAndFileLine(reflex.ValueOf(fn).Pointer())
	return ns.GetCustomName(funcName, funcFile), funcFile, funcLine
}

func trimFuncName(name string) string {
	// trim the path for Iris' internal middlewares, e.g.
	if internalName, internalPrefix := rootPackageName, "iris"; strings.HasPrefix(name, internalName) {
		name = strings.Replace(name, internalName, internalPrefix, 1)
	} else if internalName, internalPrefix := "github.com/iris-contrib", "iris-contrib"; strings.HasPrefix(name, internalName) {
		name = strings.Replace(name, internalName, internalPrefix, 1)
	} else {
		// replace the module full name (e.g github.com/kataras/my-software) with the module's suffix name (e.g. my-software).
		moduleSuffixName := buildinfo.ModuleName
		if moduleSuffixName != "" {
			parts := strings.Split(buildinfo.ModuleName, "/")
			if len(parts) > 0 {
				moduleSuffixName = parts[len(parts)-1]
			}
			if internalName, internalPrefix := buildinfo.ModuleName, moduleSuffixName; strings.HasPrefix(name, internalName) {
				name = strings.Replace(name, internalName, internalPrefix, 1)
			}
		}
	}

	return name
}

// NameExpr regex or literal comparison through `MatchString`.
type NameExpr struct {
	regex      *regexp.Regexp
	literal    string
	normalized string
}

// MatchString reports whether "s" is equal to the "literal"
// or it matches the regex expression at "regex".
func (expr *NameExpr) MatchString(s string) bool {
	if expr.literal == s { // if matches as string, as it's.
		return true
	}

	if expr.regex != nil {
		return expr.regex.MatchString(s)
	}

	return false
}

// MatchFilename reports whether "filename" contains the normalized form of the
// expression, that is the "literal" lower-cased and without the kataras or
// iris-contrib module prefix, the "/vN" version suffix and the ".*" part.
// The given filename is expected to be already lower-cased, see GetCustomName.
func (expr *NameExpr) MatchFilename(filename string) bool {
	if filename == "" {
		return false
	}

	return strings.Contains(filename, expr.normalized)
}
