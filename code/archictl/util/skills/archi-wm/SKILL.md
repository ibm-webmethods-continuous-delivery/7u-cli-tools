---
name: archi-wm
description: Use when modeling webMethods components (Integration Server, MSR, MFT, API Gateway, Universal Messaging, webMethods.io) in an ArchiMate model — provides element-type mappings, deployment patterns, and YAML archetypes as building blocks.
---

# wm-archimate — webMethods ArchiMate building blocks

**Requires**: load `archictl` and `archi-base` first.

---

## 1. Component → ArchiMate type mapping

| webMethods Component | ArchiMate Type | Notes |
|---|---|---|
| Integration Server (IS) instance | `ApplicationComponent` | Runtime hosting IS packages |
| Microservices Runtime (MSR) instance | `ApplicationComponent` | Lightweight IS, container-native |
| IS Package | `ApplicationComponent` (part of IS) | Deployment unit — composed within IS |
| IS Flow Service / operation | `ApplicationService` | Exposed by the IS/MSR instance |
| ActiveTransfer Server | `ApplicationComponent` | MFT server instance |
| MFT Transfer endpoint | `ApplicationService` | |
| API Gateway instance | `ApplicationComponent` | |
| Published API | `ApplicationService` / `ApplicationInterface` | REST or SOAP product |
| Universal Messaging Realm | `ApplicationComponent` | Pub/sub messaging server |
| UM Channel / Queue | `ApplicationService` | |
| webMethods.io tenant | `ApplicationComponent` | SaaS iPaaS |
| Command Central / Platform Manager | `ApplicationComponent` | Service Delivery tooling |
| webMethods Helm Chart | `ApplicationComponent` | Kubernetes deployment artefact |
| OpenShift / Kubernetes Cluster | `Node` | Container hosting platform |
| IS Runtime Operator | `SystemSoftware` | On OCP: serves IS ApplicationComponent |
| MSR Runtime | `SystemSoftware` | On OCP: serves MSR ApplicationComponent |

---

## 2. Deployment patterns (always start from archetype)

### IS / MSR on Kubernetes

Read `archetypes/wm-patterns.model.yaml` for element IDs, then read
`archetypes/wm-is-msr-ocp.view.yaml` for the layout pattern:

```
Node (OCP Cluster)
  └─ composes → SystemSoftware (IS Runtime Operator)
                  └─ serves → ApplicationComponent (IS Instance)
                                └─ composes → ApplicationComponent (IS Package)
                                               └─ realizes → ApplicationService (Flow Service)
```

### MFT Hybrid pattern

Read `archetypes/wm-mft-hybrid.view.yaml`:

```
ApplicationComponent (ActiveTransfer On-Prem)
  realizes → ApplicationService (MFT Transfer Service)
    ↕ Association (hybrid channel)
ApplicationComponent (ActiveTransfer SaaS)
```

### On-premises IS (no Kubernetes)

```
TechnologyService (OS / JVM)
  └─ serves → ApplicationComponent (IS Instance)
                └─ realizes → ApplicationService
```

---

## 3. Archetype workflow

1. Read `archetypes/wm-patterns.model.yaml` — note element names and IDs.
2. Read the relevant view archetype for the deployment pattern.
3. Split the target model; clone matching elements, adjust names/IDs/docs.
4. Connect via `archictl connect` or edit YAML; then `archictl join`.
5. `archictl validate` before reporting success.

---

## 4. ID convention

```
id-wm-<layer>-<component>-<nn>
```

Examples: `id-wm-app-is-01`, `id-wm-app-msr-01`, `id-wm-app-mft-01`,
`id-wm-tech-ocp-node-01`, `id-wm-app-agw-01`, `id-wm-app-um-realm-01`.

---

## 5. Adding new archetypes

When the user introduces a new webMethods component pattern that is not in `archetypes/`:

1. Model it in `archetypes/wm-patterns.model.yaml` (add elements + relations).
2. Create a new `archetypes/<name>.view.yaml` for the layout.
3. Update the mapping table in §1 of this skill.
