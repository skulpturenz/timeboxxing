# `application` — application category lookups

The `application` component answers one question: *which categories is this application
classified under?* It is the only component [`timeline`](../timeline/README.md) reads from
outside its own tree.

Classification itself happens elsewhere. The enrichers in [`monitor`](../../monitor/README.md)
resolve a category at ingest, and the timeline write path persists it against the seeded
`application_categories` taxonomy. This package is the read side of that link.

## Surface

| Symbol | Path | Role |
| --- | --- | --- |
| `QueryGetApplicationCategories` | [`get_application_categories.go`](get_application_categories.go) | `{ApplicationIDs []int64}` → `map[int64][]enumscategories.Category` |
| `models.ApplicationCategories` | [`models/`](models/) | A `[]Category` wrapped in a struct, for the goverter merge |

`Exec` short-circuits to an empty map when handed no ids — before it touches the service
registry, so a caller with nothing to resolve pays nothing. Otherwise it runs one
`GetApplicationCategories` read over the whole id set and parses each stored `code` back into
a [`enumscategories.Category`](../../enums/README.md) with `enumscategories.Parse`. That parse
is why the enum's `Parse` has to stay a total inverse of its `String()`: a code this package
cannot read back is a failed timeline page, not a missing label.

## Why the category is a separate read

An application maps to categories **many-to-many** through
`application_application_categories_map`. Joining the taxonomy into
[`get_timeline.sql`](../../db/read_queries/get_timeline.sql) would fan a single timeline entry
out into one duplicate row per classification, so the timeline row carries only
`application_id`. Each page collects the distinct ids it mentions, resolves them all in one
call here, and merges the result onto both endpoints of every entry via
`ApplicationCategoriesConverter.MergeAppMetadata`.

That merge is also why `models.ApplicationCategories` exists at all rather than the bare slice:
a goverter `update` converter maps *from a struct*, so the slice needs a wrapper to be a legal
source type. Nothing else constructs or reads it.

> **Note:** the schema, the query and this package's return type all admit many categories per
> application, but `Exec` asserts exactly one after each append. The comment there records the
> intent — Linux is the only platform expected to yield multiple, and the plan is to pick the
> most specific — but that reconciliation is not written yet. Until it is, a genuinely
> multi-category application trips the assertion in dev and CI, and in a release build silently
> hands the caller a slice whose extra members `firstCategory` discards.

## Cross-cutting design themes

- **One read per page, not per row.** The batched id set is the whole point of the API shape;
  entries chain, so a row's closing observation is the next row's opening one and the id set
  deduplicates naturally.
- **The taxonomy is seeded, the link is data.** Categories come from the frozen
  [`enums_categories`](../../enums/enums_categories/category.go) const block seeded into
  `application_categories`; which applications point at which is ordinary rows written at
  ingest. That split is what keeps classification consistent regardless of enrichment timing.
- **Codes cross the boundary, integers stay inside.** The map is keyed by surrogate
  `application_id` and valued by the enum, but what lives in the column is the code — so the
  read path is a `Parse`, never a cast.
