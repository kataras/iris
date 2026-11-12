package funcer

import (
	"maps"
	"slices"
	"sync/atomic"
)

// ListActionFunc is a function type that can be used to use a function from a list.
type ListActionFunc[T any] func(T, ...Option[T]) *List[T]

// nextFuncID is the process-wide registration id counter. Every function that
// enters a List receives a fresh id; clones keep the id of the registration
// they were cloned from, so a Func carried between lists stays the same
// registration. The id, not the pointer-derived name, is the identity of
// anonymous functions (see FuncInfo.Equal).
var nextFuncID atomic.Uint64

// newRegistration resolves fn through the program counter and stamps a fresh
// registration id on the result.
func newRegistration[T any](pc *ProgramCounter[T], fn T) *Func[T] {
	f := pc.Resolve(fn) // no need to clone, already returns a clone of the cache.
	f.ID = nextFuncID.Add(1)
	return f
}

// List can be used to manage a list of functions of any type.
type List[T any] struct {
	pc        *ProgramCounter[T]
	skipper   func(T) bool
	funcs     []*Func[T]
	listeners map[string][]eventCallback `json:"-"`
}

// NewList returns a new List of T type.
func NewList[T any](pc *ProgramCounter[T], skipper func(T) bool, funcs ...T) *List[T] {
	fs := make([]*Func[T], 0, len(funcs))
	for _, fn := range funcs {
		if skipper != nil && skipper(fn) {
			continue
		}
		fs = append(fs, newRegistration(pc, fn))
	}

	return &List[T]{
		pc:        pc,
		skipper:   skipper,
		funcs:     fs,
		listeners: make(map[string][]eventCallback),
	}
}

// Clone returns a deep copy of the list.
func (l *List[T]) Clone() *List[T] {
	fs := make([]*Func[T], 0, len(l.funcs))
	for _, f := range l.funcs {
		if f == nil || l.shouldSkip(f.Value) {
			continue
		}

		fs = append(fs, f.Clone())
	}

	return &List[T]{
		pc:        l.pc,
		funcs:     fs,
		listeners: maps.Clone(l.listeners),
	}
}

// applyOptions clones the fn source, stamps a fresh registration id and
// applies the given options (an option copying identity from another
// registration may deliberately override the fresh id).
func (l *List[T]) applyOptions(fn T, opts []Option[T]) *Func[T] {
	f := newRegistration(l.pc, fn)
	for _, opt := range opts {
		opt(f)
	}

	return f
}

func (l *List[T]) shouldSkip(fn T) bool {
	return l.skipper != nil && l.skipper(fn)
}

// Set replaces the list of functions with the given functions.
// No event is fired.
func (l *List[T]) Set(fns ...T) *List[T] {
	l.funcs = l.funcs[:0]
	for _, fn := range fns {
		if l.shouldSkip(fn) {
			continue
		}

		l.funcs = append(l.funcs, newRegistration(l.pc, fn))
	}

	return l
}

// SetIfEmpty replaces the list of functions with the given functions if the list is empty.
func (l *List[T]) SetIfEmpty(fns ...T) *List[T] {
	if l.Count() == 0 {
		l.Set(fns...)
	}

	return l
}

// Prepend adds a new function at the beginning of the list.
// Insert event is fired.
// Returns itself.
func (l *List[T]) Prepend(fn T, opts ...Option[T]) *List[T] {
	if l.shouldSkip(fn) {
		return l
	}

	f := l.applyOptions(fn, opts)
	l.funcs = slices.Insert(l.funcs, 0, f)
	l.publish(eventOpInsert, f.Name)

	return l
}

// PrependList prepends the functions of the given list to the beginning of the list.
func (l *List[T]) PrependList(list *List[T]) *List[T] {
	for _, v := range slices.Backward(list.funcs) {
		f := v.Clone()

		l.funcs = slices.Insert(l.funcs, 0, f) // prepend the *Func as it's, so if the name was changed, this change will apply to this list's func.
		l.publish(eventOpInsert, f.Name)
	}

	return l
}

// Append adds a new function at the end of the list.
// Insert event is fired.
// Returns itself.
func (l *List[T]) Append(fn T, opts ...Option[T]) *List[T] {
	if l.shouldSkip(fn) {
		return l
	}

	f := l.applyOptions(fn, opts)
	l.funcs = append(l.funcs, f)
	l.publish(eventOpInsert, f.Name)

	return l
}

// AppendSlice appends the given functions to the end of the list.
// Insert event is fired.
// Returns itself.
func (l *List[T]) AppendSlice(fns []T, opts ...Option[T]) *List[T] {
	for _, fn := range fns {
		if l.shouldSkip(fn) {
			continue
		}

		l.Append(fn, opts...)
	}

	return l
}

// AppendList appends the functions of the given list to the end of the list.
func (l *List[T]) AppendList(list *List[T]) *List[T] {
	for _, f := range list.funcs {
		if f == nil || l.shouldSkip(f.Value) {
			continue
		}

		l.funcs = append(l.funcs, f.Clone()) // append the *Func as it's, so if the name was changed, this change will apply to this list's func.
		l.publish(eventOpInsert, f.Name)
	}

	return l
}

// Replace finds a function by name and replaces it. It does nothing if the function does not exist.
// No event is fired.
// Returns itself.
func (l *List[T]) Replace(fn T) *List[T] {
	if l.shouldSkip(fn) {
		return l
	}

	f := l.applyOptions(fn, nil)
	for i, h := range l.funcs {
		if h.Equal(f) {
			l.funcs[i] = f
			return l
		}
	}

	return l
}

// Upsert inserts a new function if it doesn't exist, or replaces the existing one if it does.
// If not exist, insert event is fired.
// Returns itself.
func (l *List[T]) Upsert(fn T, opts ...Option[T]) *List[T] {
	if l.shouldSkip(fn) {
		return l
	}

	f := l.applyOptions(fn, opts)
	for i, h := range l.funcs {
		if h.Equal(f) {
			l.funcs[i] = f
			return l
		}
	}
	l.funcs = append(l.funcs, f)
	l.publish(eventOpInsert, f.Name)

	return l
}

// UpsertSlice inserts a new function if it doesn't exist, or replaces the existing one if it does.
// If not exist, insert event is fired.
// Returns itself.
func (l *List[T]) UpsertSlice(fns []T, opts ...Option[T]) *List[T] {
	for _, fn := range fns {
		if l.shouldSkip(fn) {
			continue
		}
		l.Upsert(fn, opts...)
	}

	return l
}

func (l *List[T]) compareFunc(name string) func(h *Func[T]) bool {
	return func(h *Func[T]) bool { return h.Name == name }
}

// Delete deletes a function from the list by name.
// Remove event is fired.
// Returns itself.
func (l *List[T]) Delete(name string) *List[T] {
	index := slices.IndexFunc(l.funcs, l.compareFunc(name))
	if index != -1 {
		l.funcs = slices.Delete(l.funcs, index, index+1)
		l.publish(eventOpRemove, name)
	}

	return l
}

// FuncDelete deletes a function from the list by the function itself.
func (l *List[T]) FuncDelete(fn T) *List[T] {
	return l.Delete(l.pc.Resolve(fn).Name)
}

// Lookup returns a function by name.
// If the function does not exist, the second return value is false.
func (l *List[T]) Lookup(name string) (*Func[T], bool) {
	index := slices.IndexFunc(l.funcs, l.compareFunc(name))
	if index == -1 {
		return nil, false
	}

	return l.funcs[index], true
}

// Exists checks if a function exists in the list by name.
func (l *List[T]) Exists(name string) bool {
	index := slices.IndexFunc(l.funcs, l.compareFunc(name))
	return index != -1
}

// FuncExists checks if a function exists in the list by the function itself.
func (l *List[T]) FuncExists(fn T) bool {
	return l.Exists(l.pc.Resolve(fn).Name)
}

// List returns all the functions in the list.
// Any modifications to the returned slice will affect the list.
func (l *List[T]) List() []*Func[T] {
	return l.funcs
}

// Values returns all the function values in the list.
func (l *List[T]) Values() []T {
	funcs := make([]T, 0, len(l.funcs))
	for _, f := range l.funcs {
		funcs = append(funcs, f.Value)
	}

	return funcs
}

// UniqueValues returns all the unique function values in the list.
//
// Uniqueness follows the identity rules of FuncInfo.Equal: trusted names
// deduplicate by name, anonymous functions deduplicate by registration id
// only — two separate registrations of code-identical anonymous closures are
// both kept, they are not the same function.
func (l *List[T]) UniqueValues() []T {
	uniqueValues := make([]T, 0)
	seenNames := make(map[string]struct{})
	seenIDs := make(map[uint64]struct{})

	for _, f := range l.funcs {
		if f.TrustedFuncName() {
			if _, found := seenNames[f.Name]; found {
				continue
			}
			seenNames[f.Name] = struct{}{}
		} else if f.ID != 0 {
			if _, found := seenIDs[f.ID]; found {
				continue
			}
			seenIDs[f.ID] = struct{}{}
		}

		uniqueValues = append(uniqueValues, f.Value)
	}

	return uniqueValues
}

// First returns the first function in the list.
func (l *List[T]) First() *Func[T] {
	if len(l.funcs) == 0 {
		return nil
	}

	return l.funcs[0]
}

// Last returns the last function in the list.
func (l *List[T]) Last() *Func[T] {
	if len(l.funcs) == 0 {
		return nil
	}

	return l.funcs[len(l.funcs)-1]
}

// FuncName returns the name of the function in the list.
func (l *List[T]) FuncName(fn T) string {
	name := l.pc.Resolve(fn).Name
	if !l.Exists(name) {
		return ""
	}

	return name
}

// Walk calls the given function for each function in the list.
// If the function returns false, the iteration stops.
func (l *List[T]) Walk(fn func(*Func[T]) bool) bool {
	for _, f := range l.funcs {
		ok := fn(f)
		if !ok {
			return false
		}
	}

	return true
}

// Count returns the number of functions in the list.
func (l *List[T]) Count() int {
	if l == nil {
		return 0
	}
	return len(l.funcs)
}

// Reset removes all functions and listeners from the list.
func (l *List[T]) Reset() {
	for _, h := range l.funcs {
		l.publish(eventOpRemove, h.Name)
	}

	l.funcs = l.funcs[:0]
	l.listeners = make(map[string][]eventCallback)
}

type eventOp int64

const (
	eventOpInsert eventOp = 1 << iota
	eventOpRemove
)

// Is reports whether the individual event is the specified operation.
func (op eventOp) Is(values ...eventOp) bool {
	for _, value := range values {
		if op&value == 0 {
			return false
		}
	}
	return true
}

type eventCallback func(eventOp) bool

func (l *List[T]) subscribe(onFuncName string, cb eventCallback) {
	if onFuncName == "" || cb == nil {
		return // Skip subscribing if the function name or callback is empty.
	}

	l.listeners[onFuncName] = append(l.listeners[onFuncName], cb)
}

// publish triggers the event and calls the subscribed handlers.
func (l *List[T]) publish(op eventOp, funcName string) {
	if funcName == "" {
		return // Skip publishing if the function name is empty.
	}

	if callbacks, ok := l.listeners[funcName]; ok {
		for i, cb := range callbacks {
			// fire the listener action.
			if cb(op) {
				// and remove it.
				l.listeners[funcName] = append(callbacks[:i], callbacks[i+1:]...)
			}
		}
	}
}

// SubscribePrependBefore adds a new function before the "beforeName" function.
// If the function with the same name is already registered, it will be added before it.
// If the function with the same name is not registered yet, it will be added before that when it will be registered.
func (l *List[T]) SubscribePrependBefore(fn T, beforeName string, opts ...Option[T]) {
	if l.shouldSkip(fn) {
		return
	}

	if l.Exists(beforeName) {
		l.PrependBefore(fn, beforeName, opts...)
	} else {
		l.subscribe(beforeName, func(op eventOp) bool {
			if !op.Is(eventOpInsert) {
				return false
			}

			return l.PrependBefore(fn, beforeName, opts...)
		})
	}
}

// SubscribeAppendAfter adds a new function after the "afterName" function.
// If the function with the same name is already registered, it will be added after it.
// If the function with the same name is not registered yet, it will be added after that when it will be registered.
func (l *List[T]) SubscribeAppendAfter(fn T, afterName string, opts ...Option[T]) {
	if l.shouldSkip(fn) {
		return
	}

	if l.Exists(afterName) {
		l.AppendAfter(fn, afterName, opts...)
	} else {
		l.subscribe(afterName, func(op eventOp) bool {
			if !op.Is(eventOpInsert) {
				return false
			}

			return l.AppendAfter(fn, afterName, opts...)
		})
	}
}

// PrependBefore adds a new function before another function identified by name.
func (l *List[T]) PrependBefore(newFunction T, beforeName string, opts ...Option[T]) bool {
	if l.shouldSkip(newFunction) {
		return false
	}

	if beforeName == "" {
		return false
	}

	newFunc := l.applyOptions(newFunction, opts)
	index := -1
	for i, h := range l.funcs {
		if h.Name == beforeName {
			index = i
			break
		}
	}
	if index != -1 {
		l.funcs = append(l.funcs[:index], append([]*Func[T]{newFunc}, l.funcs[index:]...)...)
		l.publish(eventOpInsert, newFunc.Name)
		return true
	}

	return false
}

// AppendAfter adds a new function after another function identified by name.
func (l *List[T]) AppendAfter(newFunction T, afterName string, opts ...Option[T]) bool {
	if l.shouldSkip(newFunction) {
		return false
	}

	if afterName == "" {
		return false
	}

	newFunc := l.applyOptions(newFunction, opts)
	index := -1
	for i, h := range l.funcs {
		if h.Name == afterName {
			index = i
			break
		}
	}
	if index != -1 {
		if index < len(l.funcs)-1 {
			l.funcs = append(l.funcs[:index+1], append([]*Func[T]{newFunc}, l.funcs[index+1:]...)...)
		} else {
			l.funcs = append(l.funcs, newFunc) // it's the last.
		}

		l.publish(eventOpInsert, newFunc.Name)
		return true
	}

	return false
}

// ConcatLists concatenates multiple lists into a single list.
func ConcatLists[T any](lists ...*List[T]) *List[T] {
	if len(lists) == 0 {
		return nil
	}

	funcs := make([]*Func[T], 0)
	for _, list := range lists {
		if list == nil || len(list.funcs) == 0 {
			continue
		}

		funcs = append(funcs, list.Clone().funcs...)
	}

	pc := lists[len(lists)-1].pc
	return &List[T]{
		pc:        pc,
		funcs:     funcs,
		listeners: make(map[string][]eventCallback),
	}
}
