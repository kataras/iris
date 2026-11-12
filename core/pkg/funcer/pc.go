package funcer

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"

	"github.com/kataras/iris/v14/core/pkg/reflex"
)

var (
	// rootPackageName is the Iris Go module package name.
	rootPackageName = strings.TrimSuffix(reflect.TypeFor[NameResolver]().PkgPath(), "/core/pkg/funcer")

	// workingDir is the (initial) current directory.
	workingDir, _ = os.Getwd()
)

// RelFilename returns the relative filename of a file.
func RelFilename(filename string) string {
	if strings.HasPrefix(filename, "..") {
		return filename
	}

	if relFile, err := filepath.Rel(workingDir, filename); err == nil {
		if !strings.HasPrefix(relFile, "..") {
			// Only if it's relative to this path, not parent.
			filename = "./" + relFile
		}
	}

	return filename
}

// ProgramCounter is a type that can be used to resolve the name and location of a function.
//
// The cache below is keyed by the function's code pointer and is DISPLAY-ONLY:
// since Go 1.27 two distinct code-identical function literals may share a code
// pointer, so entries may be shared between genuinely different functions.
// Identity never flows from this cache — it comes from the registration id
// assigned when a function enters a List (see FuncInfo.Equal).
type ProgramCounter[T any] struct {
	resolver *NameResolver

	cache map[uintptr]*FuncInfo
	mu    sync.RWMutex
}

// NewProgramCounter returns a new ProgramCounter.
func NewProgramCounter[T any](resolver *NameResolver) *ProgramCounter[T] {
	return &ProgramCounter[T]{resolver: resolver, cache: make(map[uintptr]*FuncInfo)}
}

// Resolve returns the custom name and the location of a function.
func (pc *ProgramCounter[T]) Resolve(h T) *Func[T] {
	info := pc.resolveInfo(h)

	return &Func[T]{ // always return a new pointer to Func, so any modifications do not affect the information cache.
		Value:    h,
		FuncInfo: info.Clone(),
	}
}

// GetName returns the custom name of a function.
// It caches the result for future calls.
func (pc *ProgramCounter[T]) GetName(h any) string {
	return pc.resolveInfo(h).Name
}

func (pc *ProgramCounter[T]) resolveInfo(h any) *FuncInfo {
	value := reflex.ValueOf(h)
	if value.IsNil() {
		panic("funcer: Resolve(nil) is not allowed")
	}
	if value.Kind() != reflect.Func {
		panic(fmt.Sprintf("funcer: Resolve(non-func) is not allowed: got: %s", value.Kind().String()))
	}
	ptr := value.Pointer()

	pc.mu.RLock()
	info := pc.cache[ptr]
	pc.mu.RUnlock()

	if info == nil {
		pc.mu.Lock()
		// Double-check under the write lock: another goroutine may have populated
		// the entry between releasing the read lock and acquiring the write lock,
		// which avoids resolving (and storing) the same ptr more than once.
		if info = pc.cache[ptr]; info == nil {
			info = NewFuncInfo(pc.resolver, ptr)
			pc.cache[ptr] = info
		}
		pc.mu.Unlock()
	}

	return info
}
