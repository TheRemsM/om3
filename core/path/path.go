package path

import (
	"bytes"
	"encoding/json"
	"regexp"
	"strings"

	"github.com/danwakefield/fnmatch"

	"github.com/pkg/errors"
	"opensvc.com/opensvc/core/kind"
)

type (
	// T represents an opensvc object path-like identifier. Ex: ns1/svc/svc1
	T struct {
		// Name is the name part of the path
		Name string
		// Namespace is the namespace part of the path
		Namespace string
		// Kind is the kind part of the path
		Kind kind.T
	}

	// RelationPath is an object path or an instance path (path@node).
	Relation string
)

const (
	// Separator is the character separating a path's namespace, kind and name
	Separator = "/"

	hostnameRegexStringRFC952 = `^[a-zA-Z]([a-zA-Z0-9\-]+[\.]?)*[a-zA-Z0-9]$` // https://tools.ietf.org/html/rfc952
	fqdnRegexStringRFC1123    = `^([a-zA-Z0-9]{1}[a-zA-Z0-9_-]{0,62})(\.[a-zA-Z0-9_]{1}[a-zA-Z0-9_-]{0,62})*?(\.[a-zA-Z]{1}[a-zA-Z0-9]{0,62})\.?$`
)

var (

	// ErrInvalid is raised when the path allocator can not return a path
	// because one of the path element is not valid.
	ErrInvalid = errors.New("invalid path")

	hostnameRegexRFC952 = regexp.MustCompile(hostnameRegexStringRFC952)
	fqdnRegexRFC1123    = regexp.MustCompile(fqdnRegexStringRFC1123)
	forbiddenNames      = append(
		kind.KindNames(),
		[]string{
			"node",
		}...,
	)
)

// New allocates a new path type from its elements
func New(name string, namespace string, kd string) (T, error) {
	var path T
	name = strings.ToLower(name)
	namespace = strings.ToLower(namespace)
	kd = strings.ToLower(kd)
	// apply defaults
	if kd == "" {
		kd = "svc"
	}
	if namespace == "" {
		namespace = "root"
	}

	k := kind.New(kd)
	switch k {
	case kind.Invalid:
		return path, errors.Wrapf(ErrInvalid, "invalid kind %s", kd)
	case kind.Nscfg:
		name = "namespace"
	}

	if name == "" {
		return path, errors.Wrap(ErrInvalid, "name is empty")
	}
	if !hostnameRegexRFC952.MatchString(name) {
		return path, errors.Wrapf(ErrInvalid, "invalid name %s (rfc952)", name)
	}
	if !hostnameRegexRFC952.MatchString(namespace) {
		return path, errors.Wrapf(ErrInvalid, "invalid namespace %s (rfc952)", namespace)
	}
	for _, reserved := range forbiddenNames {
		if reserved == name {
			return path, errors.Wrapf(ErrInvalid, "reserved name '%s'", name)
		}
	}
	path.Namespace = namespace
	path.Name = name
	path.Kind = k
	return path, nil
}

func (t T) String() string {
	var s string
	if t.Kind == kind.Invalid {
		return ""
	}
	if t.Namespace != "" && t.Namespace != "root" {
		s += t.Namespace + Separator
	}
	if (t.Kind != kind.Svc && t.Kind != kind.Ccfg) || s != "" {
		s += t.Kind.String() + Separator
	}
	return s + t.Name
}

// Parse returns a new path struct from a path string representation
func Parse(s string) (T, error) {
	var (
		name      string
		namespace string
		kd        string
	)
	s = strings.ToLower(s)
	l := strings.Split(s, Separator)
	switch len(l) {
	case 3:
		namespace = l[0]
		kd = l[1]
		name = l[2]
	case 2:
		switch l[1] {
		case "": // ex: ns1/
			namespace = l[0]
			kd = "nscfg"
			name = "namespace"
		default: // ex: cfg/c1
			namespace = "root"
			kd = l[0]
			name = l[1]
		}
	case 1:
		switch l[0] {
		case "cluster":
			namespace = "root"
			kd = "ccfg"
			name = l[0]
		default:
			namespace = "root"
			kd = "svc"
			name = l[0]
		}
	}
	return New(name, namespace, kd)
}

// MarshalJSON implements the json interface
func (t T) MarshalJSON() ([]byte, error) {
	buffer := bytes.NewBufferString(`"`)
	buffer.WriteString(t.String())
	buffer.WriteString(`"`)
	return buffer.Bytes(), nil
}

// UnmarshalJSON implements the json interface
func (t *T) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	p, err := Parse(s)
	if err != nil {
		return err
	}
	t.Name = p.Name
	t.Namespace = p.Namespace
	t.Kind = p.Kind
	return nil
}

//
// Match returns true if the object matches the pattern, using a fnmatch
// matching algorithm with a few special cases to mask the root namespace
// tricks and the svc object kind as default.
//
// Trick:
// The 'f*' pattern matches all svc objects in the root namespace.
// The '*' pattern matches all svc objects in all namespaces.
//
func (t T) Match(pattern string) bool {
	l := strings.Split(pattern, "/")
	s := t.String()
	f := fnmatch.FNM_IGNORECASE | fnmatch.FNM_PATHNAME
	switch len(l) {
	case 1:
		switch pattern {
		case "**":
			return true
		case "*":
			if fnmatch.Match("*/svc/*", s, f) {
				return true
			}
			if fnmatch.Match("*", s, f) {
				return true
			}
		default:
			if fnmatch.Match(pattern, s, f) {
				return true
			}
		}
	case 2:

		if l[0] == "svc" {
			// svc/foo => foo ... for root namespace
			if fnmatch.Match(l[1], s, f) {
				return true
			}
		}
		pattern = strings.Replace(pattern, "**", "*/*", 1)
		if fnmatch.Match(pattern, s, f) {
			return true
		}
	case 3:
		if l[1] == "svc" && l[0] == "*" {
			// */svc/foo => foo ... for root namespace
			if fnmatch.Match(l[2], s, f) {
				return true
			}
		}
		if fnmatch.Match(pattern, s, f) {
			return true
		}
	}
	return false
}

func (t Relation) String() string {
	return string(t)
}
