## ADDED Requirements

### Requirement: Tags link in the sidebar

The `Sidebar` navigation SHALL contain a "Tags" item, visible to an
authenticated visitor, positioned alongside "Categories", that navigates
to `/tags` and is shown as active for `/tags` and every route nested
under it.

#### Scenario: Navigating to tags from the sidebar

- **WHEN** an authenticated visitor activates "Tags" in the sidebar
- **THEN** the client navigates to `/tags`

### Requirement: Tags routes require authentication

`/tags` and every route nested under it SHALL be accessible only to an
authenticated visitor. An anonymous visitor navigating to `/tags` SHALL be
redirected to `/login`. While `useAuth` is `loading`, the route SHALL
render nothing that would flash before the redirect-or-render decision is
made.

#### Scenario: Anonymous visitor is redirected

- **WHEN** an anonymous visitor navigates to `/tags`
- **THEN** the client redirects them to `/login`

### Requirement: The tags page lists the caller's own tags, each showing its entry count and status

`/tags` SHALL fetch the caller's tags (`GET /api/tags`, now owned tags
plus every tag shared with the caller — see `tag-sharing`) and render the
caller's own tags as a flat list, each row showing its name, its
`entry_count` (the number of the caller's own entries currently carrying
it), and an Active/Disabled status. A tag with `entry_count: 0` SHALL show
`0` rather than omitting the indicator.

#### Scenario: Each row shows how many entries use it

- **WHEN** a tag is directly carried by three of the caller's entries
- **THEN** that tag's row shows `3`

#### Scenario: A tag with no entries still shows its count

- **WHEN** a tag has an `entry_count` of `0`
- **THEN** its row shows `0` rather than omitting the indicator

#### Scenario: Disabled tags are visually distinguished

- **WHEN** a tag in the list has `disabled: true`
- **THEN** it is shown with a visibly different status than an enabled
  tag

### Requirement: Tags can only be created and renamed from this page

`/tags` SHALL be the only page offering a form to create a new tag or
rename an existing one — other than the entry form's inline
create-while-tagging flow, which is unchanged by this capability. Creating
calls `POST /api/tags`; renaming calls `PATCH /api/tags/{id}` with the new
`name`.

#### Scenario: Creating a tag

- **WHEN** an authenticated visitor submits the create form with a name
- **THEN** `POST /api/tags` is called and the new tag appears in the list,
  Active, with an entry count of `0`

#### Scenario: Renaming a tag

- **WHEN** an authenticated visitor edits an existing tag's name and
  saves
- **THEN** `PATCH /api/tags/{id}` is called and the list reflects the new
  name

### Requirement: Disabling and enabling a tag apply immediately, without a confirmation dialog

Each tag row SHALL offer a toggle calling `POST /api/tags/{id}/disable` or
`/enable` as appropriate, applied immediately on activation — disabling or
enabling a tag SHALL NOT require a confirmation step, since it is cheaply
reversible and does not affect any entry already carrying the tag.

#### Scenario: Disabling a tag in use

- **WHEN** an authenticated visitor disables a tag that existing entries
  currently carry
- **THEN** the tag's status shows Disabled immediately, with no
  confirmation step, and those entries remain visible and functional
  elsewhere in the app

### Requirement: Deleting a tag requires confirmation, and is blocked while the tag is shared

Each tag row's delete action SHALL require an explicit confirmation step
before `DELETE /api/tags/{id}` is called. Unlike an in-use-by-entry tag —
which stays freely deletable, same as before this capability — a tag
currently shared with anyone SHALL be rejected by the backend (`409`);
the client SHALL surface this as an inline error explaining the tag must
first be unshared, rather than silently removing it from the list. A
tag with no active shares SHALL delete exactly as before: unconditionally,
removing it from every entry that carries it.

#### Scenario: Confirmation blocks an accidental delete

- **WHEN** an authenticated visitor activates "Delete" for a listed tag
- **THEN** a confirmation step appears and `DELETE /api/tags/{id}` is not
  called until it is confirmed

#### Scenario: Deleting an unshared tag in use

- **WHEN** an authenticated visitor confirms deleting a tag, not
  currently shared with anyone, that is attached to one or more of their
  entries
- **THEN** `DELETE /api/tags/{id}` is called, the tag is removed from the
  list, and it no longer appears on any entry

#### Scenario: Deleting a shared tag is rejected

- **WHEN** an authenticated visitor confirms deleting a tag that is
  currently shared with at least one other user
- **THEN** the request returns `409`, the tag remains in the list, and an
  inline error explains that the tag must be unshared first

### Requirement: The tags page shows tags shared with the caller in a separate, flat section, without an entry count

`/tags` SHALL render a "Shared with me" section, positioned below the
caller's own tag list, listing every tag returned by `GET /api/tags` with
`shared: true`. Each row SHALL show the tag's name, a shared indicator
naming the real owner (via `owner_name`, see the shared-icon requirement
below), and the caller's permission tier (`view` or `append`) on it —
SHALL NOT show `entry_count`, even though the field is present on the
underlying tag object. This section SHALL be omitted entirely when the
caller has no tags shared with them.

#### Scenario: A shared tag appears without an entry count

- **WHEN** the caller has a tag shared with them
- **THEN** it appears in the "Shared with me" section showing its name,
  owner, and permission tier, but no entry-count indicator

#### Scenario: No section when nothing is shared

- **WHEN** the caller has no tags shared with them
- **THEN** the "Shared with me" section does not render at all

### Requirement: A shared tag always shows an icon next to its name, revealing the owner on hover

Every tag rendered anywhere it can appear with `shared: true` — the
caller's own list is unaffected, since an owned tag is never `shared` —
SHALL show a small shared-indicator icon immediately next to its name via
`TagLabel`. The icon SHALL be present at all times a shared tag is shown,
not toggled by any interaction; hovering (or focusing, for keyboard/touch
parity) the icon SHALL reveal the tag's `owner_name` in a tooltip. The
owner's name SHALL NOT be shown as standing, always-visible text — unlike
`CategoryLabel`'s badge treatment for categories, `TagLabel` keeps the
owner name hidden until hovered.

#### Scenario: A shared tag's icon is always visible

- **WHEN** a tag with `shared: true` is rendered anywhere in the client
- **THEN** its shared icon is shown next to its name without requiring
  any interaction

#### Scenario: Hovering the icon reveals the owner

- **WHEN** a visitor hovers the shared icon on a tag
- **THEN** a tooltip shows that tag's `owner_name`

#### Scenario: An owned tag shows no shared icon

- **WHEN** a tag the caller owns (`permission: owner`) is rendered
- **THEN** no shared icon is shown next to its name

### Requirement: Each of the caller's own tags offers a Share action

Each tag in the caller's own list on `/tags` SHALL offer a Share action
(alongside its existing rename/disable/delete actions) that navigates to
`/tags/{id}/sharing`.

#### Scenario: Opening the sharing page

- **WHEN** the caller activates Share on one of their own tags
- **THEN** the client navigates to `/tags/{id}/sharing`
