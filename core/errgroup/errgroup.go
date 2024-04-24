package errgroup

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

// Check reports whether the "err" is not nil.
// If it is a group then it returns true if that or its children contains any error.
func Check(err error) error {
	if isNotNil(err) {
		return err
	}

	return nil
}

// Walk loops through each of the errors of "err".
// If "err" is *Group then it fires the "visitor" for each of its errors, including children.
// if "err" is *Error then it fires the "visitor" with its type and wrapped error.
// Otherwise it fires the "visitor" once with typ of nil and err as "err".
func Walk(err error, visitor func(typ interface{}, err error)) error {
	if err == nil {
		return nil
	}

	if group, ok := err.(*Group); ok {
		list := group.getAllErrors()
		for _, entry := range list {
			if e, ok := entry.(*Error); ok {
				visitor(e.Type, e.Err) // e.Unwrap() <-no.
			} else {
				visitor(nil, err)
			}
		}
	} else if e, ok := err.(*Error); ok {
		visitor(e.Type, e.Err)
	} else {
		visitor(nil, err)
	}

	return err
}

/*
func Errors(err error, conv bool) []error {
	if err == nil {
		return nil
	}

	if group, ok := err.(*Group); ok {
		list := group.getAllErrors()
		if conv {
			for i, entry := range list {
				if _, ok := entry.(*Error); !ok {
					list[i] = &Error{Err: entry, Type: group.Type}
				}
			}
		}

		return list
	}

	return []error{err}
}

func Type(err error) interface{} {
	if err == nil {
		return nil
	}

	if e, ok := err.(*Error); ok && e.Err != nil {
		return e.Type
	}

	return nil
}

func Fill(parent *Group, errors []*Error) {
	for _, err := range errors {
		if err.Type == parent.Type {
			parent.Add(err)
			continue
		}

		parent.Group(err.Type).Err(err)
	}
	return
}
*/

// Error implements the error interface.
// It is a special error type which keep the "Type" of the
// Group that it's created through Group's `Err` and `Errf` methods.
type Error struct {
	Err  error       `json:"error" xml:"Error" yaml:"Error" toml:"Error" sql:"error"`
	Type interface{} `json:"type" xml:"Type" yaml:"Type" toml:"Type" sql:"type"`
}

// Error returns the error message of the "Err".
func (e *Error) Error() string {
	return e.Err.Error()
}

// Unwrap calls and returns the result of the "Err" Unwrap method or nil.
func (e *Error) Unwrap() error {
	return errors.Unwrap(e.Err)
}

// Is reports whether the "err" is an *Error.
func (e *Error) Is(err error) bool {
	if err == nil {
		return false
	}

	ok := errors.Is(e.Err, err)
	if !ok {
		te, ok := err.(*Error)
		if !ok {
			return false
		}

		return errors.Is(e.Err, te.Err)
	}

	return ok
}

// As reports whether the "target" can be used as &Error{target.Type: ?}.
func (e *Error) As(target interface{}) bool {
	if target == nil {
		return target == e
	}

	ok := errors.As(e.Err, target)
	if !ok {
		te, ok := target.(*Error)
		if !ok {
			return false
		}

		if te.Type != nil {
			if te.Type != e.Type {
				return false
			}
		}

		return errors.As(te.Err, &e)
	}

	return ok
}

// Group is an error container of a specific Type and can have child containers per type too.
type Group struct {
	parent *Group
	// a list of children groups, used to get or create new group through Group method.
	children map[interface{}]*Group
	depth    int

	Type   interface{}
	Errors []error // []*Error

	// if true then this Group's Error method will return the messages of the errors made by this Group's Group method.
	// Defaults to true.
	IncludeChildren bool // it clones.
	// IncludeTypeText bool
	index int // group index.
}

// New returns a new empty Group.
func New(typ interface{}) *Group {
	return &Group{
		Type:            typ,
		IncludeChildren: true,
	}
}

const delim = "\n"

func (g *Group) Error() (s string) {
	if len(g.Errors) > 0 {
		msgs := make([]string, len(g.Errors))
		for i, err := range g.Errors {
			msgs[i] = err.Error()
		}

		s = strings.Join(msgs, delim)
	}

	if g.IncludeChildren && len(g.children) > 0 {
		// return with order of definition.
		groups := g.getAllChildren()
		sortGroups(groups)

		for _, ge := range groups {
			for _, childErr := range ge.Errors {
				s += childErr.Error() + delim
			}
		}

		if s != "" {
			return s[:len(s)-1]
		}
	}

	return
}

func (g *Group) getAllErrors() []error {
	list := g.Errors

	if len(g.children) > 0 {
		// return with order of definition.
		groups := g.getAllChildren()
		sortGroups(groups)

		for _, ge := range groups {
			list = append(list, ge.Errors...)
		}
	}

	return list
}

func (g *Group) getAllChildren() []*Group {
	if len(g.children) == 0 {
		return nil
	}

	var groups []*Group
	for _, child := range g.children {
		groups = append(groups, append([]*Group{child}, child.getAllChildren()...)...)
	}

	return groups
}

// Unwrap implements the dynamic std errors interface and it returns the parent Group.
func (g *Group) Unwrap() error {
	if g == nil {
		return nil
	}

	return g.parent
}

// Group creates a new group of "typ" type, if does not exist, and returns it.
func (g *Group) Group(typ interface{}) *Group {
	if g.children == nil {
		g.children = make(map[interface{}]*Group)
	} else {
		for _, child := range g.children {
			if child.Type == typ {
				return child
			}
		}
	}

	child := &Group{
		Type:            typ,
		parent:          g,
		depth:           g.depth + 1,
		IncludeChildren: g.IncludeChildren,
		index:           g.index + 1 + len(g.children),
	}

	g.children[typ] = child

	return child
}

// Add adds an error to the group.
func (g *Group) Add(err error) {
	if err == nil {
		return
	}

	g.Errors = append(g.Errors, err)
}

// Addf adds an error to the group like `fmt.Errorf` and returns it.
func (g *Group) Addf(format string, args ...interface{}) error {
	err := fmt.Errorf(format, args...)
	g.Add(err)
	return err
}

// Err adds an error to the group, it transforms it to an Error type if necessary and returns it.
func (g *Group) Err(err error) error {
	if err == nil {
		return nil
	}

	e, ok := err.(*Error)
	if !ok {
		if ge, ok := err.(*Group); ok {
			if g.children == nil {
				g.children = make(map[interface{}]*Group)
			}

			g.children[ge.Type] = ge
			return ge
		}

		e = &Error{err, 0}
	}
	e.Type = g.Type

	g.Add(e)
	return e
}

// Errf adds an error like `fmt.Errorf` and returns it.
func (g *Group) Errf(format string, args ...interface{}) error {
	return g.Err(fmt.Errorf(format, args...))
}

func sortGroups(groups []*Group) {
	sort.Slice(groups, func(i, j int) bool {
		return groups[i].index < groups[j].index
	})
}

func isNotNil(err error) bool {
	if g, ok := err.(*Group); ok {
		if len(g.Errors) > 0 {
			return true
		}

		for _, child := range g.children {
			if isNotNil(child) {
				return true
			}
		}

		return false
	}

	return err != nil
}
