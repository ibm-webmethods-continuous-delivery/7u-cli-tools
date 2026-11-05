package archi

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

// REL11 are the relationship families written as folder-level groups.
var REL11 = map[string]bool{"Access": true, "Aggregation": true, "Assignment": true, "Association": true,
	"Composition": true, "Flow": true, "Influence": true, "Realization": true, "Serving": true,
	"Specialization": true, "Triggering": true}

var (
	relRe    = regexp.MustCompile(`(?s)^([\pL\pN_]+)#(\S+) (\S+)>(\S+)(?: (.*))?$`)
	spacedRe = regexp.MustCompile(`(?s)^([\pL\pN_]+)#(\S+) (.+?) > (.+?)(?: \| (.*))?$`)
	itemRe   = regexp.MustCompile(`(?s)^(\S+) (\S+)>(\S+)(?: (.*))?$`)
	idRe     = regexp.MustCompile(`^[\pL\pN_]+#(\S+) `)
	badName  = regexp.MustCompile(`[>|\r\n\t]`)
)

const relSuffix = "Relationship"

var notElementSuffix = []string{"Relationship", "Connection", "DiagramModel", "Model"}

type match struct {
	s   string
	loc []int
}

func doMatch(re *regexp.Regexp, s string) (match, bool) {
	loc := re.FindStringSubmatchIndex(s)
	return match{s, loc}, loc != nil
}

func (m match) g(i int) string {
	if m.loc[2*i] < 0 {
		return ""
	}
	return m.s[m.loc[2*i]:m.loc[2*i+1]]
}

func (m match) has(i int) bool { return m.loc[2*i] >= 0 }

// ------------------------------------------------------------ element index

// ElementIndex maps element names and ids, used to resolve and abbreviate relation endpoints.
type ElementIndex struct {
	Names map[string][]string // name -> ids
	IDs   map[string]bool     // every id seen (elements, relations, nodes)
	ByID  map[string]string   // element id -> name
}

// UsableName returns the element name for id when it is safe to use as an endpoint, else the id itself.
func (ix *ElementIndex) UsableName(ref string) string {
	n, ok := ix.ByID[ref]
	if ok && n != "" && n == strings.TrimSpace(n) && !badName.MatchString(n) &&
		len(ix.Names[n]) == 1 && !ix.IDs[n] {
		return n
	}
	return ref
}

// Resolve maps an endpoint (id or unique name) to an id.
func (ix *ElementIndex) Resolve(ref string) (string, error) {
	if ix.IDs[ref] {
		return ref, nil
	}
	found := ix.Names[ref]
	if len(found) == 1 {
		return found[0], nil
	}
	kind := "unknown"
	if len(found) > 0 {
		kind = "ambiguous"
	}
	return "", fmt.Errorf("%s element reference: %s", kind, PyRepr(ref))
}

// walkKids calls fn for every "kids" list found under x (parents before descendants).
func walkKids(x any, fn func(kids []any)) {
	switch v := x.(type) {
	case Obj:
		if kids, ok := v.Get("kids").([]any); ok {
			fn(kids)
		}
		for _, k := range v.keys {
			walkKids(v.m[k], fn)
		}
	case []any:
		for _, i := range v {
			walkKids(i, fn)
		}
	}
}

// BuildElementIndex scans an object (with relations in compact notation) for ids and element names.
func BuildElementIndex(obj any) *ElementIndex {
	ix := &ElementIndex{Names: map[string][]string{}, IDs: map[string]bool{}, ByID: map[string]string{}}
	walkKids(obj, func(kids []any) {
		for _, kid := range kids {
			if s, ok := kid.(string); ok {
				if m, ok := doMatch(idRe, s); ok {
					ix.IDs[m.g(1)] = true
				}
				continue
			}
			ko, ok := kid.(Obj)
			if !ok {
				continue
			}
			if text, _, ok := relItem(kid); ok {
				if m, ok := doMatch(idRe, text); ok {
					ix.IDs[m.g(1)] = true
				}
				continue
			}
			for _, k := range ko.keys {
				v, ok := ko.m[k].(Obj)
				if !ok {
					continue
				}
				id, ok := v.StrOK("id")
				if !ok {
					continue
				}
				ix.IDs[id] = true
				if name, ok := v.StrOK("name"); ok && startsUpper(k) && !ChildTypes[k] && !hasAnySuffix(k, notElementSuffix) {
					ix.Names[name] = append(ix.Names[name], id)
					ix.ByID[id] = name
				}
			}
		}
	})
	return ix
}

func startsUpper(k string) bool {
	for _, r := range k {
		return unicode.IsUpper(r)
	}
	return false
}

func hasAnySuffix(s string, sfx []string) bool {
	for _, x := range sfx {
		if strings.HasSuffix(s, x) {
			return true
		}
	}
	return false
}

// ------------------------------------------------------------------ compact

var relBases = [][]string{{"id", "source", "target"}, {"name", "id", "source", "target"}}
var relTails = [][]string{{}, {"doc"}, {"props"}, {"doc", "props"}}

func hasPrefix(keys, p []string) bool {
	if len(keys) < len(p) {
		return false
	}
	for i := range p {
		if keys[i] != p[i] {
			return false
		}
	}
	return true
}

func sliceEq(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// relString returns the compact string form plus doc/props extras, or ok=false when not representable.
func relString(k string, v any) (string, Obj, bool) {
	o, isObj := v.(Obj)
	if !strings.HasSuffix(k, relSuffix) || !isObj {
		return "", nil, false
	}
	keys := o.keys
	var base []string
	for _, b := range relBases {
		if hasPrefix(keys, b) {
			base = b
			break
		}
	}
	if base == nil {
		return "", nil, false
	}
	tail := keys[len(base):]
	okTail := false
	for _, t := range relTails {
		if sliceEq(tail, t) {
			okTail = true
		}
	}
	if !okTail {
		return "", nil, false
	}
	extras := NewObj()
	for _, a := range tail {
		extras.Set(a, o.m[a])
	}
	if d, ok := extras.Lookup("doc"); ok {
		if _, isStr := d.(string); !isStr {
			return "", nil, false
		}
	}
	if p, ok := extras.Lookup("props"); ok {
		if _, isList := p.([]any); !isList {
			return "", nil, false
		}
	}
	for _, a := range base {
		if _, ok := o.m[a].(string); !ok {
			return "", nil, false
		}
	}
	if n, ok := o.StrOK("name"); ok && n == "" {
		return "", nil, false
	}
	for _, a := range []string{"id", "source", "target"} {
		s := o.Str(a)
		if s == "" || hasPySpace(s) || strings.Contains(s, ">") {
			return "", nil, false
		}
	}
	s := fmt.Sprintf("%s#%s %s>%s", strings.TrimSuffix(k, relSuffix), o.Str("id"), o.Str("source"), o.Str("target"))
	if n, ok := o.StrOK("name"); ok {
		s += " " + n
	}
	return s, extras, true
}

// CompactRelations rewrites relationship kids to "Type#id src>tgt [name]" strings (in place).
func CompactRelations(obj Obj) Obj {
	walkKids(obj, func(kids []any) {
		for i, kid := range kids {
			ko, ok := kid.(Obj)
			if !ok {
				continue
			}
			k, v, one := ko.Single()
			if !one {
				continue
			}
			s, extras, ok := relString(k, v)
			if !ok {
				continue
			}
			if extras.Len() == 0 {
				kids[i] = s
			} else {
				kids[i] = O(s, extras)
			}
		}
	})
	return obj
}

// relItem normalises a relation kid (string or {string: extras}) to (text, extras).
func relItem(kid any) (string, Obj, bool) {
	switch x := kid.(type) {
	case string:
		return x, NewObj(), true
	case Obj:
		k, v, one := x.Single()
		if !one {
			return "", nil, false
		}
		vo, isObj := v.(Obj)
		if !isObj {
			return "", nil, false
		}
		if relRe.MatchString(k) || spacedRe.MatchString(k) {
			return k, vo, true
		}
	}
	return "", nil, false
}

// ExpandRelations turns relation strings back into Type+"Relationship" objects, resolving names to ids.
func ExpandRelations(obj Obj) (Obj, error) {
	ix := BuildElementIndex(obj)
	var firstErr error
	walkKids(obj, func(kids []any) {
		for i, kid := range kids {
			if firstErr != nil {
				return
			}
			text, extras, ok := relItem(kid)
			if !ok {
				continue
			}
			var typ, rid, s, t string
			var nm string
			var hasNm bool
			if m, ok := doMatch(relRe, text); ok {
				typ, rid, s, t, nm, hasNm = m.g(1), m.g(2), m.g(3), m.g(4), m.g(5), m.has(5)
			} else if m, ok := doMatch(spacedRe, text); ok {
				typ, rid, s, t, nm, hasNm = m.g(1), m.g(2), m.g(3), m.g(4), m.g(5), m.has(5)
				var err error
				if s, err = ix.Resolve(s); err != nil {
					firstErr = err
					return
				}
				if t, err = ix.Resolve(t); err != nil {
					firstErr = err
					return
				}
			} else {
				continue
			}
			v := NewObj()
			if hasNm {
				v.Set("name", nm)
			}
			v.Set("id", rid)
			v.Set("source", s)
			v.Set("target", t)
			for _, k := range extras.keys {
				v.Set(k, extras.m[k])
			}
			kids[i] = O(typ+relSuffix, v)
		}
	})
	if firstErr != nil {
		return nil, firstErr
	}
	return obj, nil
}

// ------------------------------------------------------------ name endpoints

func relDictForm(v any) (string, any, bool) {
	o, ok := v.(Obj)
	if !ok {
		return "", nil, false
	}
	k, val, one := o.Single()
	if !one {
		return "", nil, false
	}
	if _, isObj := val.(Obj); !isObj {
		return "", nil, false
	}
	return k, val, true
}

func namesConv(item string, ix *ElementIndex) string {
	m, ok := doMatch(itemRe, item)
	if !ok {
		return item
	}
	rid, s, t := m.g(1), m.g(2), m.g(3)
	sn, tn := ix.UsableName(s), ix.UsableName(t)
	if sn == s && tn == t {
		return item
	}
	out := rid + " " + sn + " > " + tn
	if m.has(4) {
		out += " | " + m.g(4)
	}
	return out
}

// NamesCompact rewrites relation endpoints (in REL11 groups of a compacted tree) to element names where safe.
func NamesCompact(tree any, ix *ElementIndex) {
	switch x := tree.(type) {
	case Obj:
		for _, k := range x.Keys() {
			v := x.m[k]
			list, isList := v.([]any)
			if REL11[k] && isList {
				out := make([]any, len(list))
				for j, it := range list {
					if s, ok := it.(string); ok {
						out[j] = namesConv(s, ix)
					} else if ik, iv, ok := relDictForm(it); ok {
						out[j] = O(namesConv(ik, ix), iv)
					} else {
						out[j] = it
					}
				}
				x.Set(k, out)
			} else {
				NamesCompact(v, ix)
			}
		}
	case []any:
		for _, i := range x {
			NamesCompact(i, ix)
		}
	}
}
