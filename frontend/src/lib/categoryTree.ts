import type { components } from "../api/schema";

type Category = components["schemas"]["Category"];

export type CategoryOption = {
  id: string;
  label: string;
  shared: boolean;
  ownerName: string | undefined;
};

/**
 * Flattens the category tree into a depth-indented list, root categories
 * first (alphabetical), each followed by its descendants (also
 * alphabetical) — a simple, searchable stand-in for a tree widget (see
 * design.md's frontend-architecture decision).
 */
export function flattenCategoryTree(categories: Category[]): CategoryOption[] {
  const byParent = new Map<string, Category[]>();
  for (const c of categories) {
    const key = c.parent_id ?? "";
    const siblings = byParent.get(key) ?? [];
    siblings.push(c);
    byParent.set(key, siblings);
  }
  for (const siblings of byParent.values()) {
    siblings.sort((a, b) => a.name.localeCompare(b.name));
  }

  const out: CategoryOption[] = [];
  function walk(parentKey: string, depth: number) {
    for (const c of byParent.get(parentKey) ?? []) {
      out.push({
        id: c.id,
        label: "    ".repeat(depth) + c.name,
        shared: c.shared,
        ownerName: c.owner_name,
      });
      walk(c.id, depth + 1);
    }
  }
  walk("", 0);
  return out;
}

export type CategoryNode = Category & { children: CategoryNode[] };

/**
 * Builds the nested tree the /categories management page renders: each
 * node's children ordered by sort_order (falling back to id for a stable
 * tiebreak), for the ▲/▼ reorder controls and the parent/child nesting
 * itself.
 */
export function buildCategoryTree(categories: Category[]): CategoryNode[] {
  const byParent = new Map<string, Category[]>();
  for (const c of categories) {
    const key = c.parent_id ?? "";
    const siblings = byParent.get(key) ?? [];
    siblings.push(c);
    byParent.set(key, siblings);
  }
  for (const siblings of byParent.values()) {
    siblings.sort(
      (a, b) => a.sort_order - b.sort_order || a.id.localeCompare(b.id),
    );
  }

  function build(parentKey: string): CategoryNode[] {
    return (byParent.get(parentKey) ?? []).map((c) => ({
      ...c,
      children: build(c.id),
    }));
  }
  return build("");
}
