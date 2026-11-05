---
name: archi-base
description: Use when designing or editing an ArchiMate model — provides the core metamodel grammar, cloud provider element-type mappings, ID conventions, and YAML archetype files to use as building blocks.
---

# archi-base — ArchiMate grammar and building-block archetypes

**Requires**: load `archictl` first for the tool protocol and YAML syntax.

This skill provides:
1. **Metamodel rules** — which ArchiMate type represents which concept
2. **Domain router** — which archetype file to read for each domain
3. **ID convention** — how to name IDs
4. **Workflow** — how to use archetypes to build new models

---

## 1. Core metamodel rules

### Rule 1 — Shared Responsibility Stack

Every cloud-managed capability follows this vertical chain:

```
Capability                (Strategy — what the solution does)
    ▲ Realization
TechnologyService         (Provider-managed platform feature, e.g. Azure ACR, IBM MQ Cloud)
    │ Serving
    ▼
ApplicationComponent      (Solution-provisioned instance owned by the team)
    ▲ Realization
ApplicationService        (Business capability exposed to upstream consumers)
```

Reference archetype: `archetypes/rule1-capability-managed-service.view.yaml`

### Rule 2 — SDI standpoints (two valid perspectives)

- **Service perspective**: `ApplicationComponent` realizes `ApplicationService`, served by `TechnologyService`
- **Infrastructure/topology perspective**: `Node` (compute), `CommunicationNetwork` (VNet/Subnet), `TechnologyInterface` (NIC, Public IP)

Reference: `archetypes/rule2-networking-standpoints.view.yaml`

### Rule 3 — Scoped containers (Resource Group, Namespace)

```
TechnologyService (platform capability) → serves → ApplicationComponent (team instance)
ApplicationComponent → realizes → ApplicationService (boundary service)
Specialization: specific RG → generic RG archetype
```

Reference: `archetypes/rule3-resource-group.view.yaml`

### Rule 4 — NICs and Public IPs

```
Node composes TechnologyInterface (NIC)
TechnologyInterface (NIC) associates CommunicationNetwork (Subnet)
TechnologyInterface (NIC) associates TechnologyInterface (Public IP)
```

Reference: `archetypes/rule4-nic-public-ip.view.yaml`

---

## 2. Domain router — load only what the task needs

| Domain | Read first | YAML archetype (elements) | YAML archetype (example view) |
|---|---|---|---|
| Generic metamodel + shared responsibility rules | this skill §1 | `archetypes/style-guide.model.yaml` | `archetypes/rule1-capability-managed-service.view.yaml` |
| Azure (RG, VNet, VMSS, NIC/PIP, ACR) | `archetypes/style-guide.model.yaml` | (same — azure types are in the StyleGuide model) | `archetypes/rule3-resource-group.view.yaml`, `rule4-nic-public-ip.view.yaml` |
| IBM Integration / CP4I (MQ, ACE, APIC) | `archetypes/ibm-integration.model.yaml` | `archetypes/ibm-integration.model.yaml` | `archetypes/ibm-integration-e2e.view.yaml` |
| webMethods | load `wm-archimate` skill | — | — |

**Workflow**: read the archetype model YAML to understand available element names and IDs,
then clone and adapt them into the target model's YAML, then `archictl join`.

---

## 3. ID convention

```
id-<scope>-<layer>-<type>-<name>-<nn>
```

| Scope prefix | Meaning |
|---|---|
| `id-sg-` | StyleGuide generic archetype |
| `id-az-` | Azure-specific instance |
| `id-aws-` | AWS-specific instance |
| `id-ibm-` | IBM CP4I component |
| `id-wm-` | webMethods component |
| `id-el-` | Expert Labs internal |

Layer codes: `cap` (Capability), `app` (Application), `tech` (Technology), `biz` (Business), `rel` (Relation).

Replace any `id-[0-9a-f]{32}` found in an existing model before adding new elements.

---

## 4. Archetype-driven workflow

When asked to build or extend a model:

1. **Identify domain** from the request.
2. **Read the relevant archetype model YAML** from `archetypes/` to see available element names and IDs.
3. **Read the relevant view YAML** to understand the expected layout pattern.
4. **Split the target model** with `archictl split` to read its current state.
5. **Draft a YAML patch** using the archetype elements as templates, adapting names/IDs/docs.
6. **Apply via commands** (`archictl add`, `archictl connect`) for small additions, or
   edit the split YAML and `archictl join` for larger changes.
7. **Validate**: `archictl validate`, then `archictl roundtrip`.

---

## 5. AWS quick-reference (no archetype file yet)

| AWS Resource | ArchiMate Type | Notes |
|---|---|---|
| Lambda Function | `ApplicationComponent` + `TechnologyService` (platform) | |
| API Gateway | `ApplicationComponent` + `ApplicationService` | |
| SQS / SNS | `ApplicationComponent` + `ApplicationService` | |
| S3 Bucket | `ApplicationComponent` + `ApplicationService` | |
| VPC / Subnet | `CommunicationNetwork` | |
| ENI | `TechnologyInterface` | |

ID prefix: `id-aws-<layer>-<service>-<nn>`
