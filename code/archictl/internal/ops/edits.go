package ops

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/ibm-webmethods-continuous-delivery/7u-cli-tools/code/archictl/internal/archi"
	"github.com/ibm-webmethods-continuous-delivery/7u-cli-tools/code/archictl/internal/rules"
)

// FolderOf maps an element type to the type of the top-level folder that holds it.
var FolderOf = func() map[string]string {
	groups := []struct{ folder, types string }{
		{"business", "BusinessActor BusinessRole BusinessCollaboration BusinessInterface BusinessProcess " +
			"BusinessFunction BusinessInteraction BusinessEvent BusinessService BusinessObject " +
			"Contract Representation Product"},
		{"application", "ApplicationComponent ApplicationCollaboration ApplicationInterface ApplicationFunction " +
			"ApplicationInteraction ApplicationProcess ApplicationEvent ApplicationService DataObject"},
		{"technology", "Node Device SystemSoftware TechnologyCollaboration TechnologyInterface Path " +
			"CommunicationNetwork TechnologyFunction TechnologyProcess TechnologyInteraction " +
			"TechnologyEvent TechnologyService Artifact Equipment Facility DistributionNetwork Material"},
		{"strategy", "Resource Capability ValueStream CourseOfAction"},
		{"motivation", "Stakeholder Driver Assessment Goal Outcome Principle Requirement Constraint Meaning Value"},
		{"implementation_migration", "WorkPackage Deliverable ImplementationEvent Plateau Gap"},
		{"other", "Location Grouping Junction"},
	}
	m := map[string]string{}
	for _, g := range groups {
		for _, t := range strings.Fields(g.types) {
			m[t] = g.folder
		}
	}
	return m
}()

func (m *Model) topFolder(ftype string) (archi.Obj, error) {
	root, _ := archi.RootNode(m.Root)
	for _, k := range root.Kids() {
		if k.Key == "folder" && k.Obj.Str("type") == ftype {
			return k.Obj, nil
		}
	}
	return nil, archi.Refusef("no top-level folder of type %s", ftype)
}

func (m *Model) subfolder(top archi.Obj, path string) archi.Obj {
	cur := top
	for _, name := range strings.Split(path, "/") {
		if name == "" {
			continue
		}
		var next archi.Obj
		nFolders := 0
		for _, kid := range cur.List("kids") {
			ko, ok := kid.(archi.Obj)
			if !ok {
				continue
			}
			if k, v, one := ko.Single(); one && k == "folder" {
				nFolders++
				if vo, ok := v.(archi.Obj); ok && next == nil && vo.Str("name") == name {
					next = vo
				}
			}
		}
		if next == nil {
			next = archi.O("name", name, "id", m.NewID())
			// folders precede elements, as Archi writes them
			kids := cur.List("kids")
			pos := 0
			for seen := 0; pos < len(kids) && seen < nFolders; pos++ {
				if ko, ok := kids[pos].(archi.Obj); ok {
					if k, _, one := ko.Single(); one && k == "folder" {
						seen++
					}
				}
			}
			nk := make([]any, 0, len(kids)+1)
			nk = append(nk, kids[:pos]...)
			nk = append(nk, archi.O("folder", next))
			nk = append(nk, kids[pos:]...)
			cur.Set("kids", nk)
		}
		cur = next
	}
	return cur
}

func (m *Model) register(n archi.Node) { m.ByID[n.ID()] = n }

func sortedSet(set map[string]bool, trim string) []string {
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, strings.TrimSuffix(k, trim))
	}
	sort.Strings(out)
	return out
}

// Add creates an element in the folder of its type (optionally in a sub-folder path).
func Add(m *Model, typ, name, folder, doc string) (archi.Obj, error) {
	if !rules.ElementTypes[typ] {
		return nil, archi.Refusef("unknown element type %s; valid: %s", archi.PyRepr(typ),
			strings.Join(sortedSet(rules.ElementTypes, ""), " "))
	}
	top, err := m.topFolder(FolderOf[typ])
	if err != nil {
		return nil, err
	}
	parent := top
	if folder != "" {
		parent = m.subfolder(top, folder)
	}
	e := archi.O("name", name, "id", m.NewID())
	if doc != "" {
		e.Set("doc", doc)
	}
	archi.AppendKid(parent, typ, e)
	m.register(archi.Node{Key: typ, Obj: e, Parent: parent})
	return e, nil
}

// Connect adds a relationship after checking Archi's legality tables.
func Connect(m *Model, relType, src, tgt, name string) (archi.Obj, error) {
	rtype := relType
	if !strings.HasSuffix(rtype, rules.RelSuffix) {
		rtype += rules.RelSuffix
	}
	if !rules.RelTypes[rtype] {
		return nil, archi.Refusef("unknown relationship type %s; valid: %s", archi.PyRepr(relType),
			strings.Join(sortedSet(rules.RelTypes, rules.RelSuffix), " "))
	}
	s, err := m.Resolve(src)
	if err != nil {
		return nil, err
	}
	t, err := m.Resolve(tgt)
	if err != nil {
		return nil, err
	}
	if !rules.Allowed(s.Type(), t.Type(), rtype) {
		hint := "none"
		if set := rules.AllowedSet(s.Type(), t.Type()); len(set) > 0 {
			short := make([]string, len(set))
			for i, a := range set {
				short[i] = strings.TrimSuffix(a, rules.RelSuffix)
			}
			hint = strings.Join(short, ", ")
		}
		return nil, archi.Refusef("%s not allowed from %s to %s; allowed: %s", relType, s.Type(), t.Type(), hint)
	}
	r := archi.NewObj()
	if name != "" {
		r.Set("name", name)
	}
	r.Set("id", m.NewID())
	r.Set("source", s.ID())
	r.Set("target", t.ID())
	top, err := m.topFolder("relations")
	if err != nil {
		return nil, err
	}
	archi.AppendKid(top, rtype, r)
	m.register(archi.Node{Key: rtype, Obj: r, Parent: top})
	return r, nil
}

// Rename changes an element's name and returns it with the previous name.
func Rename(m *Model, ref, newName string) (archi.Obj, string, error) {
	e, err := m.Resolve(ref)
	if err != nil {
		return nil, "", err
	}
	old := e.Obj.Str("name")
	e.Obj.Set("name", newName)
	return e.Obj, old, nil
}

// Delete removes an element, the relationships touching it (transitively) and the view items
// that reference them. Returns the element, the number of relationships and of view items removed.
func Delete(m *Model, ref string) (archi.Obj, int, int, error) {
	e, err := m.Resolve(ref)
	if err != nil {
		return nil, 0, 0, err
	}
	doomed := map[string]bool{e.ID(): true}
	for changed := true; changed; {
		changed = false
		for _, r := range m.Nodes() {
			if r.IsRelationship() && !doomed[r.ID()] &&
				(doomed[r.Obj.Str("source")] || doomed[r.Obj.Str("target")]) {
				doomed[r.ID()] = true
				changed = true
			}
		}
	}
	removed := map[string]bool{}
	var prune func(owner archi.Obj)
	prune = func(owner archi.Obj) {
		for _, kid := range owner.List("kids") {
			ko, ok := kid.(archi.Obj)
			if !ok {
				continue
			}
			k, v, one := ko.Single()
			vo, isObj := v.(archi.Obj)
			if !one || !isObj {
				continue
			}
			n := archi.Node{Key: k, Obj: vo, Parent: owner}
			if tag := n.Tag(); (tag == "child" || tag == "sourceConnection") &&
				(doomed[vo.Str("archimateElement")] || doomed[vo.Str("archimateRelationship")]) {
				archi.WalkNode(n, func(x archi.Node) {
					if id := x.ID(); id != "" {
						removed[id] = true
					}
				})
				archi.RemoveKid(owner, vo)
				continue
			}
			prune(vo)
		}
	}
	if root, ok := archi.RootNode(m.Root); ok {
		prune(root.Obj)
	}
	// connections hanging off removed nodes
	var orphans []archi.Node
	for _, n := range m.Nodes() {
		if n.Tag() == "sourceConnection" && (removed[n.Obj.Str("target")] || removed[n.Obj.Str("source")]) {
			orphans = append(orphans, n)
		}
	}
	for _, n := range orphans {
		removed[n.ID()] = true
		if n.Parent != nil {
			archi.RemoveKid(n.Parent, n.Obj)
		}
	}
	for _, n := range m.Nodes() {
		tc := n.Obj.Str("targetConnections")
		if tc == "" {
			continue
		}
		var keep []string
		for _, x := range strings.Fields(tc) {
			if !removed[x] {
				keep = append(keep, x)
			}
		}
		if len(keep) > 0 {
			n.Obj.Set("targetConnections", strings.Join(keep, " "))
		} else {
			n.Obj.Del("targetConnections")
		}
	}
	var gone []archi.Node
	done := map[string]bool{}
	for _, n := range m.Nodes() {
		if n.Tag() == "element" && doomed[n.ID()] && !done[n.ID()] {
			done[n.ID()] = true
			gone = append(gone, n)
		}
	}
	for _, n := range gone {
		if n.Parent != nil {
			archi.RemoveKid(n.Parent, n.Obj)
		}
	}
	m.Refresh()
	return e.Obj, len(doomed) - 1, len(removed), nil
}

func atoiDefault(s string, def int) (int, bool) {
	if s == "" || s == "_" {
		return def, true
	}
	v, err := strconv.Atoi(s)
	return v, err == nil
}

// bottomOf returns max(y+height) over all bounds in the view (0 when none).
func bottomOf(view archi.Node) int {
	best := 0
	consider := func(y, h int) {
		if y+h > best {
			best = y + h
		}
	}
	archi.WalkNode(view, func(n archi.Node) {
		if n.Key == "bounds" {
			y, ok1 := atoiDefault(n.Obj.Str("y"), 0)
			h, ok2 := atoiDefault(n.Obj.Str("height"), 55)
			if ok1 && ok2 {
				consider(y, h)
			}
			return
		}
		if b, ok := n.Obj.StrOK("b"); ok {
			f := strings.Fields(b)
			for len(f) < 4 {
				f = append(f, "_")
			}
			y, ok1 := atoiDefault(f[1], 0)
			h, ok2 := atoiDefault(f[3], 55)
			if ok1 && ok2 {
				consider(y, h)
			}
		}
	})
	return best
}

// AddToView places an element on a view and draws its existing relationships to elements already there.
// Returns the view, the element, the new diagram node and the number of connections drawn.
func AddToView(m *Model, viewRef, ref string, x, y *int) (archi.Obj, archi.Obj, archi.Obj, int, error) {
	view, err := m.ResolveView(viewRef)
	if err != nil {
		return nil, nil, nil, 0, err
	}
	e, err := m.Resolve(ref)
	if err != nil {
		return nil, nil, nil, 0, err
	}
	if e.IsRelationship() {
		return nil, nil, nil, 0, archi.Refusef("add elements, relationships are connected automatically")
	}
	nodes := map[string]archi.Node{}
	drawn := map[string]bool{}
	archi.WalkNode(view, func(n archi.Node) {
		switch n.Tag() {
		case "child":
			if a := n.Obj.Str("archimateElement"); a != "" {
				nodes[a] = n
			}
		case "sourceConnection":
			drawn[n.Obj.Str("archimateRelationship")] = true
		}
	})
	if _, dup := nodes[e.ID()]; dup {
		return nil, nil, nil, 0, archi.Refusef("%s is already in view %s", archi.PyRepr(e.Obj.Str("name")),
			archi.PyRepr(view.Obj.Str("name")))
	}
	px, py := 12, 12
	if x != nil && y != nil {
		px, py = *x, *y
	} else if bottom := bottomOf(view); bottom != 0 {
		py = bottom + 24
	}
	bounds := fmt.Sprintf("%d %d %d %d", px, py, 120, 55)
	nodeObj := archi.O("id", m.NewID(), "archimateElement", e.ID(), "b", bounds)
	archi.AppendKid(view.Obj, "DiagramObject", nodeObj)
	node := archi.Node{Key: "DiagramObject", Obj: nodeObj, Parent: view.Obj}
	m.register(node)

	nConn := 0
	for _, r := range m.RelationsOf(e) {
		if drawn[r.ID()] {
			continue
		}
		s, hasS := nodes[r.Obj.Str("source")]
		t, hasT := nodes[r.Obj.Str("target")]
		if r.Obj.Str("source") == e.ID() {
			s, hasS = node, true
		}
		if r.Obj.Str("target") == e.ID() {
			t, hasT = node, true
		}
		if !hasS || !hasT {
			continue
		}
		cid := m.NewID()
		conn := archi.O("id", cid, "source", s.ID(), "target", t.ID(), "archimateRelationship", r.ID())
		archi.AppendKid(s.Obj, "Connection", conn)
		m.register(archi.Node{Key: "Connection", Obj: conn, Parent: s.Obj})
		tc := strings.Fields(t.Obj.Str("targetConnections"))
		t.Obj.Set("targetConnections", strings.Join(append(tc, cid), " "))
		nConn++
	}
	return view.Obj, e.Obj, nodeObj, nConn, nil
}

// RunEdit applies an edit under the advisory lock, refuses it when it introduces new validation
// errors, and otherwise writes the model atomically to out (or in place when out is empty).
// apply returns the message to print.
func RunEdit(modelPath, out string, apply func(m *Model) (string, error)) (string, error) {
	unlock, err := Lock(modelPath)
	if err != nil {
		return "", err
	}
	defer unlock()
	m, err := LoadModel(modelPath)
	if err != nil {
		return "", err
	}
	before := ErrorSet(m.ValidateAll())
	msg, err := apply(m)
	if err != nil {
		return "", err
	}
	if fresh := NewErrors(before, m.ValidateAll()); len(fresh) > 0 {
		if len(fresh) > 10 {
			fresh = fresh[:10]
		}
		return "", archi.Refusef("edit would make the model invalid:\n  %s", strings.Join(fresh, "\n  "))
	}
	if out == "" {
		out = modelPath
	}
	if err := m.Save(out); err != nil {
		return "", err
	}
	return msg, nil
}
