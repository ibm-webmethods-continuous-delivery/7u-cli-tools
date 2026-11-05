---
name: archictl
description: Use when the user asks to read, query or change an Archi (.archimate) ArchiMate model — converts it to compact YAML and applies validated edits with the archictl CLI instead of reading or editing the verbose XML.
---

# archictl — Work with Archi models cheaply and safely

Status: draft; commands are specified in PRODUCT_REQUIREMENTS.md §5 and §17.1.

1. Run `archictl version`. If missing, see the README.
2. Orient: `archictl split model.archimate --out model.d`, then read `model.d/model.yaml`; open only the `views/*.yaml` you need.
3. Prefer commands for single changes: `add`, `connect`, `rename`, `delete`, `add-to-view`. They validate and are atomic.
4. For bulk changes edit the YAML, then `archictl join model.d --out model.archimate`.
5. Run `archictl validate model.archimate` before reporting success.
6. Do not read or edit the raw XML. Do not run two edits on one model in parallel.
7. Exit code 1 means refused: read the message (candidates / near matches) and retry with an id.
8. Tell the user to reopen the model in Archi.
