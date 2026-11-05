package archi

import (
	"strings"
	"testing"
)

const sample = `<?xml version="1.0" encoding="UTF-8"?>
<archimate:model xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:archimate="http://www.archimatetool.com/archimate" name="S" id="m1" version="5.0.0">
  <folder name="Business" id="f1" type="business">
    <element xsi:type="archimate:BusinessActor" name="A &amp; B" id="a1"><documentation>line 1&#10;line 2</documentation><property key="k" value="v"/></element>
  </folder>
  <folder name="Views" id="f2" type="diagrams">
    <element xsi:type="archimate:ArchimateDiagramModel" name="V" id="v1">
      <child xsi:type="archimate:DiagramObject" id="d1" archimateElement="a1">
        <bounds x="10" y="20" width="120" height="55"/>
      </child>
    </element>
  </folder>
</archimate:model>`

func TestParseSerializeRoundTrip(t *testing.T) {
	obj, err := ParseXML(sample)
	if err != nil {
		t.Fatal(err)
	}
	out, err := SerializeXML(obj)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(out, "<?xml version='1.0' encoding='UTF-8'?>\n<archimate:model xmlns:xsi=") {
		t.Errorf("unexpected header: %.100q", out)
	}
	for _, want := range []string{
		`name="A &amp; B"`,
		`<documentation>line 1` + "\n" + `line 2</documentation>`,
		`<property key="k" value="v"/>`,
		`<bounds x="10" y="20" width="120" height="55"/>`,
		`xsi:type="archimate:DiagramObject"`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output lacks %q:\n%s", want, out)
		}
	}
	ok, err := CanonEqual(sample, out)
	if err != nil || !ok {
		t.Errorf("round trip changed the canonical form (err=%v)", err)
	}
}

func TestParseFoldsChildren(t *testing.T) {
	obj, err := ParseXML(sample)
	if err != nil {
		t.Fatal(err)
	}
	var actor Obj
	Walk(obj, func(n Node) {
		if n.Key == "BusinessActor" {
			actor = n.Obj
		}
	})
	if actor == nil {
		t.Fatal("BusinessActor not found")
	}
	if actor.Str("doc") != "line 1\nline 2" {
		t.Errorf("doc = %q", actor.Str("doc"))
	}
	if len(actor.List("props")) != 1 || actor.Has("kids") {
		t.Errorf("props/kids not folded: %v", actor.Keys())
	}
	var node Obj
	Walk(obj, func(n Node) {
		if n.Key == "DiagramObject" {
			node = n.Obj
		}
	})
	if node.Str("b") != "10 20 120 55" {
		t.Errorf("b = %q", node.Str("b"))
	}
}

func TestParseRejects(t *testing.T) {
	cases := map[string]string{
		"comment":  `<?xml version="1.0"?><a><!-- c --></a>`,
		"doctype":  `<?xml version="1.0"?><!DOCTYPE a [<!ENTITY e "x">]><a>&e;</a>`,
		"reserved": `<?xml version="1.0"?><a doc="x"/>`,
	}
	for name, text := range cases {
		if _, err := ParseXML(text); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}

func TestInferTag(t *testing.T) {
	for k, want := range map[string]string{
		"Connection": "sourceConnection", "DiagramObject": "child", "Group": "child",
		"BusinessActor": "element", "archimate:Note": "child",
	} {
		if got := InferTag(k); got != want {
			t.Errorf("InferTag(%q) = %q, want %q", k, got, want)
		}
	}
}

func TestYAMLKeepsStringsAndOrder(t *testing.T) {
	obj := O("id", "007", "name", "null", "flag", "true", "doc", "a\nb")
	txt, err := DumpYAML(obj)
	if err != nil {
		t.Fatal(err)
	}
	back, err := LoadYAML(txt)
	if err != nil {
		t.Fatal(err)
	}
	bo := back.(Obj)
	if strings.Join(bo.Keys(), ",") != "id,name,flag,doc" {
		t.Errorf("key order lost: %v", bo.Keys())
	}
	for _, k := range obj.Keys() {
		if bo.Str(k) != obj.Str(k) {
			t.Errorf("%s: got %q want %q", k, bo.Str(k), obj.Str(k))
		}
	}
}
