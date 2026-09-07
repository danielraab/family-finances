## MODIFIED Requirements

### Requirement: Home navigation

The sidebar navigation SHALL contain a single "Home" item linking to
`/home` (the authenticated account dashboard, see `web-client-home`). The
active navigation item SHALL be visually distinguished when its route is
the current route.

#### Scenario: Navigating home

- **WHEN** the visitor activates the "Home" navigation item
- **THEN** the browser navigates to `/home` and the "Home" item is shown as
  active
