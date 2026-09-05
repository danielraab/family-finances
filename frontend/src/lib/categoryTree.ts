import type { components } from "../api/schema";

type Category = components["schemas"]["Category"];

export type CategoryOption = { id: string; label: string };

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
      out.push({ id: c.id, label: "    ".repeat(depth) + c.name });
      walk(c.id, depth + 1);
    }
  }
  walk("", 0);
  return out;
}
