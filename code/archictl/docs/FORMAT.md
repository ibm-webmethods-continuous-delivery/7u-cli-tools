# Archi YAML format and conformance spec

Status: spike v0, derived from the Python reference of the ArhiYamlFiles spike. Any other implementation
(the Go CLI) must reproduce the behaviour below and pass the data in `testdata/golden/`.

Goal: a lossless, deterministic, token-cheap view of an Archi `.archimate` file for AI agents.
On Archisurance (about 33k XML tokens) the semantic file is about 11% and one view file at most
about 1.5k tokens (tiktoken `o200k_base`).

## 1. Files

`split <model.archimate> --out <dir> [--ids]` writes:

* `model.yaml`: the whole model except the content of ArchiMate diagrams.
* `views/<slug>.yaml`: one per `ArchimateDiagramModel` that has children. The slug is the view name
  with every run of non `[\w.-]` characters replaced by `_`, stripped of `_`, cut to 50 characters;
  collisions (case-insensitive) get `_2`, `_3`, and so on. The view entry in `model.yaml` gets
  `file: views/<slug>.yaml` and loses its content.

`join <dir> --out <out.archimate> [--no-rules]` reads every `*.yaml` under `<dir>` (`views/...` keyed by relative POSIX
path) and rebuilds the XML. Join validates (section 7); `--no-rules` skips only the relationship legality check.

Input: plain XML only (UTF-8, BOM allowed). Zip containers (`model_zipped.archimate`, first byte
not `<`) are out of scope for v0. XML comments and processing instructions are rejected.

## 2. YAML dialect

* Every plain scalar is a **string** (no int/bool/null resolution). `id: 007` stays `"007"`.
  Writers must quote where a reader would otherwise resolve differently; readers must
  not resolve implicit types. Multi-line text uses block scalars (`|-`).
* Key order is significant for output stability but not for meaning, except inside lists.
* The Python writer emits flow style for short collections. YAML text is **not** compared
  byte-for-byte; conformance compares parsed data (section 9).

## 3. model.yaml: folder tree

Top key is `model`; its scalars are the model attributes (`name`, `id`, `version`, ...), then children.

```yaml
model:
  name: Archisurance
  id: 11f5304f
  version: 5.0.0
  business/:                       # top-level folder, key = Archi folder type + "/"
    Processes/:                    # sub-folder, key = name + "/"
      BusinessProcess: {Handle Claim: 556, Register: 564}   # Type: {name: id}
      BusinessActor:
        Insurer: {id: 1180, color: red, doc: Some text}     # extra attributes -> dict form
      Flow: [1333 Handle Claim > Inform Customer]           # relations, section 5
  diagrams/:
    ArchimateDiagramModel:
      Business Process View: {id: 3761, viewpoint: business_process_cooperation, file: views/Business_Process_View.yaml}
  kids:                            # residual: anything that did not fit, in raw form (section 3, last bullet)
  - {purpose: An example of a fictional Insurance company.}
```

* Folders: key `Name/`. Top-level folders use their Archi type as the name (`strategy business application
  technology motivation implementation_migration other relations diagrams`). Their display name is
  the default (Strategy, Business, Application, "Technology & Physical", Motivation,
  "Implementation & Migration", Other, Relations, Views) unless a `name:` entry says otherwise. A top-level
  folder with no type whose name equals one of those type keys stays in `kids` (raw).
* Folder ids are **not** stored; they are regenerated on join (section 6).
* Elements are grouped per type: `Type: {name: id}`. Elements with other attributes, `doc`
  or `props` use `Name: {id: ..., attr: ...}`. A null/empty value means "generate an id".
  The group key is the type without the `archimate:` prefix.
* An element that does not fit (empty or missing name, duplicate name in the same type group of the same folder, missing id,
  unknown tag) goes to the residual `kids` list of its folder in raw form.
* `doc` is `<documentation>`, `props` is a list of `{key: value}` from `<property key= value=>`; `b: "x y w h"`
  is `<bounds>` (`_` = attribute absent). Both appear before nested children. A `<documentation>` is folded to `doc`
  wherever it appears among the children (Archi writes it last for views).
* Raw form (used in `kids`): `{Type: {attr: v, ..., kids: [...]}}` mirrors an XML element; `Type` is the
  `xsi:type` without `archimate:`; plain tags use the tag name; `_tag` records a tag that is not the
  default for that kind (`element`, `child`, `sourceConnection`); `_text` is element text alongside attributes;
  `_ns` holds non-standard namespaces on the root. Reserved attribute names (`doc props b kids _tag _text _ns`)
  cannot occur as XML attributes (the tool refuses such a file).

## 4. views/<slug>.yaml: compact diagrams

A view file is a list of nodes. Notation (inside `ArchimateDiagramModel` only):

| Text | Meaning |
|---|---|
| `o:<elementId> x y w h` | `DiagramObject` for that element with bounds |
| `c:<relationshipId>` | `Connection` for that relationship; source and target derived |
| `{o:<id>: {b: ..., fillColor: ..., kids: [...]}}` | same, with extra attributes or nested nodes |
| `{c:<id>: {kids: [{bendpoint: {startX: ..}}]}}` | connection with bendpoints and so on |

Rules:

* A connection is nested in (listed under `kids` of) its **source node**. Archi refuses files where it is not;
  the expander moves misplaced connections into their source node.
* Node ids are omitted unless referenced (by `targetConnections`, or by a non-derivable source/target). Omitted ids
  are regenerated as `d<base36 counter>`.
* `targetConnections` is derived (all incoming connection ids) when it equals that set.
* Connection `source`/`target` are derived only when both ends are unique drawn nodes of that element and the connection
  is nested in its source node; otherwise they are written explicitly.
* If any node in a view has no `id` already (old files), the view is left in raw form.
* Groups, Notes, DiagramModelReference, Canvas*/Sketch types stay in raw form as `{Group: {...}}`.
* Documentation of a view lives in its `model.yaml` entry (`doc:`), not in the view file.

## 5. Relationships

In the semantic file, per folder, grouped by relationship type
(`Access Aggregation Assignment Association Composition Flow Influence Realization Serving Specialization Triggering`):

```yaml
Flow:
- 1784 Claims Handling > Customer                 # id source > target
- 1786 Claims Handling > Customer | claim info    # optional relationship name after " | "
- r9 BusinessProcess > a10 | docrel:              # extra attributes: doc / props hang off the item
    doc: |-
      text
- {id: r2, source: a1, target: a2, accessType: 1} # raw dict when attributes other than name/doc/props exist
```

* Endpoints are **element names** when the name is unique across all elements, non-empty, equal to its strip, free of
  `>`, `|`, tab, CR/LF, and not equal to any element or relationship id. Otherwise the element **id** is used. In ids mode
  (`split --ids`) the compact `id src>tgt [name]` form is used (ids, no spaces around `>`); the reader accepts both forms.
  When only one end can be a name, the spaced form is used with the id on the other end.
* Reading: an endpoint that is a known id is an id; otherwise it must be a unique name, else error
  `ambiguous element reference: 'X'` or `unknown element reference: 'X'`.
* Relationship types outside the eleven (for example `SomeUnknown`) stay raw in `kids` as `Type#id src>tgt`.

## 6. Id handling

* Archi accepts any unique string as an id. The optional `shortids` pre-step replaces GUID-like ids
  (`id-<32 hex>`, 32 hex, or 8-4-4-4-12 hex) and every reference (including `profiles`) with `i<base36>`, and
  `expand` reverses it with the saved map. `split` itself never rewrites ids.
* Generated ids: folders `f<b36>` and elements without id `e<b36>` (one counter, document order, depth first), view nodes
  `d<b36>` (separate counter). Generated ids never collide with existing ones. Exact generated values are **not**
  part of the contract (section 9 normalises them); uniqueness is.
* New ids created by commands are `x<N>`, N from `count(ids)+1` upward, first unused.

## 7. Validation

On `join` and on every editing command, unless `--no-rules`:

1. References: `source`, `target`, `archimateElement`, `archimateRelationship`, `model`, and each entry of
   `targetConnections` must exist; ids must be unique. (Join only; the one fixture with dangling `targetConnections`
   therefore round-trips only with validation off, as in `testdata/golden/`.)
2. Relationship legality from Archi's `relationships.xml` (embedded from `internal/rules/`): for each relationship whose
   ends are elements, the type must be allowed for (source type, target type). Message:
   `Composition Handle Claim (BusinessProcess) > Insurer (BusinessRole) [x9] not allowed; allowed: Association, Flow, Serving, Triggering`.
   Pre-existing illegal relations do not block a command; only newly introduced errors do.

## 8. Commands (archi_ops)

`find [text] [--type T] [--json]`, `show <ref> [--json]`, `orphans [--json]`, `unused [--json]`,
`add <Type> <name> [--folder A/B] [--doc]`, `connect <Type> <src> <tgt> [--name] [--doc]`, `rename <ref> <new>`,
`delete <ref>`, `add-to-view <view> <ref> [--x --y]`, `validate`.

* `<ref>`: an id or a unique element name; ambiguity lists candidates with ids; unknown names get up to 3 near
  matches ("did you mean").
* `find` matches case-insensitively on a name substring, or exactly on an id.
* Output strings and failure messages are recorded verbatim in `testdata/golden/ops/commands.json` and are part of the contract.
* Edits are atomic: if validation after the edit shows a new error, nothing is written.
* `delete` removes the element, all relationships touching it (transitively, relations on relations), every diagram
  object and connection that references them, and cleans `targetConnections`.
* `add-to-view`: new node at (12, bottom+24) unless `--x/--y`, size 120x55; draws every existing relationship between
  the new element and elements already in that view; refuses duplicates. Views are found by name or id.
* `add`: top-level folder per type (`FOLDER_OF` in `archi_ops.py`); `--folder` creates missing sub-folders
  (placed before elements, as Archi writes them).

## 9. Conformance

`testdata/golden/manifest.json` lists the cases (13 models, 24 command steps, 4 join-error cases).

* `models/<case>/`: `in.archimate`, `yaml/` (names mode), `yaml_ids/` (`--ids`), `out.archimate`, `canon.json`.
  * **split**: parse your `model.yaml` and `views/*.yaml` and compare with the golden files **as parsed data** (maps compare
    unordered, lists ordered, all scalars as strings). Generated ids (folders, nodes) do not appear in the YAML, so this is exact.
  * **join**: join the golden `yaml/` (and `yaml_ids/`) without validation and compare with `canon.json` using the canonical form below.
  * **round trip**: `split(in)` then `join` must be canonically equal to `in`.
* Canonical form (`canon_sig(text, loose=True)` in `archi_spike.py`): for the root and every `folder`, sort children by
  (tag, xsi:type, name, id (blank for folders), descendant count, joined sorted descendant names); ignore the `id` of
  `folder`s; rename every `child` and `sourceConnection` id inside an `ArchimateDiagramModel` to `n<ordinal>` (document order) in
  all references (`id`, `source`, `target`, `targetConnections`, the last sorted); put `documentation` children first;
  then compare (tag, sorted attributes, stripped text for leaves, children) recursively. `canon.json` holds that structure.
* `ops/commands.json`: apply `steps[*].args` in order to a copy of `models/archisurance_annotated/in.archimate`
  (edited in place); each step's exit code (0 or 1) and trimmed output (the error text on failure) must match;
  the final file must equal `ops/canon.json`.
* `join_errors/cases.json`: apply each text replacement to `base/yaml/<file>`, join with validation, expect a failure
  whose message contains `expect_error`.

Golden data is frozen. It was generated by the spike's `gen_golden.py` (Python oracle, not part of this repository).

Known gaps for v0: zip containers; Sketch and Canvas diagram models are kept raw; Archi 5 profiles and nested-element
semantics are carried through raw only; no merge or diff of concurrent edits.
