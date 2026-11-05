# Product Requirements — Archi Model Agent CLI (`archictl`)

**Version:** 0.1.0-draft
**Status:** Implemented (all §5 commands); see §20 for status and the monorepo review
**Source spike:** `ArhiYamlFiles` (Python spike and its Go scaffold `go-cli`). Normative format: [docs/FORMAT.md](docs/FORMAT.md). Acceptance data: `testdata/golden/`.
**Location:** tool `archictl` of the `7u-cli-tools` monorepo, `code/archictl/` (own Go module; build, image, release and dev environment are shared with the other tools, see §3 and §13).

---

## 1. Purpose

`archictl` is a portable, statically linked command-line tool that lets humans and AI agents read and edit [Archi](https://www.archimatetool.com/) models (`.archimate`) cheaply and safely. It:

1. Converts the verbose Archi XML into a token-optimised, lossless YAML (`model.yaml` plus one file per view) and back.
2. Offers deterministic, validated edit and query commands, so an agent need not hand-edit XML or YAML.
3. Validates references and ArchiMate relationship legality (Archi's own rule tables) before anything is written.

Measured in the spike on Archisurance: `model.yaml` is about 11% of the XML's tokens (3.4–4.1k vs ~33k); a single view file is at most ~1.5k tokens.

The tool is a **batch CLI**. It is used by an agent through a **skill** (shell invocation) or through a **local stdio MCP wrapper** (§17). It has no network, socket or daemon component.

---

## 2. Non-Goals (Explicit Out of Scope)

- No network of any kind: no HTTP client or server, no sockets, no telemetry, no update check. This is enforced at build time (§13, `check-no-network`).
- No daemon or long-running mode in v0.1. (A stdio `mcp serve` is deferred, §16.)
- No rendering of diagrams, no layout computation, no image export.
- No replacement for Archi's GUI, and no use of Archi's jArchi scripting or Archi CLI.
- No semantic/architectural advice; the tool is deterministic.
- No concurrent-edit merge in v0.1 (§16).

---

## 3. Platform and Deployment Requirements

### 3.1 Cross-Platform Binary

| Platform | Architecture | Required |
|---|---|---|
| Linux | x86_64 (amd64) | ✅ |
| Linux | ARM64 | ✅ |
| macOS | Apple Silicon (arm64) | ✅ |
| macOS | Intel (amd64) | ✅ |
| Windows | x86_64 | ✅ |

- `CGO_ENABLED=0`; fully statically linked, single file, no runtime dependencies.
- Relationship rule tables (`relationships.xml`, `relationships-keys.xml`) are embedded with `go:embed`; no data files are needed at runtime.
- Output is byte-for-byte identical across platforms (LF line endings, UTF-8 without BOM, on every OS including Windows).

### 3.2 Container Delivery

- Image built by the monorepo's shared `build/Dockerfile.tool` (build context `code/archictl/`): `golang:1-alpine` builder → `FROM scratch` final image with `/archictl` and `/SKILL.md`. Compose service `archictl`, image `iwcd-archictl-s:scratch`, entrypoint `/archictl`.
- The shared Dockerfile also copies CA certificates into the image. `archictl` does not need them (no network); this is accepted as a deviation to keep one Dockerfile for all tools.
- Multi-arch images (`linux/amd64`, `linux/arm64`) are produced by building on each architecture or with `docker buildx`; the monorepo does not yet publish a manifest list.
- Usage: `docker compose run --rm archictl <command> …` from the monorepo, or `docker run --rm -v "$PWD":/work -w /work iwcd-archictl-s:scratch /archictl <command> …`. (The shared image sets neither `WORKDIR` nor `ENTRYPOINT`; compose supplies the entrypoint.)
- The builder stage verifies static linkage (shared Dockerfile).

### 3.3 Binary Release

- Binaries named `archictl-<os>-<arch>[.exe]` for the five targets, in `target/bin/`, plus `checksums.txt` (SHA-256) written by `make build-all`. The checksum file is shared with the other tools of the monorepo.
- Release tag format is `archictl/vX.Y.Z` (see CHANGELOG), independent of the other tools. Binaries and image are built from the same commit.

### 3.4 Development Environment (no host toolchain)

- All Go tooling runs in containers; nothing is installed on the host.
- The monorepo-level `.devcontainer/` (VS Code) and the `dev` service in `docker-compose.yml` (VS Code-independent) share one dev image (Go, golangci-lint, make) and the Go caches under `.cache/`. `go.work` is for editor tooling only; `make` sets `GOWORK=off`.
- Standard invocation (from the monorepo root): `docker compose --profile dev run --rm dev make <target> [TOOL=archictl]`.

---

## 4. Files and Layout

`archictl` operates on files and directories given on the command line. There is no global data folder, no cache and no configuration file.

```
<model>.archimate            # native Archi XML (input and output)
<dir>/                       # YAML form produced by `split`
  model.yaml                 # folder tree, elements, relations, view index
  views/<view>.yaml          # one file per diagram
```

- File and directory layout, dialect, tree rules, view notation, relation notation, and id scheme are defined in [docs/FORMAT.md](docs/FORMAT.md) §1–§6 and are **normative**. Where this document and FORMAT.md disagree, FORMAT.md wins.
- The tool never writes outside the paths given as arguments or `--out`.
- Output files are written atomically (temp file in the same directory, then rename).

---

## 5. Command Reference

```
archictl [global flags] <command> [flags] [args]
```

### 5.1 Global Flags

| Flag | Default | Description |
|---|---|---|
| `--json` | false | Machine-readable JSON on stdout for commands that support it (§9.2) |
| `--verbose`, `-v` | false | Debug logging to stderr |
| `--no-color` | false | Disable ANSI colour |

Environment fallbacks use the prefix `ARCHICTL_` (e.g. `ARCHICTL_NO_COLOR`).

### 5.2 Conversion commands

| Command | Description |
|---|---|
| `split <model.archimate> --out <dir> [--ids]` | XML → YAML directory. Names are used as relation endpoints where unique and safe; `--ids` forces the id form. |
| `join <dir> --out <model.archimate> [--no-rules]` | YAML directory → XML. Runs reference validation and relationship-legality validation first; refuses on error. `--no-rules` skips legality only. |
| `roundtrip <model.archimate>` | Split then join in memory and report whether the canonical form is unchanged (diagnostic). |

### 5.3 Query commands (read-only)

| Command | Description |
|---|---|
| `find <model> <text> [--type T] [--json]` | Case-insensitive search of element, relation and view names. |
| `show <model> <ref> [--json]` | Element details: type, documentation, properties, relations in/out, views. |
| `orphans <model> [--json]` | Elements with no relations. |
| `unused <model> [--json]` | Elements not on any view. |
| `validate <model-or-dir> [--no-rules] [--json]` | Reference and legality validation. |

### 5.4 Edit commands

All edits operate in place on a `.archimate` file, or write to `--out` when given.

| Command | Description |
|---|---|
| `add <model> <Type> <name> [--folder F] [--doc D]` | Add an element; prints the new id (`x<N>`). |
| `connect <model> <Type> <source> <target> [--name N]` | Add a relationship; checked against Archi's legality tables. |
| `rename <model> <ref> <new-name>` | Rename an element. |
| `delete <model> <ref>` | Delete an element, cascading to its relations and view items. |
| `add-to-view <model> <view> <ref>` | Place an element on a view and auto-draw its existing relations. |

**References (`<ref>`):** an id, or a name that is unique in the model. An ambiguous name lists the candidates and exits 1. An unknown name prints up to 3 near matches ("did you mean") and exits 1.

**Atomicity:** an edit is validated after being applied in memory. If validation reports a *new* error, nothing is written and the command exits 1. Pre-existing errors do not block an edit.

Exact stdout strings for every command are recorded in `testdata/golden/ops/commands.json` and are normative.

---

## 6. Conformance to the Spike (Golden Data)

The Go implementation MUST reproduce the oracle's behaviour on `testdata/golden/`:

| Suite | Comparison | Source |
|---|---|---|
| Split | Parsed-YAML data equality (not text) | `models/<case>/yaml`, `yaml_ids` |
| Join / round trip | Canonical form equality (children sorted, folder ids ignored, view-node ids renamed by ordinal, `documentation` first) | `models/<case>/canon.json` |
| Ops | Exact stdout/exit code and resulting canonical form | `ops/commands.json`, `ops/canon.json` |
| Join errors | Exact error text | `join_errors/cases.json` |

13 models, 24 command steps and 4 join-error cases at spike close. Golden data is maintained in `testdata/golden/`, never modified by the Go test run.

The Python oracle (`tools/python-oracle/`) is **not** part of the monorepo: the golden data is the frozen contract. It was generated by the spike's `gen_golden.py` (`archi_spike.py`, `archi_ops.py`, `archi_rules.py`) and can be recovered from the `ArhiYamlFiles` spike if a deliberate format change requires regeneration. The Go code never writes into `testdata/golden/`.

---

## 7. Validation Rules

1. **References**: every `source`, `target`, `archimateElement`, `archimateRelationship`, `model` and `targetConnections` value resolves; ids are unique.
2. **Relationship legality**: source type, target type and relationship type are checked against Archi's `relationships.xml` (embedded). Junctions follow Archi's rules.
3. **View structure**: a diagram connection is nested in its source node; `targetConnections` on the target node lists incoming connection ids.
4. Errors are listed one per line and carry the offending id and, where known, a name.

---

## 8. Configuration Constants

| Constant | Default | Description |
|---|---|---|
| `NEW_ID_PREFIX` | `x` | Prefix of ids generated by edit commands |
| `NEAR_MATCH_LIMIT` | `3` | Max "did you mean" suggestions |
| `NEAR_MATCH_CUTOFF` | `0.6` | Similarity cutoff for suggestions (oracle uses `difflib`; Go port MUST give the same set on the golden cases) |

There are no network, cache or timeout constants.

---

## 9. Output Formats

### 9.1 Human output (default)

- Results on **stdout**, diagnostics and `[ERROR]`-prefixed messages on **stderr**.

### 9.2 JSON output (`--json`)

- Supported on `find`, `show`, `orphans`, `unused`, `validate`. A single JSON document is written to stdout; stderr is unaffected.
- Field names are stable and documented in FORMAT.md §8.

### 9.3 Exit codes

| Code | Meaning |
|---|---|
| `0` | Success |
| `1` | Operation understood but refused: validation failure, ambiguous or unknown reference, illegal relationship |
| `2` | Usage, configuration or I/O error (bad flags, unreadable file, unsupported container) |

Exit-code semantics are identical in human and `--json` modes; in `--json` mode an error is also emitted as `{"error": "..."}` on stdout.

---

## 10. Format Fidelity

- Lossless: every XML construct in the supported set round-trips; unsupported constructs are preserved as raw passthrough, never dropped.
- **Raw passthrough (v0.1):** Sketch and Canvas diagrams, profiles, nested elements the YAML dialect does not model.
- `documentation` may appear anywhere among child elements on input; output order is canonical.
- Multi-line text uses YAML block scalars. All YAML scalars are strings (no implicit typing of ids such as `650`).
- The YAML library MUST NOT reorder keys, and MUST produce the same text on every run (deterministic output; the same input always yields byte-identical files).

---

## 11. Id Management

- Folder ids and view-node ids are tool-managed: they are regenerated on `join` and are not stable across a split/join cycle.
- Element and relationship ids are preserved unless `split` is used in names mode, where they are kept in the YAML and not regenerated.
- Newly generated ids never collide with existing ones.
- Id convention for folder trees, collision handling, and the stress-test cases are specified in FORMAT.md §6.

---

## 12. Performance Baselines

Spike targets (Python, Archisurance ≈ 400 KB XML):

| Operation | Target for Go v0.1 |
|---|---|
| `split` | < 100 ms |
| `join` (with validation) | < 150 ms |
| Any single query/edit command | < 150 ms including process start |
| Binary size | < 8 MB per target |

Models up to 50 MB MUST be handled in streaming-friendly memory (< 10× file size).

---

## 13. Build and Release Requirements

- Language: **Go 1.26+**; CLI framework: **Cobra**; YAML: a pure-Go library that preserves key order (`gopkg.in/yaml.v3` Node API). No other third-party dependency.
- The monorepo `Makefile` is shared and iterates over `code/*/`; restrict to this tool with `TOOL=archictl`. Targets: `build`, `build-all`, `build-<os>-<arch>`, `test`, `test-race`, `vet`, `lint`, `tidy`, `check-no-network`, `list`, `clean`, `help`. The reference targets map as follows: `docker` / `docker-local` → `docker compose build archictl`; `coverage` → `go test -cover ./...`; `deps` → `make tidy`; `release-dry-run` → `make build-all`.
- **`check-no-network`** (optimisation over the reference): fails the build if `go list -deps ./cmd/archictl` contains `net/http`, `crypto/tls` or `net/rpc`. It is scoped by the Makefile variable `OFFLINE_TOOLS` (currently `archictl`), because other tools of the monorepo (`ibmdocs`) legitimately use the network.
- Version via `-ldflags -X github.com/ibm-webmethods-continuous-delivery/7u-cli-tools/code/archictl/cmd.version` (and `.buildDate`, `.gitCommit`); printed by `archictl version`. The Makefile and the shared Dockerfile both inject them.
- Release artefacts, naming and checksums per §3.3.
- Static linkage verified in the Dockerfile builder stage.

---

## 14. Testing Requirements

- Unit tests for: YAML dialect (de)serialisation, relation-line parsing, id scheme, reference validator, legality checker, near-match suggestions, each edit command.
- **Golden conformance tests** (§6) are run by `go test ./...` (`make test TOOL=archictl`) against `testdata/golden/` and MUST pass on every target in CI.
- Round-trip test over all 12 plain-XML fixtures (11 Archi test models plus annotated Archisurance) and the stress model.
- Test the "atomic edit" guarantee: after a refused edit the file is byte-identical.
- Test determinism: running `split` twice yields identical bytes.
- Coverage ≥ 80% on `internal/` packages.
- No integration tests against Archi itself are possible in CI; manual "opens in Archi" checks are recorded in the CHANGELOG for each release that changes `join` output.

---

## 15. Logging

- Progress and diagnostics go to **stderr**; stdout is reserved for results.
- Default level: warnings and errors only (the tool is quiet so agents consume few tokens). `--verbose` adds debug detail via `log/slog`.
- No log files are written.

---

## 16. Open Questions / Deferred Items

| # | Item | Status |
|---|---|---|
| 1 | Zip-container models (`model_zipped.archimate`) | ⏳ Post-v0.1; exit 2 with a clear message in v0.1 |
| 2 | Sketch / Canvas views as YAML | ⏳ Kept raw |
| 3 | Merge/diff of concurrent edits (`archictl diff a.archimate b.archimate`) | ⏳ Post-v0.1 |
| 4 | Stdio MCP server (`archictl mcp serve`) wrapping the same commands | ⏳ Post-v0.1; shell skill is the v0.1 integration |
| 5 | Archi behaviours unverified: sibling ordering, nested diagram-object deletion | ⏳ Verify by opening results in Archi |
| 6 | Profiles / specializations in the YAML dialect | ⏳ Raw passthrough |
| 7 | Batch edit command (`apply <ops.yaml>`) to cut agent round trips | ⏳ Proposed optimisation |
| 8 | Shell completion (`archictl completion`) | ⏳ Free with Cobra |
| 9 | Fuzzy-match parity: Python `difflib` ratio vs Go implementation | ✅ Resolved: exact port of `get_close_matches` in `internal/ops/difflib.go`; golden output matches |
| 10 | Name-based references in `--json` output | ⏳ Confirm with agent experiments |

---

## 17. Agent Integration (Skill and Local MCP)

The tool has no agent-specific code. Integration is by **process invocation over stdio** only.

### 17.1 Skill (v0.1)

`code/archictl/SKILL.md` (copied to `/SKILL.md` in the image, consumed like the other tools' skills; the spike's `util/skills/` path is not used) MUST be **agent-agnostic** (no platform-specific product names) and guide the agent through:

1. **Confirm the binary**: `archictl version`; if absent, point to the README.
2. **Orient cheaply**: `archictl split model.archimate --out model.d`, then read `model.d/model.yaml` (≈ 11% of the XML's tokens). Read only the `views/*.yaml` files that are needed.
3. **Prefer commands to hand edits** for add/connect/rename/delete/add-to-view; they validate and are atomic.
4. **Bulk or structural changes**: edit the YAML, then `archictl join` and read the validation errors.
5. **Always validate**: `archictl validate` before declaring success.
6. **Never** read or edit the raw `.archimate` XML directly unless a command is not available.
7. **Never run two edits on the same model in parallel.**
8. **Handle exit code 1** by reading the message (candidates or near matches) and retrying with an id.
9. Tell the user to re-open the model in Archi (Archi does not auto-reload).

### 17.2 Local MCP (post-v0.1)

A stdio MCP server (`archictl mcp serve`) would expose the §5 commands as tools over stdin/stdout. It is deferred, but the following v0.1 constraints keep it cheap:

- Command logic lives in `internal/` packages that return typed results; `cmd/` only parses flags and formats output.
- Every command has a `--json` form.
- No global mutable state; no `os.Exit` outside `cmd.Execute`.

### 17.3 Concurrency

The tool takes an advisory lock (`<model>.archimate.lock`, created exclusively) for edit commands, to avoid two agents clobbering each other. A stale lock is reported, never silently removed.

---

## 18. Safety

- `join` and edit commands never overwrite a model that has pre-existing *unrelated* damage without reporting it; they refuse only on *new* errors.
- `--out` is recommended for agent runs; the SKILL instructs the agent to keep the original unless told otherwise.
- The tool performs no shell execution and reads no environment secrets.
- Input XML is parsed with external entity resolution disabled.

---

## 19. Proposed Optimisations over the Reference Project

| Area | Optimisation |
|---|---|
| Network | `check-no-network` build guard; `scratch` image without CA certificates |
| Exit codes | Distinct `1` (refused) and `2` (fatal), with a typed `refusedError` |
| Determinism | Byte-identical output across OS; golden tests as the contract |
| Agent economy | `--json`, short stdout, near-match suggestions, planned `apply` batch command |
| Conformance | Golden data lives in the repo (`testdata/golden/`) so the Go implementation is always checkable |
| Build | `-trimpath` for reproducible builds |

---

## 20. Implementation Status and Monorepo Review

Review of the spike's `go-cli/PRODUCT_REQUIREMENTS.md` against the `7u-cli-tools` arrangement (one Go module per tool under `code/<tool>/`, shared `Makefile`, `build/Dockerfile.tool`, `docker-compose.yml`, `.devcontainer/`, `go.work`).

### 20.1 Requirements adapted to the monorepo

| Spike requirement | Monorepo treatment | Section |
|---|---|---|
| Own `Dockerfile`, `dev-compose.yml`, `.devcontainer/archictl-dev01/` | Replaced by the shared `build/Dockerfile.tool`, root `docker-compose.yml` (`archictl` and `dev` services) and root `.devcontainer/` | 3.2, 3.4 |
| `FROM scratch` without CA certificates | Shared Dockerfile copies CA certificates; harmless, accepted deviation | 3.2 |
| `WORKDIR /work` in the image | Neither `WORKDIR` nor `ENTRYPOINT` is set by the shared Dockerfile; callers pass `-w /work` and `/archictl` (compose does it) | 3.2 |
| Per-tool `Makefile` with `docker`, `coverage`, `deps`, `release-dry-run`, `test-verbose` | Root Makefile iterates `code/*/`; `TOOL=archictl` selects this tool; equivalents mapped | 13 |
| `check-no-network` | Added to the root Makefile, scoped by `OFFLINE_TOOLS` because `ibmdocs` uses the network | 13 |
| `util/skills/archictl/SKILL.md` | `code/archictl/SKILL.md`, shipped as `/SKILL.md` in the image | 17.1 |
| `tools/python-oracle/` in the repo | Not ported; `testdata/golden/` is the frozen contract | 6 |
| `checksums.txt` per tool | One shared `target/bin/checksums.txt` | 3.3 |
| Version tags `vX.Y.Z` | `archictl/vX.Y.Z`, independent of other tools | 3.3 |
| Open question 9 (fuzzy-match parity) | Resolved by an exact `difflib` port | 16 |

Unchanged and still normative: command set (§5), exit codes (§9.3), validation (§7), format fidelity (§10), id management (§11), safety (§18).

### 20.2 Implemented in v0.1

Layout: `cmd/` (flag parsing and output only), `internal/archi` (XML and YAML model, split/join, validation, canonical form), `internal/ops` (queries, edits, locking, `difflib`), `internal/rules` (embedded relationship tables), `internal/validate`, `internal/config`.

| Requirement | Status |
|---|---|
| §5.2 `split`, `join`, `roundtrip` | ✅ |
| §5.3 `find`, `show`, `orphans`, `unused`, `validate` (with `--json`) | ✅ |
| §5.4 `add`, `connect`, `rename`, `delete`, `add-to-view` (with `--out`) | ✅ |
| §6 golden conformance: 13 models, 24 command steps, 4 join-error cases | ✅ `go test` |
| §9.3 exit codes 0/1/2; `{"error": …}` in `--json` mode | ✅ |
| §5.1 `ARCHICTL_VERBOSE`, `ARCHICTL_NO_COLOR`, `ARCHICTL_JSON` fallbacks; `--verbose` via `log/slog` | ✅ |
| §13 `check-no-network` | ✅ `make check-no-network` |
| §14 atomic-edit test (byte-identical file after refusal), lock test | ✅ |
| §14 coverage ≥ 80% on `internal/` | ✅ archi 83.7%, ops 86.8%, rules 94.6%, validate 100% |
| §12 performance | ✅ about 10 ms per command on Archisurance; binary about 5 MB (stripped, `-trimpath`), within the 8 MB limit |
| §17.3 advisory lock `<model>.archimate.lock` | ✅ created exclusively; a stale lock is reported, not removed |
| §4 atomic file writes | ✅ temp file in the same directory, then rename |

### 20.3 Remaining gaps

| Item | Note |
|---|---|
| `cmd/` unit coverage (25%) | Cobra wiring is exercised by manual runs only; `ops` golden tests cover behaviour. Add command-level tests. |
| Streaming memory bound for 50 MB models (§12) | Not measured; the parser builds the whole tree in memory. |
| Multi-arch image manifest (§3.2) | Per-architecture builds only. |
| Zip containers (§16 item 1) | Rejected with exit 2 as specified. |
| Opens-in-Archi check (§14) | Manual, not yet recorded in the CHANGELOG. |
| Line endings | Several scaffold files had CRLF in the working tree; consider a `.gitattributes` with `* text=auto eol=lf`. |
