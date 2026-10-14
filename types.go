package smarter

import (
	"strconv"
	"strings"
)

// JSONValue is a decoded JSON value whose shape really is free: a string, a float64, a bool, nil, a
// []any or a map[string]any. It is the Python SDK's SmarterJsonType. See ADR-013.
type JSONValue = any

// JSONObject is a JSON object whose shape is not fixed, e.g. a manifest's spec, whose fields depend
// on its kind. It is the Python SDK's SmarterJsonObjectType.
type JSONObject = map[string]any

// ID is a resource's id, as text: its integer primary key, e.g. IntID(5), or its hashed id, e.g.
// ID("rMTAwMDAwNQx"). It is the Python SDK's SmarterIdType.
type ID string

// IntID is the ID of a resource's integer primary key.
func IntID(id int) ID {
	return ID(strconv.Itoa(id))
}

// segment is the id as a url path segment, or an error if it is empty or contains a slash.
func (id ID) segment() (string, error) {
	text := strings.TrimSpace(string(id))
	if text == "" || strings.Contains(text, "/") {
		return "", &ValueError{Message: "invalid resource id: " + strconv.Quote(string(id))}
	}
	return text, nil
}

// Ptr returns a pointer to a copy of v, e.g. for an optional field: smarter.Ptr(true).
func Ptr[T any](v T) *T {
	return &v
}

// Deref returns the value that p points to, or the zero value of T if p is nil.
func Deref[T any](p *T) T {
	if p == nil {
		var zero T
		return zero
	}
	return *p
}

// String returns the id's text.
func (id ID) String() string { return string(id) }
