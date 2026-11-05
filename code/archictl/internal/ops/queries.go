package ops

import (
	"fmt"
	"strings"

	"github.com/ibm-webmethods-continuous-delivery/7u-cli-tools/code/archictl/internal/archi"
	"github.com/ibm-webmethods-continuous-delivery/7u-cli-tools/code/archictl/internal/rules"
)

// Results are ordered Objs (not map[string]any): the --json output must keep the oracle's key order.

func (m *Model) nameOrIDOf(id string) string {
	if n, ok := m.ByID[id]; ok && n.Obj != nil {
		if s := n.Obj.Str("name"); s != "" {
			return s
		}
	}
	return id
}

func (m *Model) ends(r archi.Node) (string, string) {
	return m.nameOrIDOf(r.Obj.Str("source")), m.nameOrIDOf(r.Obj.Str("target"))
}

// Find matches concepts by case-insensitive name substring or exact id, optionally by type.
func Find(m *Model, query string, typFilter string) []archi.Obj {
	q := strings.ToLower(query)
	out := []archi.Obj{}
	for _, e := range m.Concepts() {
		if typFilter != "" && e.Type() != typFilter && e.Type() != typFilter+rules.RelSuffix {
			continue
		}
		if !strings.Contains(strings.ToLower(e.Obj.Str("name")), q) && q != e.ID() {
			continue
		}
		item := archi.O("type", e.Type(), "name", e.Obj.Str("name"), "id", e.ID())
		if e.IsRelationship() {
			s, t := m.ends(e)
			item.Set("source", s)
			item.Set("target", t)
		}
		out = append(out, item)
	}
	return out
}

// FormatFind renders Find results as lines.
func FormatFind(items []archi.Obj) string {
	lines := make([]string, len(items))
	for i, it := range items {
		extra := ""
		if it.Has("source") {
			extra = fmt.Sprintf(" (%s > %s)", it.Str("source"), it.Str("target"))
		}
		lines[i] = fmt.Sprintf("%s %s%s #%s", it.Str("type"), it.Str("name"), extra, it.Str("id"))
	}
	return strings.Join(lines, "\n")
}

// Show describes one element: type, name, id, doc, props, relations in/out and views.
func Show(m *Model, ref string) (archi.Obj, error) {
	e, err := m.Resolve(ref)
	if err != nil {
		return nil, err
	}
	rels := []any{}
	for _, r := range m.RelationsOf(e) {
		s, t := m.ends(r)
		out := r.Obj.Str("source") == e.ID()
		dir, other := "in", s
		if out {
			dir, other = "out", t
		}
		rels = append(rels, archi.O("type", strings.TrimSuffix(r.Type(), rules.RelSuffix), "direction", dir,
			"other", other, "name", r.Obj.Str("name"), "id", r.ID()))
	}
	props := archi.NewObj()
	for _, p := range e.Obj.List("props") {
		if po, ok := p.(archi.Obj); ok {
			if k, v, one := po.Single(); one {
				props.Set(k, v)
			}
		}
	}
	views := []any{}
	for _, v := range m.ViewsOf(e) {
		name := v.Obj.Str("name")
		if name == "" {
			name = v.ID()
		}
		views = append(views, name)
	}
	var doc any
	if d, ok := e.Obj.StrOK("doc"); ok {
		doc = d
	}
	return archi.O("type", e.Type(), "name", e.Obj.Str("name"), "id", e.ID(), "doc", doc,
		"props", props, "relations", rels, "views", views), nil
}

// FormatShow renders Show output as text.
func FormatShow(m *Model, d archi.Obj) string {
	var b strings.Builder
	label := strings.ReplaceAll(fmt.Sprintf("%s %s #%s", d.Str("type"), d.Str("name"), d.Str("id")), "  ", " ")
	b.WriteString(label)
	if doc, _ := d.Get("doc").(string); doc != "" {
		b.WriteString("\ndoc: " + doc)
	}
	props := d.Obj("props")
	for _, k := range props.Keys() {
		b.WriteString(fmt.Sprintf("\nprop: %s=%s", k, props.Str(k)))
	}
	for _, r := range d.List("relations") {
		ro := r.(archi.Obj)
		arrow := "< "
		if ro.Str("direction") == "out" {
			arrow = "> "
		}
		line := fmt.Sprintf("\n  %s %s%s", ro.Str("type"), arrow, ro.Str("other"))
		if n := ro.Str("name"); n != "" {
			line += " | " + n
		}
		b.WriteString(line + " #" + ro.Str("id"))
	}
	if views := d.List("views"); len(views) > 0 {
		names := make([]string, len(views))
		for i, v := range views {
			names[i] = v.(string)
		}
		b.WriteString("\nviews: " + strings.Join(names, ", "))
	}
	return b.String()
}

// Orphans lists concepts that appear in no view, split into elements and relationships.
func Orphans(m *Model) archi.Obj {
	used := map[string]bool{}
	for _, n := range m.Nodes() {
		switch n.Tag() {
		case "child":
			if s := n.Obj.Str("archimateElement"); s != "" {
				used[s] = true
			}
		case "sourceConnection":
			if s := n.Obj.Str("archimateRelationship"); s != "" {
				used[s] = true
			}
		}
	}
	elems, rels := []archi.Obj{}, []archi.Obj{}
	for _, e := range m.Concepts() {
		if used[e.ID()] {
			continue
		}
		item := archi.O("type", e.Type(), "name", e.Obj.Str("name"), "id", e.ID())
		if strings.HasSuffix(e.Type(), rules.RelSuffix) {
			rels = append(rels, item)
		} else {
			elems = append(elems, item)
		}
	}
	return archi.O("elements", elems, "relationships", rels)
}

// FormatOrphans renders Orphans output as text.
func FormatOrphans(d archi.Obj) string {
	var lines []string
	elems := d.Get("elements").([]archi.Obj)
	rels := d.Get("relationships").([]archi.Obj)
	for _, i := range elems {
		lines = append(lines, fmt.Sprintf("%s %s #%s", i.Str("type"), i.Str("name"), i.Str("id")))
	}
	lines = append(lines, fmt.Sprintf("%d element(s) and %d relationship(s) in no view", len(elems), len(rels)))
	return strings.Join(lines, "\n")
}

// Unused lists diagrams that contain no model elements.
func Unused(m *Model) []archi.Obj {
	out := []archi.Obj{}
	for _, v := range m.Views() {
		has := false
		archi.WalkNode(v, func(n archi.Node) {
			if n.Tag() == "child" && n.Obj.Str("archimateElement") != "" {
				has = true
			}
		})
		if !has {
			out = append(out, archi.O("name", v.Obj.Str("name"), "id", v.ID()))
		}
	}
	return out
}

// FormatUnused renders Unused output as text.
func FormatUnused(items []archi.Obj) string {
	var lines []string
	for _, v := range items {
		lines = append(lines, fmt.Sprintf("view %s #%s", v.Str("name"), v.Str("id")))
	}
	lines = append(lines, fmt.Sprintf("%d empty view(s)", len(items)))
	return strings.Join(lines, "\n")
}
