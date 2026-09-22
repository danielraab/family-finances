import { useEffect } from "react";

/**
 * Restores a list page's filter/search/sort state from `localStorage` when
 * it's arrived at with a completely bare URL, and keeps that storage in
 * sync as the state changes — shared by `/entries`, `/recurring`, and
 * `/reports`. See openspec/changes/persist-list-filters/design.md.
 *
 * The restore effect intentionally runs once on mount, not on every
 * `search` change: a route component only mounts on a real navigation into
 * it from elsewhere, whereas "Clear all filters" navigates to the same
 * bare search while already mounted. Keying the restore off mount (rather
 * than off "search is bare") is what lets an arrival restore filters while
 * a same-page clear does not immediately undo itself.
 */
export function usePersistedListFilters<S extends Record<string, unknown>>(
  storageKey: string,
  search: S,
  navigate: (opts: { search: S; replace: true }) => unknown,
): void {
  // biome-ignore lint/correctness/useExhaustiveDependencies: intentionally mount-only — see this hook's doc comment.
  useEffect(() => {
    if (!isBareSearch(search)) return;
    const persisted = loadPersisted<S>(storageKey);
    if (persisted && !isBareSearch(persisted)) {
      navigate({ search: persisted, replace: true });
    }
  }, []);

  useEffect(() => {
    try {
      localStorage.setItem(storageKey, JSON.stringify(search));
    } catch {
      // Unavailable (private browsing, blocked storage, …) — filters
      // simply won't be restored next time; nothing else depends on this.
    }
  }, [storageKey, search]);
}

function isBareSearch(search: Record<string, unknown>): boolean {
  return Object.values(search).every((value) => value === undefined);
}

function loadPersisted<S>(storageKey: string): S | undefined {
  try {
    const raw = localStorage.getItem(storageKey);
    if (!raw) return undefined;
    return JSON.parse(raw) as S;
  } catch {
    return undefined;
  }
}
