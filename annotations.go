package smarter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// Annotations are a resource's annotations, as in Kubernetes: a map of string keys to string values
// that clients and integrations attach to a resource for their own purposes, e.g.
// acme.com/crm-account-id: A-10042 to associate an LLMClient with an account in a CRM. A key is an
// optional DNS-subdomain prefix that names its owner, then a slash and a name of one or more
// slash-separated segments. The prefix smarter.sh and its subdomains are reserved for the platform.
// It is the Python SDK's SmarterAnnotations.
//
// Smarter 0.37 and earlier wrote annotations as a list of one-key maps,
// [{"acme.com/crm-account-id": "A-10042"}], with values of any type. The platform still accepts that
// form, and so does the SDK: decoding either form gives the map, with each value converted to the
// string that the platform stores (see [NormalizeAnnotations]). The platform decides whether a key
// is valid, so the SDK doesn't check keys.
type Annotations map[string]string

// UnmarshalJSON decodes annotations in either form: the map, or the legacy list of one-key maps.
func (a *Annotations) UnmarshalJSON(data []byte) error {
	var value JSONValue
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return err
	}
	if value == nil {
		*a = nil
		return nil
	}
	result, err := NormalizeAnnotations(value)
	if err != nil {
		return err
	}
	*a = result
	return nil
}

// NormalizeAnnotations returns annotations in either accepted form as a map of strings: the map,
// {"acme.com/crm-id": "42"}, or the list of one-key maps of Smarter 0.37 and earlier,
// [{"acme.com/crm-id": 42}]. When a key appears twice in a list, its last value wins. nil is an empty
// map. Each value is converted with [AnnotationValue]. It returns a [*ValueError] if value is neither
// form, or an item of a list is not a map.
func NormalizeAnnotations(value JSONValue) (Annotations, error) {
	result := Annotations{}
	switch v := value.(type) {
	case nil:
	case map[string]any:
		for key, item := range v {
			result[key] = AnnotationValue(item)
		}
	case []any:
		for _, item := range v {
			object, ok := item.(map[string]any)
			if !ok {
				return nil, &ValueError{Message: fmt.Sprintf("each item of a list of annotations must be a key-value map, not %v", item)}
			}
			for key, entry := range object {
				result[key] = AnnotationValue(entry)
			}
		}
	default:
		return nil, &ValueError{Message: fmt.Sprintf("annotations must be a map of keys to values, not %T", value)}
	}
	return result, nil
}

// AnnotationValue returns an annotation's value as the string that the platform stores. YAML reads
// unquoted scalars such as true, 42 or 2026-10-28 as other types, so they are converted to the
// string that was written: booleans to true or false, numbers as Python writes them (1.0 stays
// 1.0), dates to ISO 8601, and nil to an empty string. A list or an object, which older platforms
// stored, is kept as its JSON text, as the platform's own upgrade did.
func AnnotationValue(value JSONValue) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return v
	case bool:
		if v {
			return "true"
		}
		return "false"
	case json.Number:
		return v.String()
	case float64:
		return pythonFloat(v)
	case float32:
		return pythonFloat(float64(v))
	case time.Time:
		if v.Equal(v.Truncate(24*time.Hour)) && v.Location() == time.UTC {
			return v.Format(time.DateOnly)
		}
		if v.Nanosecond() != 0 {
			return v.Format("2006-01-02T15:04:05.000000-07:00")
		}
		return v.Format("2006-01-02T15:04:05-07:00")
	case []any, map[string]any:
		data, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprint(v)
		}
		return string(data)
	default:
		return fmt.Sprint(v)
	}
}

// pythonFloat formats a float as Python's str() does, which the platform uses: a whole number keeps
// its .0, e.g. 1.0, and very large or small numbers use an exponent, e.g. 1e+16.
func pythonFloat(f float64) string {
	magnitude := math.Abs(f)
	switch {
	case f == math.Trunc(f) && magnitude < 1e16:
		return strconv.FormatFloat(f, 'f', 1, 64)
	case magnitude >= 1e-4 && magnitude < 1e16:
		return strconv.FormatFloat(f, 'f', -1, 64)
	default:
		return strconv.FormatFloat(f, 'g', -1, 64)
	}
}

// annotationSelector returns an annotation selector's requirements, each trimmed, without blank ones.
func annotationSelector(requirements []string) []string {
	var result []string
	for _, requirement := range requirements {
		if trimmed := strings.TrimSpace(requirement); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
