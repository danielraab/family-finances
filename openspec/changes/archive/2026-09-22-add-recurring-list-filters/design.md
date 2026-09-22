## Context

Three read endpoints sit on `recurringtransaction.Store.List`: the
listing, the summary, and the preview. Today they narrow it two
different ways. The listing and the summary build a `Filter`
(`AccountIDs`, `SelfTransfers`) that the store turns into SQL. The
preview builds the same `Filter`, gets everything back, and then walks
the rows itself applying `PreviewFilter`'s `CategoryID`/`CategoryMode`/
`TagID` in a Go loop.

Adding category and tag to the listing means picking one of those two
places — and the choice decides whether the third endpoint keeps its own
copy of the rule.

## Decisions

### The filter goes in `Filter`, and the store applies it

`Filter` gains four fields, exactly the shape `entry.Filter` already
carries:

```go
CategoryID   *string      // caller-supplied
CategoryMode CategoryMode // caller-supplied, "" means subtree
CategoryIDs  []string     // resolved by Service; the only one a Store reads
TagID        *string
```

`Service.resolveFilter` (which replaces `resolveAccountIDs`, and now does
for the category what that already did for accounts) expands `CategoryID`
through `CategoryLookup.Subtree` into `CategoryIDs`, and the two stores
turn `CategoryIDs`/`TagID` into a `category_id = ANY(…)` clause and a
`recurring_transaction_tags` existence clause.

**Why not filter in the service, the way `Preview` does today?** Because
then the rule lives in `Service` for three endpoints while the account
and self-transfer halves of the same filter live in the store, and every
future filter has to choose a side. Pushing it down means `Filter` is one
coherent thing again: everything in it narrows the query, and the store
is where narrowing happens. It also matches `internal/entry`, which is
the package this one is deliberately shaped after — `entry.Filter`'s doc
comment already describes precisely this `CategoryID` → `CategoryIDs`
resolution.

The cost is that the rule is now written twice, once in SQL and once in
the memory store. That is already true of `AccountIDs` and
`SelfTransfers`, and the memory store is unit-test-only.

### `Preview` reuses it rather than keeping its own loop

With the fields on `Filter`, `PreviewFilter` keeps only `AccountIDs` and
`To`, and `Preview` builds a `Filter` carrying the caller's category and
tag. Its in-process `allowedCategories` map and its `containsString` tag
check both go.

One behaviour is deliberately preserved rather than inherited: `Preview`
still forces `SelfTransferBothLegs` and still drops ended templates
itself. Neither is a content filter.

### `CategoryIDs` is nil when there is no filter, empty when it matches nothing

`Subtree` returns an empty slice for a category the caller holds no
permission on — it doubles as the permission check. So "no category
filter" and "a category filter the caller may not use" both arrive at the
store as a `CategoryIDs` with no elements, and the store cannot tell them
apart by length.

The listing must not answer the second case with *every* row, so the
store's guard is `f.CategoryIDs != nil`, not `len(f.CategoryIDs) > 0`:
`Service` leaves it nil when `CategoryID` is nil and sets it to a
non-nil, possibly-empty slice otherwise, and an empty `ANY('{}')` matches
nothing.

This is a deliberate divergence from `internal/entry`, whose store guards
on length — so `GET /api/entries?category_id={a category the caller
cannot see}` currently returns that caller's entries *unfiltered* rather
than none. Fixing that is an entry-side change with its own tests and
isn't in this change's scope; recurring simply doesn't reproduce it.

### No `AllAccounts` equivalent

`entry.Filter.AllAccounts` lifts the account restriction entirely when
the caller holds real permission on the filtered category or tag, so that
a shared category shows its entries across accounts the caller cannot
otherwise see. The recurring listing has never had that, and neither has
its preview endpoint.

Adding it here would mean templates on accounts a caller has no
permission on start appearing in their list — the right call to make
deliberately, against `category-sharing`'s requirements, not as a
side effect of adding a filter control. So the account scope is unchanged:
every account the caller can see, narrowed by `account_id`.

That also means no `TagLookup` call is needed on this path. The tag
filter is a plain match against the tags on templates already in scope.

### The frontend controls are single selects, and the panel is shared

`/entries` renders one `<select>` per filter, with an "All …" option
standing for "unset", and keeps each in the URL. `/recurring` does the
same three. `account_id` stays repeatable on the wire — that's the
existing contract and the preview endpoint uses it — but the control
sends at most one, the same way `/entries`' does.

The panel chrome — the heading, the count badge, the collapse toggle
below `sm`, the clear-all button — is lifted out of
`entries.index.tsx` into `components/FilterPanel.tsx` and rendered by
both pages. It is a presentational shell: it owns the open/closed state
and the layout grid, and takes the active count and a clear handler from
the page, so each page keeps its own notion of what counts as a filter
(`/entries` counts its date range once across three URL parameters;
`/recurring` counts the self-transfer checkbox only when checked).
Extracting it changes no `/entries` behaviour, so `web-client-entries`
gains no delta — its requirements describe the panel, not where the JSX
lives.

The three chrome strings move from `entries.filters.*` to a top-level
`filters.*` namespace, since they are now the shared component's own. The
per-control labels stay per page: `/recurring` gets
`recurring.filters.account`/`allAccounts` and so on, rather than reaching
into the `entries` namespace for them.

### The summary takes every filter the listing takes

Already a stated requirement for the two self-transfer flags — "so the
total it returns is always the total of the rows the listing would
return" — and it extends unchanged to the three new parameters. The page
sends one query object to both requests, as it does today.
