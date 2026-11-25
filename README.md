# deslop-please-go

An opinionated linter to remove sloppy Go code structure. A Go take on the idea
of [deslop-please](https://github.com/deterministic-machines/deslop-please)
(Rust, MIT), following *A Philosophy of Software Design*: **deep** modules,
**narrow** interfaces. Here a module is a Go package.

```bash
go install github.com/vasilvestre/deslop-please-go/cmd/deslop-please@latest   # or: go build ./cmd/deslop-please
```

```bash
deslop-please                   # ./... of the module in the current directory
deslop-please ./internal/...    # only these packages
```

Exit code is 1 while errors remain, 2 on a load error (type errors, bad flags).
Test files and generated files are skipped. Only packages of the main module are
analysed.

```text
error[field-leak]: `Items` is known outside `RunQueue`: 1 place in 1 package
  --> sched/sched.go:6:4
  = note: sched/sched.go:6:4: `sched.Admit`
    queue.RunQueue
      Items ◀━━━ 1  sched
      kept: limit

deslop-please: 1 error, well done!
```

## It's not my fault

```bash
deslop-please --update-allow .deslop-please.allow
deslop-please --allow .deslop-please.allow
```

One `<lint> <key>` per line, sorted. Fixing an entry never fails the check, and
a tangle or module cycle that shrinks stays allowed. The tool tells you when
entries are fixed so you can tighten the file.

## The lints

- **module-cycle**: sibling directories, each with the packages below it, that
  use each other in a circle. Go forbids import cycles between packages, so a
  circle can only run between subtrees (`a/x` uses `b/y`, `b/z` uses `a/w`).
  A package and what is inside its directory make no circle.
- **tangle**: things of one package (a type with its fields and methods, a
  function, a variable) that know each other in a circle, written in more than
  one file, or more than seven of them.
- **field-leak**: a struct that keeps at least one field to itself and has a
  method that uses its fields guards them. Each of its fields that code of
  another package writes, borrows, indexes, ranges over or calls into is known
  outside it. Plain reads (copies) and composite literals are no leak.
- **shallow-module**: a package that shows more than seven functions (used by
  other packages), at least as many as it hides, and a list that long: more
  than seven things, or more than seven functions of one type. Forwarders
  (one-statement bodies) and library-protocol methods (`String`, `Error`, ...)
  count for nothing.
- **long-file**: a non-test, non-generated `.go` file of more than 800 lines.
- **package-bucket**: a package whose name (last element of its path) is
  `util`, `utils`, `common`, `helpers`, `helper`, `misc`, `shared`, `base` or
  `types`. Name-based only.
- **exported-mutex**: an exported field of an exported struct whose type is
  `sync.Mutex`, `sync.RWMutex`, `sync.Once`, `sync.WaitGroup` (value or pointer)
  or a channel, or an exported embedded `sync.Mutex` / `sync.RWMutex`.
- **lonely-interface**: an exported interface with at least one method that
  has exactly one implementing named type (value or pointer receiver) among the
  analysed packages, declared in the interface's own package, with no use of
  the interface or its methods from another package. Interfaces embedding an
  interface from outside the analysed packages (`error`, `io.Reader`) and
  constraints are skipped. With no implementer, or two, nothing is reported.

## Agent skill

`skills/deslop-please/SKILL.md` is a skill that tells a coding agent when to run
the tool and how to fix findings from first principles. Install it with your
skill manager, for example `npx skills add vasilvestre/deslop-please-go`, or copy
the folder into your agent's skills directory.

## Limits

- It is syntactic and type based: no call graph through interfaces, no
  see-through getters for `field-leak`.
- Test files are not loaded, so a `lonely-interface` whose second implementer
  is a fake in a `_test.go` file is reported: that interface is a test seam,
  allowlist it.
- Constructors (`NewX`) are things of their own, not part of `X`.
- Needs a Go toolchain at least as new as the module under analysis.

## License

MIT
