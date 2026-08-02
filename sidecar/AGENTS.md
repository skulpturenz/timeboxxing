# `sidecar` — agent guide

Cross-cutting conventions, as established in [`components/`](components), [`enums/`](enums), [`queue/`](queue) and [`utils/`](utils). **Where other code deviates from a
guideline below, follow those five.**

Only what holds module-wide lives here. Module detail belongs in that module's own `README.md` —
duplicating it here just creates two things to keep in sync.

## Verification

Every change must build with assertions on and pass every lint check:

```bash
go tool task lint             # golangci-lint + gopls, over the branch diff
go build -tags assert .       # CI: .github/workflows/sidecar-build.yml
go test  -tags assert ./...   # CI: .github/workflows/sidecar-test.yml
```

- **Always pass `-tags assert`.** A plain `go build .` compiles assertions out and does not
  reproduce CI. `task build` is that plain build — not a substitute.
- `task lint` is scoped to the branch diff, so your changes must be clean. `task lint-all` covers the
  module and does **not** pass yet — it is not a baseline you may add to. `task lint-fix` applies the
  formatters.
- `task lint-gopls` gates both lint tasks and catches what golangci-lint cannot, notably
  `infertypeargs`: never write type arguments the compiler can infer.
- Tooling is declared in `go.mod`'s `tool (…)` block and invoked as `go tool <x>` — including `task`
  itself. gopls is deliberately excluded (its `honnef.co/go/tools` dependency crashes golangci-lint);
  do not tidy it in. `CGO_ENABLED=1` is mandatory.
- There is no `task test`. Run `task --list` for current targets.

## Schema, migrations and generated code go through the Taskfile

Never hand-create a migration, seed or query file, and never hand-edit generated output.
[`Taskfile.yml`](Taskfile.yml) owns the sequence numbers, naming and generators:

| Change | Command |
| --- | --- |
| Schema migration | `go tool task create-migration -- <name>` |
| Seed for a table | `go tool task create-seeder TABLE=<table> NAME=<name>` |
| Read query / write query | `go tool task create-query -- <name>` / `create-mutation -- <name>` |
| Regenerate | `go tool task generate-models` / `generate-converters` / `generate-env-docs` |

- Migrations are golang-migrate, embedded, and run when the database opens — DDL under `db/schema/`,
  reference data under `db/seeds/<table>/`, each seed directory with its own sequence and
  bookkeeping table.
- **Adding a migration also means** bumping the version constant asserted in `db/db_test.go`,
  updating the schema doc alongside the DDL, and regenerating models if a row shape moved.
- `*.gen.go`, `*.sql.go` and `CONFIG.md` are committed generated output. Regenerate, never edit.
- Write columns out in SQL — `SELECT *` is rejected.

## Structure and naming

- **snake_case files, one operation per file, named for the operation** — `get_timeline.go` holds
  `QueryGetTimeline`. In `utils/` that reads as one helper family per file, named for the concept.
- Directories are snake_case and **package names drop the underscores** (`read_queries` →
  `readqueries`), so every import site aliases explicitly: an all-lowercase alias matching the
  package name for enum and query packages, a camelCase disambiguator otherwise.
- Component layout: operation structs at the package root, domain types under `models/`, generated
  conversion under `converters/`, a self-contained algorithm in its own subpackage.
- Constructors are `New*` for infrastructure and `*From` for build-from-input. **No package-name
  stutter** in exported identifiers. Receivers are a single letter on operation structs, a
  descriptive word on domain types and enums.

## Operations are Command/Query structs

A unit of work is a struct holding its inputs, with one method that resolves dependencies from the
registry at call time. Role-prefixed and verb-first (`QueryGetTimeline`,
`CommandUpsertForegroundProcess`), with **no constructor** — a struct literal with exported inputs
and unexported cursor state.

- `Command*` writes and returns `error`; `Query*` reads and returns a value from `Exec` or a
  `utils.StreamFn` from `Stream`.
- **Never chain the call onto the literal** — a method hung off the closing brace buries the call
  among the fields. Bind the struct, then invoke on the next line. Enforced by ruleguard.

  ```go
  query := QueryGetSecret{Namespace: namespace, Key: key}
  secret, err := query.Exec(ctx, svcs)
  ```

- Dependencies come from `*services.Services[any, any]` inside the method, never through a
  constructor. Infrastructure packages expose a `Register*` + `*FromServices` pair.
- `context.Context` is the first parameter and is threaded through; long-running loops exit on
  `ctx.Done()`. It is never a struct field.
- A self-contained algorithm subpackage is the exception — it exposes methods on its own type, keeps
  storage generic, and puts domain specifics behind a strategy interface. A second flavour is a new
  strategy, not a second builder.

## Enums

[`enums/README.md`](enums/README.md) is the reference. The invariants that bind other packages:

- **`type X int`.** The integer is the identity. A *persisted* enum under `enums/` either backs a
  seeded table, where the integer *is* the row id, or stores its `String()` code in an ordinary
  column; some are not persisted at all. The const block is **append-only** in every case —
  reordering is a data migration.
- One `const` block from `iota`, index 0 a sentinel or the safe default. `String()` is an index into
  a slice literal ordered to match the block, never a `switch`.
- `Parse(s string) (X, error)` at package level, normalising with
  `strings.ToLower(strings.TrimSpace(s))`, no `default:`, closing on the safe default plus
  `fmt.Errorf("unrecognized <thing>: %s", s)`. It must be **total over every code its column can
  hold**, which is not always every code `String()` emits — where the sentinel is stored as `NULL`
  rather than written out, `Parse` rejecting it is correct.
- `fmt.Stringer` is the only interface implemented — no `driver.Valuer`, `sql.Scanner`, JSON hooks
  or codegen. Plain domain methods are fine where the rule belongs to the enum.
- Cross-cutting enums live under `enums/`, with bare member names by default and a package-level
  `Parse` (`enumsreleasechannel.Stable`, `enumsmaskingcategory.Parse`) — the alias carries the noun.
  One owned by a single package stays beside its domain type and takes type-prefixed members and a
  `Parse<Thing>` name (`models.TitleSourceAX`, `models.ParseTitleSource`). Moving an enum between
  the two renames both.

## Reads and writes are separate

Two `sql.DB` pools on one DSN, two sqlc packages over one schema. Read verbs are `Get*` / `List*`;
write verbs are `Upsert*` / `Create*` / `Insert*` / `Delete*`.

- **Reads** go through the read querier — no lock, WAL permits concurrent readers — or the read
  connection for raw SQL sqlc cannot generate.
- **Writes** go through `SerialWriteQuerier` and nothing else. SQLite allows one writer; taking the
  mutex in-process makes `SQLITE_BUSY` unreachable. Its generated wrappers, `WriteTx` and
  `WithWriteConn` share the one mutex. Never construct it outside `db/`.
- **Adding a write query is a two-step change**: the inner querier is a *named field, not embedded*,
  so the `var _ writequeries.Querier` assertion fails to compile until you hand-write the
  lock/defer/delegate wrapper.
- Consumers depend on narrow interfaces, not the concrete database type.
- **Nil means absent, zero means empty** — `utils.Coalesce` reads an optional with a default,
  `utils.ZeroNil` collapses a zero value back to `NULL`.

## goverter

Row↔model conversion lives in a `converters/` package and nowhere else. Each converter is a pair: a
hand-written file declaring an **unexported annotated interface**, and a generated `*.gen.go`
holding the exported struct, with `goverter:output:file` pinning the name.

- Only the entry point is exported; sub-mappings are unexported methods on the same interface.
- Hand-written code there is limited to the **pipe functions** and the helpers they call.
- **Gotcha:** goverter resolves sub-mappings by `(source, target)` signature, so one converter cannot
  hold two mappings over the same source type — split it into two named converters.

## Assertions (`github.com/negrel/assert`)

Build-tag gated and compiled out of a release build. **The idiom is assert-*then*-check-anyway**, so
behaviour is identical with the tag off — an assertion is a development tripwire, never a substitute
for error handling:

```go
category, err := enumscategories.Parse(row.Code)
assert.NoError(err)
if err != nil {
	return appCategories, err
}
```

Assert what the type system cannot express — registry lookups, argument preconditions,
postconditions, sort and shape invariants. Never assert on external or user input; that is a real
error path.

## Errors, logging and concurrency

- Errors are `fmt.Errorf` wrapping with `%w`; messages lowercase and context-prefixed. No
  `samber/oops`, `lo`, `mo` or `do`. Compare with `errors.Is`/`errors.As`.
- A sentinel is only for an error a caller must branch on. `Err*` prefix, `*Error` type suffix.
- Logging is stdlib `log/slog`, always the `*Context` variants, never a global; plain `log` is banned
  outside `main.go`. **Operation and component code does not log** — it returns errors and the wiring
  layer logs them.
- **Backpressure over buffering**: delivery channels are unbuffered so the consumer drives the
  producer and the backlog stays in SQLite, not RAM.
- **Where a helper drops data or reorders concurrent results, that is a documented contract** —
  state it rather than leaving it to be discovered.
- A mutex is a **named field, never embedded**.

## Lint rules that shape the code

`.golangci.yml` is maratori's golden config with deviations. Every threshold change and exclusion
carries a comment saying why, and so must any you add. The ones that change how you write code:

| Rule | What it means |
| --- | --- |
| `exhaustruct` | Every struct literal names every field, including the zero ones. |
| `golines` / `lll` | 120 columns. Wrapped signatures are the formatter's work — leave them. |
| `nolintlint` | `//nolint:<linter> // reason` — specific linter *and* explanation required. |
| `gochecknoglobals` / `gochecknoinits` | No package-level `var`, no `init()`, absent a justified `//nolint`. |
| `mnd` | No magic numbers — name the constant. |
| `exhaustive` | Enum `switch` **and** `map` literals cover every member. |
| `nonamedreturns` / `nakedret` | No named returns, no naked returns. |
| `containedctx` | `ctx` is a parameter, never a struct field. |
| `errcheck` | Type assertions checked too; deliberate discards are explicit `_ =`. |
| `ireturn` | Accept interfaces, return concrete types; returning a type parameter is allowed. |
| `funcorder` | A constructor sits directly after its type. |
| `reassign` | Package variables are never reassigned. |
| `gocritic` ruleguard | House rules an off-the-shelf linter cannot express, in [`.golangci/rules.go`](.golangci/rules.go) — currently the ban on chaining a call onto a struct literal. |

`ctx` is exempt from unused-parameter checks — it stays first and threaded through even where a body
does not read it. Test files are exempt from the globals, error-check and length linters.

Imports use the plain two-group form — stdlib, blank line, everything else by path. No third "local"
group, no license headers.

## Testing

- Tests run fully parallel — no `-p 1`, and **new and changed tests call `t.Parallel()`**.
- Tests live in the **same package**, never an external `_test` one — they reach into unexported
  state, which is why `testpackage` is disabled.
- testify: `require` for setup and preconditions, `assert` for the expectation, with the trailing
  argument as an explanatory message. No suites, no mocks, no golden files.
- **Real SQLite in `t.TempDir()`**, never in-memory. Opening the database runs migrations and seeds,
  so setup is one call. **Seed through the real production write path.** No shared test-helper
  package — each package writes its own fixture helper.
- `<operation>_test.go` holds that operation's tests; `<package>_test.go` holds shared fixtures and
  **zero test functions**. Build-tag pairs use `_on`/`_off` suffixes.
- Names read as sentences: `TestQueryGetTimeline_HasNoClosingBound`. Mostly one scenario per test.
- **The name is the documentation** — a test needing a paragraph above it to say what it covers is
  usually named wrong, or testing two things.

## Comments and READMEs

- **Comments are for important things only, and most declarations have none — that is the target,
  not a gap to fill.** Do not restate a signature, narrate a test its name already describes, or add
  ceremonial package comments. Write one for non-obvious rationale, a load-bearing invariant, an edge
  case, or regression provenance worth preserving. Where a declaration seems to need explaining,
  check first whether a better name removes the need. **The test is whether deleting it would let
  someone make a wrong change** — if not, delete it yourself.
- **A comment under the `-- name:` line of a `.sql` file becomes a Go doc comment.** sqlc copies it
  onto the generated `Querier` method, so it is held to the same bar as any other — and editing one
  means regenerating.
- **Leave existing comments alone unless they are inaccurate.** If your change makes one wrong, fix
  it in the same edit — but do not reword a comment that is merely phrased differently from how you
  would have phrased it.
- **Lowercase where possible.** In-body comments are lowercase-first and usually have no closing
  period. Doc comments read as capitalised only because they start with the declared identifier.
- **Not every function needs a Godoc — prefer a plain note.** Doc comments are optional; most
  exported declarations have none. When the note is about *how* the body works, put it **inside the
  body**, lowercase, where the comment linters do not reach.
- **A comment attached to a declaration is a doc comment, and then the linter applies**: it must
  start with the declared identifier, end with a period (a trailing URL and a `TODO` are the
  exemptions), fit 120 columns, and link stdlib symbols as `[sql.Register]`.
- **A module's `README.md` is the reference for that module — update it in the same change that
  alters the module's behaviour, API, structure or wiring.** Documentation has drifted this way
  before: a fix landed while the README kept describing the bug as outstanding.
- **A README inside a module directory stays high level, at the altitude it is now.** Role, how the
  pieces fit, load-bearing invariants, known gaps. Not an API reference: no per-function
  documentation, no restating signatures. Replace stale paragraphs rather than appending, so it
  keeps its shape instead of growing into a reference.
- House shape when editing one: `` # `pkg` — one-line role ``, cross-links to sibling READMEs, an
  early symbol table, a mermaid `flowchart` for non-obvious flows, `> **Note:**` callouts for known
  gaps, and a closing `## Cross-cutting design themes`. Not every module has one — the shape applies
  when you are writing a module README, not as a quota to fill.
