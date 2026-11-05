package rules

import (
	_ "embed"
	"encoding/xml"
	"sort"
)

//go:embed relationships.xml
var RelXML []byte

//go:embed relationships-keys.xml
var RelKeysXML []byte

// ARCHIMATE namespace prefix.
const ARCHIMATE = "archimate:"

// RelSuffix is appended to all relationship type short names.
const RelSuffix = "Relationship"

// ElementTypes is the set of all ArchiMate element source concept names.
var ElementTypes map[string]bool

// RelTypes is the set of all relationship type strings (e.g. "TriggeringRelationship").
var RelTypes map[string]bool

// rules: (srcType, tgtType) -> set of relationship type strings.
var rules map[[2]string]map[string]bool

func init() {
	// Parse keys file: char -> relationship name.
	type keyEntry struct {
		Char         string `xml:"char,attr"`
		Relationship string `xml:"relationship,attr"`
	}
	type keysRoot struct {
		Keys []keyEntry `xml:"key"`
	}
	var kr keysRoot
	if err := xml.Unmarshal(RelKeysXML, &kr); err != nil {
		panic("rules: parse relationships-keys.xml: " + err.Error())
	}
	charMap := make(map[string]string, len(kr.Keys))
	for _, k := range kr.Keys {
		charMap[k.Char] = k.Relationship
	}

	// Parse main relationships file.
	type targetEntry struct {
		Concept   string `xml:"concept,attr"`
		Relations string `xml:"relations,attr"`
	}
	type sourceEntry struct {
		Concept string        `xml:"concept,attr"`
		Targets []targetEntry `xml:"target"`
	}
	type relsRoot struct {
		Sources []sourceEntry `xml:"source"`
	}
	var rr relsRoot
	if err := xml.Unmarshal(RelXML, &rr); err != nil {
		panic("rules: parse relationships.xml: " + err.Error())
	}

	rules = make(map[[2]string]map[string]bool)
	eltTypes := make(map[string]bool)
	relTypes := make(map[string]bool)

	for _, s := range rr.Sources {
		eltTypes[s.Concept] = true
		for _, t := range s.Targets {
			key := [2]string{s.Concept, t.Concept}
			if rules[key] == nil {
				rules[key] = make(map[string]bool)
			}
			for _, c := range t.Relations {
				if rel, ok := charMap[string(c)]; ok {
					rules[key][rel] = true
					relTypes[rel] = true
				}
			}
		}
	}
	delete(eltTypes, "Relationship") // relation-to-relation entries are not an element type
	ElementTypes = eltTypes
	RelTypes = relTypes
}

// Allowed reports whether relType is allowed from srcType to tgtType.
func Allowed(srcType, tgtType, relType string) bool {
	return rules[[2]string{srcType, tgtType}][relType]
}

// AllowedSet returns the sorted list of allowed relationship types for (srcType, tgtType).
func AllowedSet(srcType, tgtType string) []string {
	set := rules[[2]string{srcType, tgtType}]
	out := make([]string, 0, len(set))
	for r := range set {
		out = append(out, r)
	}
	sort.Strings(out)
	return out
}
