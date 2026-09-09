## RENAMED Requirements

- FROM: `### Requirement: Home placeholder content`
- TO: `### Requirement: Home feature overview`

## MODIFIED Requirements

### Requirement: Home feature overview

The home page (`/`) SHALL render a feature-overview component in the main
content region, unconditionally for every visitor regardless of
authentication status (no redirect, no auth-based branching). It MUST NOT
contain any `create-next-app` template content, Next.js or Vercel branding,
or links to Next.js documentation or templates.

The overview SHALL present exactly four cards, one per core capability:
creating and managing accounts; adding entries as either a transaction or a
balance adjustment; managing entries with categories and tags; and managing
the category tree. Each card SHALL show an icon and a short title and
description for its capability; the icon SHALL occupy one side of the card
at the card's full height, with the title and description on the other
side, and this side SHALL alternate from card to card. Cards SHALL NOT be
interactive (no navigation link, button, or click handler on the card
itself).

#### Scenario: Home shows the feature overview

- **WHEN** a visitor opens `/`
- **THEN** the main content region shows four cards covering accounts,
  entries (transaction/balance adjustment), entry categorization
  (categories and tags), and the category tree
- **AND** no Next.js logo, Vercel logo, "Deploy Now" button, or template
  links are present

#### Scenario: Feature overview is audience-independent

- **WHEN** the visitor is authenticated rather than anonymous
- **THEN** `/` still renders the same four-card feature overview, with no
  redirect and no difference in content

#### Scenario: Cards are non-interactive

- **WHEN** the visitor examines a feature card
- **THEN** it contains no link, button, or other control that navigates
  away from `/`
