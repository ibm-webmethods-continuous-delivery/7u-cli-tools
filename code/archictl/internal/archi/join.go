package archi

import (
	"errors"
	"sort"
	"strings"
)

// JoinToObj rebuilds the full model object from the YAML files.
func JoinToObj(files map[string]string) (Obj, error) {
	modelText, ok := files["model.yaml"]
	if !ok {
		return nil, Refusef("model.yaml not found")
	}
	names := make([]string, 0, len(files))
	for n := range files {
		if n != "model.yaml" {
			names = append(names, n)
		}
	}
	sort.Strings(names)
	views := map[string][]any{}
	extra := map[string]bool{}
	for _, n := range names {
		v, err := LoadYAML(files[n])
		if err != nil {
			return nil, Refusef("%s: %v", n, err)
		}
		list, ok := v.([]any)
		if !ok {
			return nil, Refusef("%s: a view file must hold a list of nodes", n)
		}
		views[n] = list
		CollectIDs(list, extra)
	}
	mv, err := LoadYAML(modelText)
	if err != nil {
		return nil, Refusef("model.yaml: %v", err)
	}
	model, ok := mv.(Obj)
	if !ok {
		return nil, Refusef("model.yaml: top level must be a mapping")
	}
	obj := TreeExpand(model, extra)
	for _, view := range findViews(obj) {
		if !view.Has("file") {
			continue
		}
		fn := view.Str("file")
		kids, ok := views[fn]
		if !ok {
			return nil, Refusef("view file not found: %s", fn)
		}
		view.Del("file")
		view.Set("kids", kids)
	}
	if _, err := ExpandRelations(obj); err != nil {
		return nil, &Refusal{err.Error()}
	}
	if _, err := ExpandViews(obj); err != nil {
		return nil, &Refusal{err.Error()}
	}
	return obj, nil
}

// ValidateObj returns reference errors plus (optionally) relationship-legality errors.
func ValidateObj(obj Obj, enforceRules bool) []string {
	errs := CheckRefs(obj)
	if enforceRules {
		errs = append(errs, ValidateRelationships(obj)...)
	}
	return errs
}

// JoinFromFiles rebuilds XML text from the YAML files. With validate, reference errors are
// refused; enforceRules additionally refuses illegal relationships.
func JoinFromFiles(files map[string]string, validate bool, enforceRules bool) (string, error) {
	obj, err := JoinToObj(files)
	if err != nil {
		return "", err
	}
	if validate {
		if errs := ValidateObj(obj, enforceRules); len(errs) > 0 {
			if len(errs) > 20 {
				errs = errs[:20]
			}
			return "", &Refusal{"invalid model:\n  " + strings.Join(errs, "\n  ")}
		}
	}
	text, err := SerializeXML(obj)
	if err != nil {
		return "", errors.New(err.Error())
	}
	return text, nil
}
