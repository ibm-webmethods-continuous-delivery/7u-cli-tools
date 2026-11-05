package archi

import (
	"fmt"
	"strings"

	"github.com/ibm-webmethods-continuous-delivery/7u-cli-tools/code/archictl/internal/rules"
)

// Refusal marks a domain-level refusal (validation failure, ambiguous/unknown reference, ...).
// The command layer maps it to exit code 1.
type Refusal struct{ Msg string }

func (r *Refusal) Error() string { return r.Msg }

// Refusef builds a *Refusal.
func Refusef(format string, a ...any) error { return &Refusal{fmt.Sprintf(format, a...)} }

// Node is one XML element represented inside the object tree.
type Node struct {
	Key    string
	Obj    Obj
	Parent Obj // the Obj owning the kids list; nil for the root
}

// Tag is the XML tag of the element.
func (n Node) Tag() string {
	if IsTypeKey(n.Key) {
		if t := n.Obj.Str("_tag"); t != "" {
			return t
		}
		return InferTag(n.Key)
	}
	return n.Key
}

// Type is the short xsi:type ("" when the element has none).
func (n Node) Type() string {
	if IsTypeKey(n.Key) {
		return ShortType(n.Key)
	}
	return ""
}

// ID is the id attribute ("" if none).
func (n Node) ID() string { return n.Obj.Str("id") }

// Kids returns the child elements held in the "kids" list.
func (n Node) Kids() []Node {
	var out []Node
	for _, kid := range n.Obj.List("kids") {
		ko, ok := kid.(Obj)
		if !ok {
			continue
		}
		k, v, one := ko.Single()
		vo, isObj := v.(Obj)
		if one && isObj {
			out = append(out, Node{k, vo, n.Obj})
		}
	}
	return out
}

// RootNode returns the root element node.
func RootNode(root Obj) (Node, bool) {
	k, v, ok := root.Single()
	vo, isObj := v.(Obj)
	if !ok || !isObj {
		return Node{}, false
	}
	return Node{k, vo, nil}, true
}

// Walk visits every element in document order.
func Walk(root Obj, fn func(Node)) {
	if r, ok := RootNode(root); ok {
		walkNode(r, fn)
	}
}

func walkNode(n Node, fn func(Node)) {
	fn(n)
	for _, k := range n.Kids() {
		walkNode(k, fn)
	}
}

// WalkNode visits n and all its descendants in document order.
func WalkNode(n Node, fn func(Node)) { walkNode(n, fn) }

// AppendKid appends child (a single-key Obj) to the "kids" list of o.
func AppendKid(o Obj, key string, val Obj) {
	o.Set("kids", append(o.List("kids"), O(key, val)))
}

// RemoveKid removes the kid whose value is child from o's "kids" list; false when absent.
func RemoveKid(o Obj, child Obj) bool {
	kids := o.List("kids")
	for i, kid := range kids {
		if ko, ok := kid.(Obj); ok {
			if _, v, one := ko.Single(); one && v == any(child) {
				nl := make([]any, 0, len(kids)-1)
				nl = append(nl, kids[:i]...)
				nl = append(nl, kids[i+1:]...)
				o.Set("kids", nl)
				return true
			}
		}
	}
	return false
}

// IsRelationship: an <element> whose type ends in "Relationship".
func (n Node) IsRelationship() bool {
	return n.Tag() == "element" && strings.HasSuffix(n.Type(), rules.RelSuffix)
}

// IsElement: an <element> whose type is a known ArchiMate element type.
func (n Node) IsElement() bool {
	return n.Tag() == "element" && rules.ElementTypes[n.Type()]
}

var refAttrs = []string{"source", "target", "archimateElement", "archimateRelationship", "model"}

func idOrNone(n Node) string {
	if id := n.ID(); id != "" {
		return id
	}
	return "None"
}

// CheckRefs validates id references and id uniqueness.
func CheckRefs(root Obj) []string {
	var nodes []Node
	Walk(root, func(n Node) { nodes = append(nodes, n) })
	count := map[string]int{}
	var order []string
	for _, n := range nodes {
		if id := n.ID(); id != "" {
			if count[id] == 0 {
				order = append(order, id)
			}
			count[id]++
		}
	}
	var errs []string
	for _, n := range nodes {
		for _, a := range refAttrs {
			if v := n.Obj.Str(a); v != "" && count[v] == 0 {
				errs = append(errs, fmt.Sprintf("%s[%s] %s=%s not found", n.Tag(), idOrNone(n), a, v))
			}
		}
		for _, t := range strings.Fields(n.Obj.Str("targetConnections")) {
			if count[t] == 0 {
				errs = append(errs, fmt.Sprintf("%s[%s] targetConnections=%s not found", n.Tag(), idOrNone(n), t))
			}
		}
	}
	for _, id := range order {
		if count[id] > 1 {
			errs = append(errs, "duplicate id "+id)
		}
	}
	return errs
}

func nameOrID(n Node) string {
	if s := n.Obj.Str("name"); s != "" {
		return s
	}
	return n.ID()
}

// ValidateRelationships checks every relationship between two elements against Archi's tables.
func ValidateRelationships(root Obj) []string {
	byID := map[string]Node{}
	var rels []Node
	Walk(root, func(n Node) {
		if n.Tag() == "element" {
			byID[n.ID()] = n
			if n.IsRelationship() {
				rels = append(rels, n)
			}
		}
	})
	var errs []string
	for _, r := range rels {
		s, okS := byID[r.Obj.Str("source")]
		t, okT := byID[r.Obj.Str("target")]
		if !okS || !okT || s.IsRelationship() || t.IsRelationship() {
			continue // dangling refs are reported elsewhere; relation-to-relation is not in the table
		}
		rt := r.Type()
		if rules.Allowed(s.Type(), t.Type(), rt) {
			continue
		}
		hint := "none"
		if set := rules.AllowedSet(s.Type(), t.Type()); len(set) > 0 {
			short := make([]string, len(set))
			for i, a := range set {
				short[i] = strings.TrimSuffix(a, rules.RelSuffix)
			}
			hint = strings.Join(short, ", ")
		}
		errs = append(errs, fmt.Sprintf("%s %s (%s) > %s (%s) [%s] not allowed; allowed: %s",
			strings.TrimSuffix(rt, rules.RelSuffix), nameOrID(s), s.Type(), nameOrID(t), t.Type(), r.ID(), hint))
	}
	return errs
}
