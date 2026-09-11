# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/).

## [Unreleased]

## [0.1.0] - 2026-09-11

### Added

- Account sharing with permission tiers, including the frontend UI for
  managing shared accounts.
- Category sharing with view/append permissions, with shared status and
  owner name surfaced in category dropdowns and category-aware entry
  visibility.
- Tag sharing functionality.
- `entry_count` on category responses, shown in the frontend.
- Category and tags columns in the entries list.
- Independent category/tag selection when generating reports.
- User display names.
- Dependabot configuration for automated dependency updates.

### Changed

- Account types are now free-text input; the separate account types
  management UI was removed.
- Category actions consolidated into a single edit dialog.

### Fixed

- CI: the `publish` job being skipped on every tag push, so tagged
  releases actually build and publish an image.
- Sidebar mobile responsiveness.
- `AccountLabel` layout and responsiveness for shared accounts.
- Invitation-row behavior and the revoke button's enabled condition.
- Magic link and invite acceptance now always resolve to the correct
  account.
- The category sharing page now opens correctly from the Share button.
- Sharing/invite form layout and accessibility.
- Frontend build and runtime compatibility with Node 26.

### Build

- Bumped Go, Node, and various GitHub Actions and frontend/backend
  dependencies.

## [0.0.0] - 2026-09-09

Initial tagged snapshot of the project.

[Unreleased]: https://github.com/danielraab/family-finances/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/danielraab/family-finances/compare/v0.0.0...v0.1.0
[0.0.0]: https://github.com/danielraab/family-finances/releases/tag/v0.0.0
