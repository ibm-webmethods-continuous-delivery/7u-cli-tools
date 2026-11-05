package rules

import (
	"reflect"
	"testing"
)

func TestAllowed(t *testing.T) {
	cases := []struct {
		src, tgt, rel string
		want          bool
	}{
		{"BusinessProcess", "BusinessProcess", "TriggeringRelationship", true},
		{"BusinessProcess", "BusinessRole", "CompositionRelationship", false},
		{"BusinessProcess", "BusinessRole", "ServingRelationship", true},
		{"NoSuchType", "BusinessRole", "ServingRelationship", false},
	}
	for _, c := range cases {
		if got := Allowed(c.src, c.tgt, c.rel); got != c.want {
			t.Errorf("Allowed(%s,%s,%s) = %v, want %v", c.src, c.tgt, c.rel, got, c.want)
		}
	}
}

func TestAllowedSet(t *testing.T) {
	want := []string{"AssociationRelationship", "FlowRelationship", "ServingRelationship", "TriggeringRelationship"}
	if got := AllowedSet("BusinessProcess", "BusinessRole"); !reflect.DeepEqual(got, want) {
		t.Errorf("AllowedSet = %v, want %v", got, want)
	}
	if got := AllowedSet("Nope", "Nope"); len(got) != 0 {
		t.Errorf("AllowedSet(unknown) = %v, want empty", got)
	}
}

func TestTypeSets(t *testing.T) {
	if !ElementTypes["BusinessActor"] || ElementTypes["Relationship"] {
		t.Error("ElementTypes content is wrong")
	}
	if len(RelTypes) != 11 || !RelTypes["FlowRelationship"] {
		t.Errorf("RelTypes = %v", RelTypes)
	}
}
