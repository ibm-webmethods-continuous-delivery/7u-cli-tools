package archi

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Canonical form used by the conformance tests (canon_sig of the Python oracle):
// order-insensitive comparison that renames view node ids by ordinal.

type cnode struct {
	tag   string // Clark notation
	attrs []cattr
	text  string
	kids  []*cnode
}

type cattr struct{ k, v string }

func clark(space, local string) string {
	if space == "" {
		return local
	}
	return "{" + space + "}" + local
}

func toCnode(x *xnode) *cnode {
	c := &cnode{tag: clark(x.space, x.local), text: x.text}
	for _, a := range x.attrs {
		c.attrs = append(c.attrs, cattr{clark(a.space, a.local), a.value})
	}
	for _, k := range x.kids {
		c.kids = append(c.kids, toCnode(k))
	}
	return c
}

func (c *cnode) get(name string) (string, bool) {
	for _, a := range c.attrs {
		if a.k == name {
			return a.v, true
		}
	}
	return "", false
}

func (c *cnode) set(name, val string) {
	for i := range c.attrs {
		if c.attrs[i].k == name {
			c.attrs[i].v = val
			return
		}
	}
	c.attrs = append(c.attrs, cattr{name, val})
}

func (c *cnode) iter(fn func(*cnode)) {
	fn(c)
	for _, k := range c.kids {
		k.iter(fn)
	}
}

const xsiType = "{" + xsiURI + "}type"

type skey struct {
	tag, typ, name, id string
	n                  int
	names              string
}

func skeyOf(e *cnode) skey {
	var names []string
	for _, k := range e.kids {
		k.iter(func(d *cnode) {
			n, _ := d.get("name")
			names = append(names, n)
		})
	}
	sort.Strings(names)
	t, _ := e.get(xsiType)
	nm, _ := e.get("name")
	id := ""
	if e.tag != "folder" {
		id, _ = e.get("id")
	}
	return skey{e.tag, t, nm, id, len(names), strings.Join(names, "|")}
}

func skeyLess(a, b skey) bool {
	switch {
	case a.tag != b.tag:
		return a.tag < b.tag
	case a.typ != b.typ:
		return a.typ < b.typ
	case a.name != b.name:
		return a.name < b.name
	case a.id != b.id:
		return a.id < b.id
	case a.n != b.n:
		return a.n < b.n
	}
	return a.names < b.names
}

func (c *cnode) sig(loose bool) []any {
	attrs := make([]cattr, 0, len(c.attrs))
	for _, a := range c.attrs {
		if loose && c.tag == "folder" && a.k == "id" {
			continue
		}
		attrs = append(attrs, a)
	}
	sort.SliceStable(attrs, func(i, j int) bool { return attrs[i].k < attrs[j].k })
	al := make([]any, len(attrs))
	for i, a := range attrs {
		al[i] = []any{a.k, a.v}
	}
	text := ""
	if len(c.kids) == 0 {
		text = strings.TrimSpace(c.text)
	}
	kids := append([]*cnode(nil), c.kids...)
	sort.SliceStable(kids, func(i, j int) bool {
		return kids[i].tag == "documentation" && kids[j].tag != "documentation"
	})
	kl := make([]any, len(kids))
	for i, k := range kids {
		kl[i] = k.sig(loose)
	}
	return []any{c.tag, al, text, kl}
}

// CanonSig computes the canonical signature of XML text as nested JSON-ready slices.
// loose also ignores folder ids and the order of children inside folders and the model.
func CanonSig(text string, loose bool) ([]any, error) {
	xr, _, _, err := parseTree(text)
	if err != nil {
		return nil, err
	}
	root := toCnode(xr)
	if loose {
		var targets []*cnode
		root.iter(func(e *cnode) {
			if e == root || e.tag == "folder" {
				targets = append(targets, e)
			}
		})
		for _, e := range targets {
			keys := make(map[*cnode]skey, len(e.kids))
			for _, k := range e.kids {
				keys[k] = skeyOf(k)
			}
			sort.SliceStable(e.kids, func(i, j int) bool { return skeyLess(keys[e.kids[i]], keys[e.kids[j]]) })
		}
	}
	mapping := map[string]string{}
	var nodes []*cnode
	root.iter(func(e *cnode) {
		if t, _ := e.get(xsiType); t != "archimate:ArchimateDiagramModel" {
			return
		}
		e.iter(func(n *cnode) {
			if n == e || (n.tag != "child" && n.tag != "sourceConnection") {
				return
			}
			if id, _ := n.get("id"); id != "" {
				mapping[id] = "n" + strconv.Itoa(len(mapping))
				nodes = append(nodes, n)
			}
		})
	})
	for _, n := range nodes {
		for _, a := range []string{"id", "source", "target"} {
			if v, ok := n.get(a); ok {
				if m, in := mapping[v]; in {
					n.set(a, m)
				}
			}
		}
		if tc, _ := n.get("targetConnections"); tc != "" {
			parts := strings.Fields(tc)
			for i, t := range parts {
				if m, ok := mapping[t]; ok {
					parts[i] = m
				}
			}
			sort.Strings(parts)
			n.set("targetConnections", strings.Join(parts, " "))
		}
	}
	return root.sig(loose), nil
}

// CanonEqual reports whether two XML texts are canonically equal (loose form).
func CanonEqual(a, b string) (bool, error) {
	sa, err := CanonSig(a, true)
	if err != nil {
		return false, err
	}
	sb, err := CanonSig(b, true)
	if err != nil {
		return false, err
	}
	return fmt.Sprint(sa) == fmt.Sprint(sb), nil
}

// RoundTrip splits and rejoins XML text (without validation) and reports canonical equality.
func RoundTrip(text string, namesMode bool) (bool, error) {
	files, err := SplitToFiles(text, namesMode)
	if err != nil {
		return false, err
	}
	back, err := JoinFromFiles(files, false, false)
	if err != nil {
		return false, err
	}
	return CanonEqual(text, back)
}
