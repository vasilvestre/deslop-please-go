---
name: deslop-please
description: Checks Go code structure with deslop-please (module-cycle, tangle, field-leak, shallow-module, long-file, package-bucket, exported-mutex, lonely-interface) and fixes the findings from first principles. Use when writing, refactoring or reviewing Go code that adds or moves packages, exported functions or struct fields, when a package grows large or its fields are touched from other packages, or when asked to deslop, deepen modules or narrow interfaces.
---

# deslop-please

Install once: `go install github.com/vasilvestre/deslop-please-go/cmd/deslop-please@latest`

## Workflow

1. Run `deslop-please ./...` from the module root, after structural Go changes and before reporting done.
2. Read every finding. Do not hide it with a mechanical move (renaming, splitting a file, adding a getter that only forwards).
3. Fix from first principles:
   - `module-cycle`, `tangle`: what is the intended layer of abstraction? Which package or file sits at which layer? Break the circle by moving the shared piece down, not by adding an interface that points back up.
   - `field-leak`, `shallow-module`: why do these fields or functions exist at all? What are the patterns of use outside? Hide them behind a narrow interface (a method that does the whole job, an unexported field). Fix the cluster together, never one leak at a time.
   - `long-file`: split by responsibility into several files of one package; if a part has its own types and vocabulary, extract it into a package. Never cut at an arbitrary line.
   - `package-bucket`: move each function to the package that owns the data it works on; the bucket empties and goes away.
   - `exported-mutex`: unexport the field and expose methods that do the whole job (lock inside, send inside), so callers cannot lock or wait on their own.
   - `lonely-interface`: delete it and use the concrete type, or move it to the consumer package that needs the abstraction (Go interfaces belong to the caller).
4. For a non-trivial restructure, draw a before and after diagram for the user.
5. Re-run until it prints `deslop-please: ok`.

## Allowlist

Only when the user accepts existing debt: `deslop-please --update-allow .deslop-please.allow`, then run with `--allow .deslop-please.allow`. Never allowlist a finding you introduced.

## Limits

No call graph through interfaces, and getters hide field leaks. Exported data structs (`*Result`, `*Error`) with plain fields are idiomatic Go: judge before restructuring.
