package validate_test

import (
	"strings"
	"testing"

	"github.com/ibm-webmethods-continuous-delivery/7u-cli-tools/code/archictl/internal/archi"
	"github.com/ibm-webmethods-continuous-delivery/7u-cli-tools/code/archictl/internal/validate"
)

const model = `<?xml version="1.0" encoding="UTF-8"?>
<archimate:model xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:archimate="http://www.archimatetool.com/archimate" name="M" id="m1">
  <folder name="Business" id="f1" type="business">
    <element xsi:type="archimate:BusinessProcess" name="P" id="p1"/>
    <element xsi:type="archimate:BusinessRole" name="R" id="r1"/>
    <element xsi:type="archimate:CompositionRelationship" id="c1" source="p1" target="r1"/>
    <element xsi:type="archimate:ServingRelationship" id="s1" source="p1" target="gone"/>
    <element xsi:type="archimate:BusinessRole" name="Dup" id="s1"/>
  </folder>
</archimate:model>`

func TestValidate(t *testing.T) {
	obj, err := archi.ParseXML(model)
	if err != nil {
		t.Fatal(err)
	}
	refs := validate.CheckRefs(obj)
	if len(refs) != 2 || !strings.Contains(refs[0], "target=gone not found") || refs[1] != "duplicate id s1" {
		t.Errorf("CheckRefs = %q", refs)
	}
	rules := validate.Rules(obj)
	want := "Composition P (BusinessProcess) > R (BusinessRole) [c1] not allowed; allowed: Association, Flow, Serving, Triggering"
	if len(rules) != 1 || rules[0] != want {
		t.Errorf("Rules = %q", rules)
	}
	if all := validate.ValidateAll(obj); len(all) != 3 {
		t.Errorf("ValidateAll returned %d errors, want 3: %q", len(all), all)
	}
}
