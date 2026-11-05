package archi

import (
	"strconv"
	"strings"
	"unicode/utf8"
)

// OMap is an insertion-ordered string-keyed map. Key order matters for XML attribute
// order, the order of folded children (doc/props/b) and for YAML output stability.
type OMap struct {
	keys []string
	m    map[string]any
}

// Obj is the intermediate model object: values are string, Obj or ObjList.
type Obj = *OMap

// ObjList is a list of strings / Objs.
type ObjList = []any

// NewObj returns an empty ordered map.
func NewObj() Obj { return &OMap{m: map[string]any{}} }

// O builds an Obj from alternating key, value arguments.
func O(kv ...any) Obj {
	o := NewObj()
	for i := 0; i+1 < len(kv); i += 2 {
		o.Set(kv[i].(string), kv[i+1])
	}
	return o
}

// Len returns the number of entries.
func (o *OMap) Len() int { return len(o.keys) }

// Keys returns a copy of the keys in insertion order.
func (o *OMap) Keys() []string { return append([]string(nil), o.keys...) }

// Has reports whether k is present.
func (o *OMap) Has(k string) bool { _, ok := o.m[k]; return ok }

// Get returns the value for k or nil.
func (o *OMap) Get(k string) any { return o.m[k] }

// Lookup returns the value and presence flag for k.
func (o *OMap) Lookup(k string) (any, bool) { v, ok := o.m[k]; return v, ok }

// Str returns the string value for k, or "" when absent or not a string.
func (o *OMap) Str(k string) string { s, _ := o.m[k].(string); return s }

// StrOK returns the string value for k and whether it is present as a string.
func (o *OMap) StrOK(k string) (string, bool) { s, ok := o.m[k].(string); return s, ok }

// Obj returns the Obj value for k, or nil.
func (o *OMap) Obj(k string) Obj { v, _ := o.m[k].(*OMap); return v }

// List returns the list value for k, or nil.
func (o *OMap) List(k string) ObjList { v, _ := o.m[k].([]any); return v }

// Set assigns k; a new key is appended, an existing key keeps its position.
func (o *OMap) Set(k string, v any) {
	if _, ok := o.m[k]; !ok {
		o.keys = append(o.keys, k)
	}
	o.m[k] = v
}

// Del removes k.
func (o *OMap) Del(k string) {
	if _, ok := o.m[k]; !ok {
		return
	}
	delete(o.m, k)
	for i, x := range o.keys {
		if x == k {
			o.keys = append(o.keys[:i], o.keys[i+1:]...)
			return
		}
	}
}

// Prepend puts k first (moving it if present).
func (o *OMap) Prepend(k string, v any) {
	o.Del(k)
	o.keys = append([]string{k}, o.keys...)
	o.m[k] = v
}

// InsertAfter inserts k right after the key `after` (appends when `after` is absent).
// An existing k is left untouched.
func (o *OMap) InsertAfter(after, k string, v any) {
	if o.Has(k) {
		return
	}
	o.m[k] = v
	for i, x := range o.keys {
		if x == after {
			o.keys = append(o.keys, "")
			copy(o.keys[i+2:], o.keys[i+1:])
			o.keys[i+1] = k
			return
		}
	}
	o.keys = append(o.keys, k)
}

// Single returns the only entry of a single-key Obj (a "kid").
func (o *OMap) Single() (string, any, bool) {
	if o == nil || len(o.keys) != 1 {
		return "", nil, false
	}
	return o.keys[0], o.m[o.keys[0]], true
}

// KeysAre reports whether the keys are exactly ks in that order.
func (o *OMap) KeysAre(ks ...string) bool {
	if len(o.keys) != len(ks) {
		return false
	}
	for i := range ks {
		if o.keys[i] != ks[i] {
			return false
		}
	}
	return true
}

// Clone makes a deep copy.
func (o *OMap) Clone() Obj {
	c := NewObj()
	for _, k := range o.keys {
		c.Set(k, cloneAny(o.m[k]))
	}
	return c
}

func cloneAny(v any) any {
	switch x := v.(type) {
	case *OMap:
		return x.Clone()
	case []any:
		out := make([]any, len(x))
		for i := range x {
			out[i] = cloneAny(x[i])
		}
		return out
	}
	return v
}

// ------------------------------------------------------------------ JSON

// PyJSON renders v like Python's json.dumps(v, ensure_ascii=False): ", " and ": " separators.
// Supported: nil, string, int, bool, Obj, []any, []Obj.
func PyJSON(v any) string {
	var b strings.Builder
	pyJSON(&b, v)
	return b.String()
}

func pyJSON(b *strings.Builder, v any) {
	switch x := v.(type) {
	case nil:
		b.WriteString("null")
	case string:
		b.WriteString(jsonString(x))
	case int:
		b.WriteString(strconv.Itoa(x))
	case bool:
		if x {
			b.WriteString("true")
		} else {
			b.WriteString("false")
		}
	case *OMap:
		b.WriteByte('{')
		for i, k := range x.keys {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(jsonString(k))
			b.WriteString(": ")
			pyJSON(b, x.m[k])
		}
		b.WriteByte('}')
	case []any:
		b.WriteByte('[')
		for i, e := range x {
			if i > 0 {
				b.WriteString(", ")
			}
			pyJSON(b, e)
		}
		b.WriteByte(']')
	case []*OMap:
		b.WriteByte('[')
		for i, e := range x {
			if i > 0 {
				b.WriteString(", ")
			}
			pyJSON(b, e)
		}
		b.WriteByte(']')
	default:
		b.WriteString(jsonString("?"))
	}
}

func jsonString(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		case '\b':
			b.WriteString(`\b`)
		case '\f':
			b.WriteString(`\f`)
		default:
			if r < 0x20 {
				b.WriteString(`\u00`)
				b.WriteByte("0123456789abcdef"[r>>4])
				b.WriteByte("0123456789abcdef"[r&0xf])
			} else if r == utf8.RuneError {
				b.WriteString(`\ufffd`)
			} else {
				b.WriteRune(r)
			}
		}
	}
	b.WriteByte('"')
	return b.String()
}
