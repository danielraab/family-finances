# Design

## Context

Three CI jobs are currently limited to pull requests: Postgres integration
tests, translation coverage, and combined image validation.

## Goals / Non-Goals

**Goals:** run those jobs for pull requests and `master` pushes only.

**Non-Goals:** publish images on `master` or run these jobs for ordinary branch
pushes.

## Decisions

Use a shared event condition in each job: pull request, or a push whose ref is
`refs/heads/master`. Keep the existing job permissions and tag publishing flow
unchanged.

## Risks / Trade-offs

- [Longer master CI] → master receives the same post-merge validation as PRs.
