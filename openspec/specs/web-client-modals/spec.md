# web-client-modals Specification

## Purpose

The shared modal shell (`src/components/Modal.tsx`) every dialog in the
web client renders through, so backdrop, centering, panel styling, title
treatment, dismissal behavior, and stacking are consistent everywhere
rather than re-implemented per call site.

## Requirements

### Requirement: Every dialog renders through one shared modal shell

The web client SHALL render every modal dialog through a single shared
component (`src/components/Modal.tsx`) that owns the backdrop overlay, the
centering wrapper, the panel container styling, and the dialog title.
No route or component SHALL construct its own `Dialog` /`DialogPanel`
markup directly. The shell SHALL expose a `size` variant — `sm` (default)
and `md` — as the only supported way to vary the panel's width and
density; it SHALL NOT accept a caller-supplied class string for the panel,
so every dialog in the app stays visually identical at a given size.

#### Scenario: A dialog renders at the default size

- **WHEN** a component opens a modal without specifying a size
- **THEN** it renders in the shell's `sm` panel, with the same overlay,
  centering, and title treatment as every other `sm` dialog

#### Scenario: A wider dialog opts into the md size

- **WHEN** a component opens a modal that needs the wider panel (a map, a
  multi-field form)
- **THEN** it passes `size: "md"` and renders in the shell's `md` panel

#### Scenario: A dialog cannot restyle its own panel

- **WHEN** a component needs a panel appearance the shell does not offer
- **THEN** the shell is extended with a named variant rather than the
  component passing panel classes through

### Requirement: A modal may be non-dismissable while work is in flight

The shell SHALL accept a `dismissable` flag, defaulting to `true`. When
`dismissable` is `false`, pressing Escape and activating the backdrop
SHALL NOT close the modal, and the shell SHALL supply that behaviour
itself rather than requiring the caller to pass a no-op close handler.

#### Scenario: A running bulk action cannot be dismissed

- **WHEN** a modal reporting an in-flight bulk action is open with
  `dismissable: false` and the visitor presses Escape or clicks the
  backdrop
- **THEN** the modal stays open

#### Scenario: The same modal becomes dismissable once finished

- **WHEN** the in-flight work completes and the modal re-renders with
  `dismissable: true`
- **THEN** Escape and a backdrop click close it

### Requirement: A modal opened from within a modal stacks in mount order

A modal opened while another modal is open SHALL render above it without
either one declaring a stacking level, relying on portal mount order. The
shell SHALL apply the same fixed z-index to every modal it renders.

#### Scenario: A nested confirmation renders above its parent

- **WHEN** a confirmation modal is opened from inside an already-open
  edit modal
- **THEN** the confirmation renders above the edit modal, and dismissing
  it returns to the edit modal still open
