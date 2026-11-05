package archi

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	xsiURI    = "http://www.w3.org/2001/XMLSchema-instance"
	archiURI  = "http://www.archimatetool.com/archimate"
	xmlURI    = "http://www.w3.org/XML/1998/namespace"
	xmlHeader = "<?xml version='1.0' encoding='UTF-8'?>\n"
)

// Reserved object keys that cannot be XML attribute names.
var Reserved = map[string]bool{"doc": true, "props": true, "b": true, "kids": true, "_tag": true, "_text": true, "_ns": true}

// Bounds attribute names in canonical order.
var Bounds = []string{"x", "y", "width", "height"}

// ChildTypes are diagram node types serialised as <child>.
var ChildTypes = map[string]bool{"DiagramObject": true, "Group": true, "Note": true,
	"DiagramModelReference": true, "CanvasModelBlock": true, "CanvasModelSticky": true, "CanvasModelImage": true}

var intRe = regexp.MustCompile(`^-?\d+$`)

// ShortType strips the "archimate:" prefix.
func ShortType(t string) string { return strings.TrimPrefix(t, "archimate:") }

// FullType adds the "archimate:" prefix unless the key is already qualified.
func FullType(k string) string {
	if strings.Contains(k, ":") {
		return k
	}
	return "archimate:" + k
}

// InferTag returns the XML tag normally used for an object with this type key.
func InferTag(key string) string {
	base := key
	if i := strings.LastIndex(key, ":"); i >= 0 {
		base = key[i+1:]
	}
	if strings.HasSuffix(base, "Connection") {
		return "sourceConnection"
	}
	if ChildTypes[base] {
		return "child"
	}
	return "element"
}

// IsTypeKey: a key naming an xsi:type (qualified or capitalised) rather than a plain tag.
func IsTypeKey(k string) bool {
	if k == "" {
		return false
	}
	if strings.Contains(k, ":") {
		return true
	}
	r, _ := utf8.DecodeRuneInString(k)
	return unicode.IsUpper(r)
}

// ------------------------------------------------------------------ reading

// ReadXML reads a plain XML model file and returns UTF-8 text (BOM removed).
func ReadXML(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	b = bytes.TrimPrefix(b, []byte{0xef, 0xbb, 0xbf})
	if len(b) == 0 || b[0] != '<' {
		return "", fmt.Errorf("not a plain xml model (zip container?): %s", path)
	}
	return string(b), nil
}

type xattr struct{ space, local, value string }

type xnode struct {
	space string
	local string
	attrs []xattr
	text  string
	kids  []*xnode
}

type nsDecl struct{ prefix, uri string }

func parseTree(text string) (*xnode, []nsDecl, map[string]string, error) {
	dec := xml.NewDecoder(strings.NewReader(text))
	dec.Strict = true
	var root *xnode
	var stack []*xnode
	var rootDecls []nsDecl
	rev := map[string]string{xsiURI: "xsi", xmlURI: "xml"}
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, nil, nil, err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			n := &xnode{space: t.Name.Space, local: t.Name.Local}
			for _, a := range t.Attr {
				if a.Name.Space == "xmlns" || (a.Name.Space == "" && a.Name.Local == "xmlns") {
					p := a.Name.Local
					if a.Name.Space == "" {
						p = ""
					}
					if root == nil {
						rootDecls = append(rootDecls, nsDecl{p, a.Value})
					}
					if _, seen := rev[a.Value]; !seen && p != "" {
						rev[a.Value] = p
					}
					continue
				}
				n.attrs = append(n.attrs, xattr{a.Name.Space, a.Name.Local, a.Value})
			}
			if len(stack) > 0 {
				p := stack[len(stack)-1]
				p.kids = append(p.kids, n)
			} else if root == nil {
				root = n
			} else {
				return nil, nil, nil, errors.New("multiple root elements")
			}
			stack = append(stack, n)
		case xml.EndElement:
			stack = stack[:len(stack)-1]
		case xml.CharData:
			if len(stack) > 0 {
				if n := stack[len(stack)-1]; len(n.kids) == 0 {
					n.text += string(t)
				}
			}
		case xml.Comment:
			return nil, nil, nil, errors.New("comments/PIs unsupported")
		case xml.ProcInst:
			if t.Target != "xml" || root != nil {
				return nil, nil, nil, errors.New("comments/PIs unsupported")
			}
		case xml.Directive:
			return nil, nil, nil, errors.New("DOCTYPE/entity declarations unsupported")
		}
	}
	if root == nil {
		return nil, nil, nil, errors.New("empty document")
	}
	return root, rootDecls, rev, nil
}

func (a xattr) qname(rev map[string]string) string {
	if a.space == "" {
		return a.local
	}
	if p, ok := rev[a.space]; ok {
		return p + ":" + a.local
	}
	return a.space + ":" + a.local
}

func nonBlank(s string) (string, bool) {
	if strings.TrimSpace(s) == "" {
		return "", false
	}
	return s, true
}

func conv(e *xnode, rev map[string]string) (string, any, error) {
	var typ string
	hasType := false
	for _, a := range e.attrs {
		if a.space == xsiURI && a.local == "type" {
			typ, hasType = a.value, true
		}
	}
	key := e.local
	if hasType {
		key = ShortType(typ)
	}
	val := NewObj()
	if hasType && InferTag(key) != e.local {
		val.Set("_tag", e.local)
	}
	for _, a := range e.attrs {
		if a.space == xsiURI && a.local == "type" {
			continue
		}
		k := a.qname(rev)
		if Reserved[k] {
			return "", nil, fmt.Errorf("reserved attribute %s", k)
		}
		val.Set(k, a.value)
	}
	txt, hasTxt := nonBlank(e.text)
	if len(e.kids) == 0 && hasTxt {
		if val.Len() == 0 {
			return key, txt, nil
		}
		val.Set("_text", txt)
	}
	var kids []any
	seenKid := false
	for _, c := range e.kids {
		simple := len(c.kids) == 0
		folded := false
		ctxt, cHas := nonBlank(c.text)
		if c.local == "documentation" && simple && len(c.attrs) == 0 && !val.Has("doc") {
			if cHas {
				val.Set("doc", ctxt)
				folded = true
			}
		}
		if !folded && !seenKid && simple {
			names := make([]string, len(c.attrs))
			for i, a := range c.attrs {
				names[i] = a.qname(rev)
			}
			switch {
			case c.local == "property" && len(names) == 2 && names[0] == "key" && names[1] == "value" && !cHas:
				props, _ := val.Get("props").([]any)
				props = append(props, O(c.attrs[0].value, c.attrs[1].value))
				val.Set("props", props)
				folded = true
			case c.local == "bounds" && !val.Has("b") && !cHas && len(names) > 0 && boundsOK(c.attrs):
				d := map[string]string{}
				for _, a := range c.attrs {
					d[a.local] = a.value
				}
				parts := make([]string, len(Bounds))
				for i, n := range Bounds {
					if v, ok := d[n]; ok {
						parts[i] = v
					} else {
						parts[i] = "_"
					}
				}
				val.Set("b", strings.Join(parts, " "))
				folded = true
			}
		}
		if !folded {
			seenKid = true
			k, v, err := conv(c, rev)
			if err != nil {
				return "", nil, err
			}
			kids = append(kids, O(k, v))
		}
	}
	if len(kids) > 0 {
		val.Set("kids", kids)
	}
	return key, val, nil
}

// boundsOK: all attrs are x/y/width/height, in that relative order, with integer values.
func boundsOK(attrs []xattr) bool {
	last := -1
	for _, a := range attrs {
		if a.space != "" {
			return false
		}
		idx := -1
		for i, n := range Bounds {
			if n == a.local {
				idx = i
			}
		}
		if idx < 0 || idx < last || !intRe.MatchString(a.value) {
			return false
		}
		last = idx
	}
	return true
}

// ParseXML converts XML text to the intermediate object {rootKey: rootVal}.
// Entity expansion is not performed; comments, PIs and DOCTYPE are rejected.
func ParseXML(text string) (Obj, error) {
	root, decls, rev, err := parseTree(text)
	if err != nil {
		return nil, err
	}
	key, val, err := conv(root, rev)
	if err != nil {
		return nil, err
	}
	v, ok := val.(Obj)
	if !ok {
		return nil, errors.New("root element has no attributes or children")
	}
	extra := NewObj()
	for _, d := range decls {
		if d.prefix == "" || (d.prefix == "xsi" && d.uri == xsiURI) || (d.prefix == "archimate" && d.uri == archiURI) {
			continue
		}
		extra.Set(d.prefix, d.uri)
	}
	if extra.Len() > 0 {
		v.Prepend("_ns", extra)
	}
	return O(key, v), nil
}

// ------------------------------------------------------------------ writing

// SerializeXML renders the object as XML, lxml pretty-print style.
func SerializeXML(obj Obj) (string, error) {
	rk, rv, ok := obj.Single()
	if !ok {
		return "", errors.New("model object must have exactly one root key")
	}
	var b strings.Builder
	b.WriteString(xmlHeader)
	ns := []nsDecl{{"xsi", xsiURI}, {"archimate", archiURI}}
	if v, ok := rv.(Obj); ok {
		if e := v.Obj("_ns"); e != nil {
			for _, p := range e.Keys() {
				ns = append(ns, nsDecl{p, e.Str(p)})
			}
		}
	}
	if err := build(&b, rk, rv, 0, ns); err != nil {
		return "", err
	}
	return b.String(), nil
}

func build(b *strings.Builder, key string, val any, depth int, rootNS []nsDecl) error {
	indent := strings.Repeat("  ", depth)
	root := rootNS != nil
	var tag, typ, text string
	hasText := false
	var attrs *OMap
	var kids []any
	switch v := val.(type) {
	case string:
		tag, text, hasText = key, v, true
	case Obj:
		if IsTypeKey(key) {
			typ = FullType(key)
			tag = v.Str("_tag")
			if tag == "" {
				tag = InferTag(key)
			}
		} else {
			tag = key
		}
		attrs = v
		text, hasText = v.StrOK("_text")
		kids = v.List("kids")
	default:
		return fmt.Errorf("element %s: unexpected value type %T", key, val)
	}
	if root {
		tag = "archimate:" + tag
	}
	b.WriteString(indent)
	b.WriteByte('<')
	b.WriteString(tag)
	if root {
		for _, d := range rootNS {
			fmt.Fprintf(b, " xmlns:%s=\"%s\"", d.prefix, escAttr(d.uri))
		}
	}
	if typ != "" {
		fmt.Fprintf(b, " xsi:type=\"%s\"", escAttr(typ))
	}
	if attrs != nil {
		for _, k := range attrs.keys {
			if Reserved[k] {
				continue
			}
			s, ok := attrs.m[k].(string)
			if !ok {
				return fmt.Errorf("%s: attribute %q must be a scalar", key, k)
			}
			fmt.Fprintf(b, " %s=\"%s\"", k, escAttr(s))
		}
	}
	// folded children, in stored order
	var folded strings.Builder
	if attrs != nil {
		for _, k := range attrs.keys {
			switch k {
			case "doc":
				s, ok := attrs.m[k].(string)
				if !ok {
					return fmt.Errorf("%s: doc must be a string", key)
				}
				fmt.Fprintf(&folded, "%s  <documentation>%s</documentation>\n", indent, escText(s))
			case "props":
				l, ok := attrs.m[k].([]any)
				if !ok {
					return fmt.Errorf("%s: props must be a list", key)
				}
				for _, p := range l {
					po, ok := p.(Obj)
					pk, pv, one := po.Single()
					s, isStr := pv.(string)
					if !ok || !one || !isStr {
						return fmt.Errorf("%s: invalid property entry", key)
					}
					fmt.Fprintf(&folded, "%s  <property key=\"%s\" value=\"%s\"/>\n", indent, escAttr(pk), escAttr(s))
				}
			case "b":
				s, ok := attrs.m[k].(string)
				if !ok {
					return fmt.Errorf("%s: b must be a string", key)
				}
				fmt.Fprintf(&folded, "%s  <bounds", indent)
				for i, x := range strings.Fields(s) {
					if i < len(Bounds) && x != "_" {
						fmt.Fprintf(&folded, " %s=\"%s\"", Bounds[i], escAttr(x))
					}
				}
				folded.WriteString("/>\n")
			}
		}
	}
	if folded.Len() == 0 && len(kids) == 0 {
		if hasText {
			b.WriteByte('>')
			b.WriteString(escText(text))
			fmt.Fprintf(b, "</%s>\n", tag)
		} else {
			b.WriteString("/>\n")
		}
		return nil
	}
	b.WriteByte('>')
	if hasText {
		b.WriteString(escText(text))
	}
	b.WriteByte('\n')
	b.WriteString(folded.String())
	for _, k := range kids {
		ko, ok := k.(Obj)
		kk, kv, one := ko.Single()
		if !ok || !one {
			return fmt.Errorf("%s: kid must be a single-key map", key)
		}
		if err := build(b, kk, kv, depth+1, nil); err != nil {
			return err
		}
	}
	fmt.Fprintf(b, "%s</%s>\n", indent, tag)
	return nil
}

func escText(s string) string {
	if !strings.ContainsAny(s, "&<>\r") {
		return s
	}
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", "\r", "&#13;")
	return r.Replace(s)
}

func escAttr(s string) string {
	if !strings.ContainsAny(s, "&<>\"\r\n\t") {
		return s
	}
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", "\"", "&quot;", "\r", "&#13;", "\n", "&#10;", "\t", "&#9;")
	return r.Replace(s)
}
