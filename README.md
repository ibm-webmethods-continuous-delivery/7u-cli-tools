# 7u-cli-tools

A collection of portable, statically linked CLI tools designed to improve local AI agent and developer performance.

The host needs only `docker compose`. Building, testing, linting, and the VS Code dev environment all run in containers.

---

## Tools

| Tool | Image | Purpose | Agent Skill |
|---|---|---|---|
| [`ibmdocs`](code/ibmdocs) | `iwcd-ibmdocs-s:scratch` | Fetches IBM product documentation from the public API, caches locally, and exports agent-ready Markdown | [`code/ibmdocs/SKILL.md`](code/ibmdocs/SKILL.md) |
| [`archictl`](code/archictl) | `iwcd-archictl-s:scratch` | Converts Archi (`.archimate`) models to compact YAML and back, with validated, deterministic edit commands | [`code/archictl/SKILL.md`](code/archictl/SKILL.md) |

---

## Installation & Binaries

### Pre-built binaries

Pre-compiled standalone binaries for all major platforms (Linux, macOS, Windows on x86_64 and ARM64) are attached to each [GitHub Release](https://github.com/ibm-webmethods-continuous-delivery/7u-cli-tools/releases):

| Platform | Architecture | `ibmdocs` Binary | `archictl` Binary |
|---|---|---|---|
| Linux | x86_64 (amd64) | `ibmdocs-linux-amd64` | `archictl-linux-amd64` |
| Linux | ARM64 (aarch64) | `ibmdocs-linux-arm64` | `archictl-linux-arm64` |
| macOS | Apple Silicon (arm64) | `ibmdocs-darwin-arm64` | `archictl-darwin-arm64` |
| macOS | Intel (amd64) | `ibmdocs-darwin-amd64` | `archictl-darwin-amd64` |
| Windows | x86_64 (amd64) | `ibmdocs-windows-amd64.exe` | `archictl-windows-amd64.exe` |

```sh
# Example: Download native binary on macOS Apple Silicon
curl -Lo ibmdocs https://github.com/ibm-webmethods-continuous-delivery/7u-cli-tools/releases/latest/download/ibmdocs-darwin-arm64
chmod +x ibmdocs && sudo mv ibmdocs /usr/local/bin/

curl -Lo archictl https://github.com/ibm-webmethods-continuous-delivery/7u-cli-tools/releases/latest/download/archictl-darwin-arm64
chmod +x archictl && sudo mv archictl /usr/local/bin/
```

---

## Layout

```text
code/<tool>/               one Go module per tool: go.mod, cmd/<tool>/main.go, SKILL.md
build/Dockerfile.tool      shared Dockerfile: static build -> scratch image with /<tool> and /SKILL.md
.devcontainer/             dev image (go, golangci-lint, make, shellcheck) + devcontainer.json
docker-compose.yml         one service per tool image, plus the `dev` service
Makefile                   runs inside `dev`; iterates over code/*/ automatically
go.work                    workspace file; keeps gopls and make build consistent
```

---

## Building from Source

All builds run inside the dev container (no Go toolchain required on the host).

### 1. Build Cross-Platform Binaries (All OSes & Architectures)

To build static binaries for all five release targets (`linux-amd64`, `linux-arm64`, `darwin-arm64`, `darwin-amd64`, `windows-amd64`) and generate SHA-256 `checksums.txt`:

```sh
export UID=$(id -u) GID=$(id -g)

# Build all tools for all platforms:
docker compose run --rm dev make build-all

# Or build a single tool for all platforms:
docker compose run --rm dev make build-all TOOL=ibmdocs
docker compose run --rm dev make build-all TOOL=archictl

# Output binaries and checksums land in target/bin/:
ls -lh target/bin/
```

Individual platform targets:

```sh
docker compose run --rm dev make build-linux-amd64   # Linux x86_64
docker compose run --rm dev make build-linux-arm64   # Linux ARM64
docker compose run --rm dev make build-darwin-arm64  # macOS Apple Silicon
docker compose run --rm dev make build-darwin-amd64  # macOS Intel
docker compose run --rm dev make build-windows       # Windows x86_64 (.exe)
```

### 2. Build Container Images

```sh
docker compose build                                  # build ALL tool scratch images
docker compose build ibmdocs                          # build one tool image
docker compose build archictl

docker compose run --rm ibmdocs version               # run a built tool
docker compose run --rm archictl version
```

Inject release metadata:

```sh
VERSION=ibmdocs/v0.0.8 GIT_COMMIT=$(git rev-parse --short HEAD) docker compose build ibmdocs
```

### 3. Tests & Quality Checks

```sh
docker compose run --rm dev make test                 # test every tool
docker compose run --rm dev make vet                  # go vet every tool
docker compose run --rm dev make lint                 # golangci-lint every tool
docker compose run --rm dev sh                        # interactive dev shell
```

---

## Consumption from Containerfiles

Each tool produces a minimal `scratch` image holding `/<tool>` (verified static binary), CA certificates, and `/SKILL.md`.

Consume tools in downstream Dockerfiles without credentials:

```dockerfile
ARG __ibmdocs_image=iwcd-ibmdocs-s:scratch
FROM ${__ibmdocs_image} AS ibmdocs_stage
FROM alpine:3
COPY --from=ibmdocs_stage /ibmdocs /usr/local/bin/ibmdocs
```

---

## Adding a Tool

1. Create `code/<name>/` with its own `go.mod`, `cmd/<name>/main.go`, and optional `SKILL.md`.
2. Add a `<name>` service block in `docker-compose.yml` (copying an existing tool service).
3. Add `./code/<name>` to `go.work`.

The root `Makefile` automatically discovers and tests all directories under `code/*/`.

---

## License

See [LICENSE](LICENSE).
