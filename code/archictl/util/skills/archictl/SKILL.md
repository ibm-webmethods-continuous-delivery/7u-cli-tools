---
name: archictl
description: Use when the user wants to run archictl commands to read, split, join, validate, search, or edit an Archi (.archimate) model file.
---

# archictl — CLI tool protocol

This skill covers **only the archictl command interface**. For ArchiMate grammar and
element-type rules, load `archi-base`. For webMethods-specific constructs, load `wm-archimate`.

## 0. Verify binary

```
archictl version
```

If missing: build from `7u-cli-tools/code/archictl` (`go build ./cmd/archictl`) or
install from the release page and stop.

## 1. Read a model (cheaply)

```bash
archictl split <model.archimate> --out <model.d>
```

- Read `<model.d>/model.yaml` for the full element + relation inventory.
- Open only the specific `views/*.yaml` you need — never load all views at once.
- Use `archictl find <model.archimate> <name>` to locate an element by name.
- Use `archictl show <model.archimate> <name-or-id>` for element details, relations, views.

## 2. Edit commands (prefer over YAML edits)

| Intent | Command |
|---|---|
| Add element | `archictl add <model> <Type> <name> [--folder F] [--doc D]` |
| Add relation | `archictl connect <model> <Type> <source> <target> [--name N]` |
| Place on view | `archictl add-to-view <model> <view> <element>` |
| Rename | `archictl rename <model> <old> <new>` |
| Delete (cascades) | `archictl delete <model> <name-or-id>` |

Each command validates before writing and is atomic.
Exit code 1 = refused — read the message, use the suggested id or spelling, retry.

## 3. Bulk YAML edit workflow

```bash
archictl split <model.archimate> --out <model.d>
# edit model.d/model.yaml and/or views/*.yaml
archictl join <model.d> --out <model.archimate>
```

`join` validates before writing. The YAML format is:

```
# model.yaml — elements
model:
  name: <name>
  id: <id>
  strategy/:
    Capability:
      <ElementName>: <id>          # compact form
      <ElementName>:               # verbose form (with doc)
        id: <id>
        doc: |
          documentation text
  application/:
    ApplicationComponent:
      <name>: <id>
    ApplicationService:
      <name>: <id>
  technology/:
    TechnologyService: ...
    Node: ...
    SystemSoftware: ...
    TechnologyInterface: ...
    CommunicationNetwork: ...
  relations/:
    Realization:
      - <rel-id> <source-name> > <target-name>
    Serving: [...]
    Composition: [...]
    Association: [...]
  diagrams/:
    ArchimateDiagramModel:
      <ViewName>:
        id: <id>
        file: views/<filename>.yaml

# views/<name>.yaml — diagram layout
- o:<element-id>:            # object (element placed on diagram)
    b: <x> <y> <w> <h>       # bounds
    kids:
    - c:<rel-id>                               # connection — id is the RELATION id from model.yaml
    - o:<child-id>:                          # nested child
        b: <x> <y> <w> <h>
- Group:                     # anonymous group box (no model element)
    name: "<label>"
    b: <x> <y> <w> <h>
    fillColor: '#rrggbb'
    kids: [...]
```

## 4. Validate and check

```bash
archictl validate <model.archimate>      # reference + relationship-legality check
archictl roundtrip <model.archimate>     # confirm YAML round-trip is stable
archictl orphans <model.archimate>       # elements in no view
archictl unused <model.archimate>        # views with no model elements
```

Always run `validate` before reporting success. Always tell the user to reopen the model in Archi.

## 5. Rules

- Never read or edit the raw `.archimate` XML directly.
- Never run two edit commands on the same model in parallel.
- Never leave Archi's default hex UUIDs — replace with semantic IDs before any other work.
