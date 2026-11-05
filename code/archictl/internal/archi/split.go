package archi

// SplitObj turns a parsed model object into the model tree plus the per-view node lists.
// Views with children move to "views/<slug>.yaml" and are replaced by a `file` entry.
// The returned model tree and view lists are ready for DumpYAML.
func SplitObj(obj Obj, namesMode bool) (Obj, map[string][]any, error) {
	if _, err := CompactViews(obj); err != nil {
		return nil, nil, err
	}
	CompactRelations(obj)

	files := map[string][]any{}
	used := map[string]bool{}
	for _, view := range findViews(obj) {
		kids := view.List("kids")
		if len(kids) == 0 {
			continue
		}
		fn := "views/" + Slug(view.Str("name"), used) + ".yaml"
		view.Del("kids")
		files[fn] = kids
		view.Set("file", fn)
	}

	ix := BuildElementIndex(obj)
	tree := TreeCompact(obj)
	if namesMode {
		NamesCompact(tree, ix)
	}
	return tree, files, nil
}

// SplitToFiles converts XML text into relative path -> YAML text.
// model.yaml holds everything except the content of ArchimateDiagramModel views.
func SplitToFiles(text string, namesMode bool) (map[string]string, error) {
	obj, err := ParseXML(text)
	if err != nil {
		return nil, err
	}
	tree, views, err := SplitObj(obj, namesMode)
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	if out["model.yaml"], err = DumpYAML(tree); err != nil {
		return nil, err
	}
	for name, kids := range views {
		if out[name], err = DumpYAML(kids); err != nil {
			return nil, err
		}
	}
	return out, nil
}
