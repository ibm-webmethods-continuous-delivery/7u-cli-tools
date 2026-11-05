// Package validate checks a parsed model object: id references and relationship legality.
package validate

import "github.com/ibm-webmethods-continuous-delivery/7u-cli-tools/code/archictl/internal/archi"

// CheckRefs validates id references and uniqueness.
func CheckRefs(obj archi.Obj) []string { return archi.CheckRefs(obj) }

// Rules validates relationship legality against Archi's tables.
func Rules(obj archi.Obj) []string { return archi.ValidateRelationships(obj) }

// ValidateAll returns reference errors followed by legality errors.
func ValidateAll(obj archi.Obj) []string { return archi.ValidateObj(obj, true) }
