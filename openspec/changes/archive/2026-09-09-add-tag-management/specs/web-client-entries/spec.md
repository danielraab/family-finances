## MODIFIED Requirements

### Requirement: Tags can be created inline from the entry form

The entry form's tag input SHALL match against the visitor's existing,
non-disabled tags as they type, and SHALL NOT offer a disabled tag as a
suggestion. On submission, any entered tag value that does not match an
existing tag SHALL be created (`POST /api/tags`) before the entry is saved
with it attached; a value that matches an existing tag reuses it,
regardless of whether that tag is disabled. An entry being edited that
already carries a tag which has since been disabled SHALL continue to
display and resubmit that tag normally — the exclusion applies only to the
autocomplete suggestion list, not to a tag the entry already has.

#### Scenario: Typing a new tag name creates it

- **WHEN** an authenticated visitor types a tag name that does not match
  any of their existing tags and submits the entry form
- **THEN** a new tag with that name is created and attached to the entry

#### Scenario: Typing an existing tag name reuses it

- **WHEN** an authenticated visitor types a tag name matching one of their
  existing tags and submits the entry form
- **THEN** the existing tag is attached, and no duplicate tag is created

#### Scenario: Disabled tags are excluded from suggestions

- **WHEN** an authenticated visitor types into the tag input and one of
  their existing tags matching the typed text is disabled
- **THEN** that tag does not appear among the suggestions

#### Scenario: An entry keeps showing a tag disabled after it was attached

- **WHEN** an authenticated visitor opens the edit form for an entry that
  carries a tag which has since been disabled
- **THEN** that tag's name still appears among the entry's attached tags,
  and saving the form without removing it succeeds
