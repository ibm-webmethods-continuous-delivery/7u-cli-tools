// Package ops implements the query and edit commands on a parsed Archi model.
package ops

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/ibm-webmethods-continuous-delivery/7u-cli-tools/code/archictl/internal/archi"
)

const (
	newIDPrefix    = "x"
	nearMatchLimit = 3
	nearMatchCut   = 0.6
	diagramModel   = "ArchimateDiagramModel"
)

// Model is a parsed .archimate file with an id index.
type Model struct {
	Root archi.Obj
	// ByID indexes every element carrying an id. NewID reserves ids with a zero Node.
	ByID map[string]archi.Node
}

// ParseModel parses XML text.
func ParseModel(text string) (*Model, error) {
	root, err := archi.ParseXML(text)
	if err != nil {
		return nil, err
	}
	m := &Model{Root: root}
	m.Refresh()
	return m, nil
}

// LoadModel reads and parses a plain-XML model file.
func LoadModel(path string) (*Model, error) {
	text, err := archi.ReadXML(path)
	if err != nil {
		return nil, err
	}
	return ParseModel(text)
}

// Text serialises the model to XML.
func (m *Model) Text() (string, error) { return archi.SerializeXML(m.Root) }

// Save writes the model atomically to path.
func (m *Model) Save(path string) error {
	text, err := m.Text()
	if err != nil {
		return err
	}
	return WriteFileAtomic(path, []byte(text))
}

// Refresh rebuilds the id index.
func (m *Model) Refresh() {
	m.ByID = map[string]archi.Node{}
	archi.Walk(m.Root, func(n archi.Node) {
		if id := n.ID(); id != "" {
			m.ByID[id] = n
		}
	})
}

// NewID returns the first unused "x<N>", N starting at count(ids)+1, and reserves it.
func (m *Model) NewID() string {
	n := len(m.ByID)
	for {
		n++
		cand := newIDPrefix + strconv.Itoa(n)
		if _, used := m.ByID[cand]; !used {
			m.ByID[cand] = archi.Node{}
			return cand
		}
	}
}

// Nodes lists every element of the tree in document order.
func (m *Model) Nodes() []archi.Node {
	var out []archi.Node
	archi.Walk(m.Root, func(n archi.Node) { out = append(out, n) })
	return out
}

// Concepts lists elements and relationships in document order.
func (m *Model) Concepts() []archi.Node {
	var out []archi.Node
	for _, n := range m.Nodes() {
		if n.IsElement() || n.IsRelationship() {
			out = append(out, n)
		}
	}
	return out
}

// Views lists the ArchimateDiagramModel elements.
func (m *Model) Views() []archi.Node {
	var out []archi.Node
	for _, n := range m.Nodes() {
		if n.Tag() == "element" && n.Type() == diagramModel {
			out = append(out, n)
		}
	}
	return out
}

// Resolve finds an element (or any <element>) by id, else an ArchiMate element by unique name.
func (m *Model) Resolve(ref string) (archi.Node, error) {
	if n, ok := m.ByID[ref]; ok && n.Obj != nil && n.Tag() == "element" {
		return n, nil
	}
	var found []archi.Node
	var names []string
	seen := map[string]bool{}
	for _, n := range m.Nodes() {
		if !n.IsElement() {
			continue
		}
		name, has := n.Obj.StrOK("name")
		if has && name == ref {
			found = append(found, n)
		}
		if name != "" && !seen[name] {
			seen[name] = true
			names = append(names, name)
		}
	}
	switch len(found) {
	case 1:
		return found[0], nil
	case 0:
		msg := "unknown element: " + archi.PyRepr(ref)
		if near := closeMatches(ref, names, nearMatchLimit, nearMatchCut); len(near) > 0 {
			msg += "; did you mean: " + strings.Join(near, ", ") + "?"
		}
		return archi.Node{}, &archi.Refusal{Msg: msg}
	}
	cands := make([]string, len(found))
	for i, n := range found {
		cands[i] = fmt.Sprintf("%s (%s)", n.ID(), n.Type())
	}
	return archi.Node{}, archi.Refusef("ambiguous name %s, use an id: %s", archi.PyRepr(ref), strings.Join(cands, ", "))
}

// ResolveView finds a diagram by id or name.
func (m *Model) ResolveView(ref string) (archi.Node, error) {
	var views []archi.Node
	for _, v := range m.Views() {
		if v.ID() == ref || (v.Obj.Has("name") && v.Obj.Str("name") == ref) {
			views = append(views, v)
		}
	}
	if len(views) != 1 {
		kind := "unknown"
		if len(views) > 0 {
			kind = "ambiguous"
		}
		return archi.Node{}, archi.Refusef("%s view: %s", kind, archi.PyRepr(ref))
	}
	return views[0], nil
}

// Label is "Type Name #id".
func (m *Model) Label(n archi.Node) string {
	return strings.ReplaceAll(fmt.Sprintf("%s %s #%s", n.Type(), n.Obj.Str("name"), n.ID()), "  ", " ")
}

// RelationsOf lists relationships whose source or target is n.
func (m *Model) RelationsOf(n archi.Node) []archi.Node {
	id := n.ID()
	var out []archi.Node
	for _, r := range m.Nodes() {
		if r.IsRelationship() && (r.Obj.Str("source") == id || r.Obj.Str("target") == id) {
			out = append(out, r)
		}
	}
	return out
}

// ViewsOf lists the diagrams that draw n (as element or as relationship).
func (m *Model) ViewsOf(n archi.Node) []archi.Node {
	id := n.ID()
	var out []archi.Node
	for _, v := range m.Views() {
		hit := false
		archi.WalkNode(v, func(x archi.Node) {
			if x.Obj.Str("archimateElement") == id || x.Obj.Str("archimateRelationship") == id {
				hit = true
			}
		})
		if hit {
			out = append(out, v)
		}
	}
	return out
}

// ValidateAll returns reference errors followed by relationship-legality errors.
func (m *Model) ValidateAll() []string { return archi.ValidateObj(m.Root, true) }

// NewErrors returns errors of now that are not in before, preserving order.
func NewErrors(before map[string]bool, now []string) []string {
	var out []string
	for _, e := range now {
		if !before[e] {
			out = append(out, e)
		}
	}
	return out
}

// ErrorSet turns a list into a set.
func ErrorSet(errs []string) map[string]bool {
	s := make(map[string]bool, len(errs))
	for _, e := range errs {
		s[e] = true
	}
	return s
}
