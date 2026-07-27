# `timeline` — foreground-process event store & analytics

The `timeline` component is the consumer end of the [`monitor`](../../monitor/README.md)
pipeline. It takes the stream of enriched foreground-process *change events*,
persists each one into the SQLite **event store**, and derives higher-level views
from it: the timeline of sessions, per-app category links, a re-enrichment/indexing
backlog, CSV export, and an app-switch **graph** used for focus scores and entry
suggestions.

Everything is expressed over one domain type —
[`models.ForegroundProcess`](models/foreground_process.go) — which is the typed,
persisted counterpart of `monitor.ForegroundProcess` (the enrichment bag resolved
into concrete fields).

## Command / Query convention

The package follows a light CQRS split. Each unit of work is a struct holding its
inputs plus a single method that resolves its dependencies (DB queriers) from the
`*services.Services` registry at call time:

| Shape | Method | Examples |
| --- | --- | --- |
| `Command*` (writes) | `Exec(ctx, svcs) error` | `CommandUpsertForegroundProcess`, `CommandSubscribeReporter` |
| `Query*` (reads) | `Exec(ctx, svcs) (T, error)` / `Stream(ctx, svcs) utils.StreamFn[…]` | `QueryExportCSV`, `QueryGetUnenrichedForegroundProcesses`, `QueryGetUnindexedForegroundProcesses` |

There is no named interface for this contract — it is a structural convention. The
free functions `CollectTimeline` / `SeqTimeline` are pure in-memory helpers with no
DB dependency.

## End-to-end data flow

```mermaid
flowchart LR
    subgraph ingest [Ingest write path]
        R[monitor reporter<br/>change events] -->|enrich| E[stack.Stack]
        E -->|monitor.ForegroundProcess| CV[MonitorForegroundProcessConverter]
        CV -->|models.ForegroundProcess| CH[(chan)]
        CH --> SR[CommandSubscribeReporter]
        SR -->|per event| UP[CommandUpsertForegroundProcess]
        UP -->|WriteTx| DB[(SQLite event store)]
    end
    subgraph derive [Read / derive]
        DB --> UN[Query*ForegroundProcesses<br/>streams] --> W[enrichment / indexing workers]
        DB --> AG[application_graph<br/>GraphFrom / GraphChan] --> M[focus scores,<br/>entry suggestions]
        DB --> CT[CollectTimeline / SeqTimeline] --> CSV[QueryExportCSV]
    end
```

The ingest side is assembled in [`app/app.go`](../../app/app.go) →
`startForegroundProjection`: it subscribes to the reporter, `enrich`-es each change
event, adapts it with `MonitorForegroundProcessConverter`, and feeds a
`chan models.ForegroundProcess` into `CommandSubscribeReporter.Exec`, which runs one
`CommandUpsertForegroundProcess` per event. Shutdown is a single `ctx` cancel that
closes the channel and exits the command loop.

## Package map

| Package | Path | Role |
| --- | --- | --- |
| `timeline` | this dir | Command/query surface over the event store (ingest, streams, export, in-memory shaping) |
| `models` | [`models/`](models/) | Typed domain types (`ForegroundProcess`, `Enrichments`, `UsageSeq`, `TitleSource`) + accessors |
| `converters` | [`converters/`](converters/) | [goverter](https://github.com/jmattheis/goverter)-generated adapters: DB row → model and `monitor.ForegroundProcess` → model |
| `application_graph` | [`application_graph/`](application_graph/) | Directed app-switch graph → focus scores, entry suggestions, productivity metrics |

## Ingest (`subscribe_reporter.go`, `upsert_foreground_process.go`)

[`CommandSubscribeReporter`](subscribe_reporter.go) is a long-running loop. It drains
its `Chan <-chan models.ForegroundProcess`, and for each event runs a
`CommandUpsertForegroundProcess`, threading the **previous** process as
`PreviousProcess` so the upsert can tell "open a new session" from "close the current
one". It exits on `ctx.Done()`.

[`CommandUpsertForegroundProcess.Exec`](upsert_foreground_process.go) persists one
observation inside a single serialized `WriteTx`. Ordering matters because of foreign
keys:

```mermaid
flowchart TD
    A{idle?} -->|no| C[UpsertApplicationCategory<br/>by category_id]
    C --> AP[UpsertApplication]
    AP --> MAP[UpsertApplicationCategoryMap<br/>application ↔ category]
    A -->|yes| FP
    MAP --> FP[UpsertForegroundProcess]
    FP --> META[InsertForegroundProcessMetadata]
    META --> TL{PreviousProcess == nil?}
    TL -->|yes| OPEN[UpsertTimeline initial_*]
    TL -->|no| CLOSE[UpsertTimeline end_*]
```

- **Category → application → map** run only for non-idle samples (an idle sample has
  no app). The category comes from `Enrichments.Appmetadata.Category`; it is resolved
  against the seeded `application_categories` taxonomy by its natural key
  (`category_id`, the [`enums_categories`](../../enums/enums_categories/category.go)
  value), and the returned surrogate id is linked to the application via
  `application_application_categories_map`.
- **Foreground process + metadata** are written for every sample (idle included);
  `application_id`/`pid` collapse to `NULL` when idle via `utils.ZeroNil`.
- **Timeline**: the first observation opens an entry (`initial_foreground_process_id`);
  each subsequent observation closes the open entry
  (`end_foreground_process_id`). Boundary rows are shared through the
  `created_at_utc` upsert on `foreground_processes`.

> This is the streaming Command ingest. It records raw observations and defers
> enrichment/indexing to the pull-based streams below; it does not publish transition
> events or enqueue semantic indexing inline.

## Re-enrichment & indexing streams (`get_unenriched_*.go`, `get_unindexed_*.go`)

`QueryGetUnenrichedForegroundProcesses` and `QueryGetUnindexedForegroundProcesses`
each expose `Stream(ctx, svcs) utils.StreamFn[models.ForegroundProcess]` — a
keyset-paginated closure that walks the event store for observations still needing
work:

- **unenriched** — rows whose metadata has not yet been decorated (app category,
  browser vendor, …).
- **unindexed** — rows not yet turned into semantic documents/embeddings.

Each page is fetched via the corresponding `read_queries` query (keyed off the last
seen `foreground_process` id), and every row is turned into a `models.ForegroundProcess`
by the matching `converters` row converter. These feed background workers that pull
work rather than being pushed to.

## In-memory timeline shaping (`foreground_process.go`)

Pure helpers over a sorted slice of observations (no DB):

- `CollectTimeline(stream) *list.List` — asserts the input is timestamp-ascending and
  collapses **consecutive equal** observations (`ForegroundProcess.IsEqual`) into a
  linked list of distinct states.
- `SeqTimeline(list) []models.UsageSeq` — walks the list producing
  `UsageSeq{Start, End, Killed}` neighbor triples (previous/next around each node),
  the shape the graph and duration math consume.

## CSV export (`export_csv.go`)

`QueryExportCSV{Timeline list.List}.Exec` renders a linked list of observations into
a CSV string (header + one row per process: identifiers, pid, window title, title
source, timestamp, idle, killed).

## `models/` — domain types

[`models.ForegroundProcess`](models/foreground_process.go) mirrors the monitor type
but with resolved enrichments:

```go
type ForegroundProcess struct {
    AppName, AppIdentifier, AppPath *string
    PID                             *int64
    WindowTitle                     *string
    TitleSource                     *TitleSource
    Timestamp                       time.Time
    Idle, Killed                    bool
    Enrichments                     Enrichments // Appmetadata / Browser / Location
}
```

Enrichment sub-structs: `AppMetadata` (FriendlyName, Description, `Category`, IconPath,
Source), `Browser` (Vendor, `Category`, Tab, CdpURL, Domain), `Location` (nullable
lat/long + public IP). Accessors carry the semantics: `IsIdle()` (asserts identity
fields are nil when idle), `IsBrowser()` (non-zero browser enrichment), `IsEqual()`
(the change-key used by `CollectTimeline`), and `Category.IsProductive()` (drives the
graph's productive/unproductive split). `UsageSeq` is the neighbor-triple type;
`Usage.IsKilled()` reports termination.

## `converters/` — goverter adapters

Boundary mapping is generated, not hand-written (`//go:generate go tool goverter gen .`).
Each converter is an annotated interface compiled to an exported zero-size struct with
a `ToForegroundProcess` method:

| Converter | Source | Used by |
| --- | --- | --- |
| `MonitorForegroundProcessConverter` | `monitor.ForegroundProcess` | ingest (`app.go`) — resolves the enrichment bag (app-metadata category, browser, location) into typed fields |
| `GetUnenrichedForegroundProcessesRowConverter` | `readqueries.GetUnenrichedForegroundProcessesRow` | unenriched stream |
| `GetUnindexedForegroundProcessesRowConverter` | `readqueries.GetUnindexedForegroundProcessesRow` | unindexed stream |

Non-trivial mappings are pipe functions in [`converters.go`](converters/converters.go):
`monitorEnrichments` (reads the `app_metadata`/`browser`/`location` bag entries),
`appmetadataCategory` (numeric cast between the two identically-ordered `Category`
enums), `pidInt32ToInt64`, `parseTitleSource`, `parseBrowserCategory`, `zeroNilString`.

## `application_graph/` — app-switch analytics

Builds a directed graph (backed by [`gograph`](https://github.com/hmdsefi/gograph))
whose vertices are apps and whose edges are observed switches, guarded by an embedded
`sync.RWMutex`. Two constructors:

- `GraphFrom(*list.List)` — batch-build from a `CollectTimeline` list.
- `GraphChan(ctx, <-chan ForegroundProcess)` — incremental; a goroutine folds live
  observations in.

Vertex/edge meta accumulate duration, visit/switch counts, intervals, and category.
Analytics methods (each takes `numMutualConnections`, the threshold for treating an
app cluster as strongly connected via Tarjan SCC):

| Method | Returns |
| --- | --- |
| `GetFocusScores(n)` | `map[string]float64` — per-app focus metric (cycle-collapsed spans) |
| `GetEntrySuggestions(start, n)` | `[]EntrySuggestion` — contiguous app-cluster time blocks to log |
| `GetStronglyConnectedApps(n)` | `map[string][]StronglyConnectedEdgesMeta` — mutually-connected app clusters |
| `GetAverageProductiveDuration()` / `GetAverageUnproductiveDuration()` | `time.Duration` — split by `Category.IsProductive()` |
| `GetTimeToProductive()` | `time.Duration` |
| `GetVertexMeta(label)` / `GetEdgeMeta(from, to)` | accumulated meta for one node/edge |

## Cross-cutting design themes

- **CQRS-ish surface.** `Command*`/`Query*` structs each own their inputs and pull
  DB dependencies from the service registry in `Exec`/`Stream`, so callers never wire
  queriers by hand.
- **The event store is the source of truth.** Every observation is persisted raw;
  the timeline, category links, graph, and exports are all *derived* from it, and
  enrichment/indexing are pull-based backlogs rather than inline work.
- **Absent vs empty, persisted.** Pointer identity fields survive the monitor→model
  boundary, and `utils.ZeroNil` maps idle zero-values back to `NULL` columns.
- **Generated boundaries.** Row→model and monitor→model conversions are goverter
  output; only genuinely custom logic (enrichment-bag resolution, enum widening) is
  hand-written as pipe functions.
- **Category is a seeded taxonomy.** `application_categories` is seeded from
  `enums_categories`; ingest resolves by the natural key and links apps through a
  many-to-many map, so category data is consistent regardless of enrichment timing.
