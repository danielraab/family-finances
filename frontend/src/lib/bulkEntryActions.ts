import { api } from "../api/client";
import type { components } from "../api/schema";

type Entry = components["schemas"]["Entry"];

export type BulkFailureReason =
  | "forbidden"
  | "notFound"
  | "invalid"
  | "generic";

export type BulkTaskOutcome =
  | { ok: true }
  | { ok: false; reason: BulkFailureReason };

/**
 * One selected entry's unit of work for a bulk action — id/title are for
 * the run modal's failure list, run() performs the actual PATCH/DELETE
 * (or resolves immediately with no network call when the entry already
 * matches the target state — see design.md's skip-if-unchanged decision).
 */
export type BulkTask = {
  id: string;
  title: string;
  run: () => Promise<BulkTaskOutcome>;
};

function reasonFor(status: number): BulkFailureReason {
  if (status === 403) return "forbidden";
  if (status === 404) return "notFound";
  if (status === 400 || status === 422) return "invalid";
  return "generic";
}

async function patchEntry(
  id: string,
  body: components["schemas"]["EntryUpdate"],
): Promise<BulkTaskOutcome> {
  const { response } = await api.PATCH("/api/entries/{id}", {
    params: { path: { id } },
    body,
  });
  return response.ok
    ? { ok: true }
    : { ok: false, reason: reasonFor(response.status) };
}

export function buildSetCategoryTasks(
  entries: Entry[],
  categoryId: string | null,
): BulkTask[] {
  return entries.map((entry) => ({
    id: entry.id,
    title: entry.title,
    run: async () => {
      if ((entry.category_id ?? null) === categoryId) return { ok: true };
      return patchEntry(entry.id, { category_id: categoryId });
    },
  }));
}

export function buildRecurringTasks(
  entries: Entry[],
  recurringTransactionId: string | null,
): BulkTask[] {
  return entries.map((entry) => ({
    id: entry.id,
    title: entry.title,
    run: async () => {
      if ((entry.recurring_transaction_id ?? null) === recurringTransactionId) {
        return { ok: true };
      }
      return patchEntry(entry.id, {
        recurring_transaction_id: recurringTransactionId,
      });
    },
  }));
}

export function buildDeleteTasks(entries: Entry[]): BulkTask[] {
  return entries.map((entry) => ({
    id: entry.id,
    title: entry.title,
    run: async () => {
      const { response } = await api.DELETE("/api/entries/{id}", {
        params: { path: { id: entry.id } },
      });
      return response.ok
        ? { ok: true }
        : { ok: false, reason: reasonFor(response.status) };
    },
  }));
}

/**
 * Add unions each entry's current tags with tagIds; Remove subtracts them;
 * Set replaces the entry's tag set outright. Every mode skips the request
 * (immediate success) when the entry's tags already equal the result —
 * see design.md.
 */
export function buildTagTasks(
  entries: Entry[],
  tagIds: string[],
  mode: "add" | "remove" | "set",
): BulkTask[] {
  return entries.map((entry) => ({
    id: entry.id,
    title: entry.title,
    run: async () => {
      let next: string[];
      if (mode === "add") {
        if (tagIds.every((id) => entry.tag_ids.includes(id)))
          return { ok: true };
        next = Array.from(new Set([...entry.tag_ids, ...tagIds]));
      } else if (mode === "remove") {
        if (!tagIds.some((id) => entry.tag_ids.includes(id)))
          return { ok: true };
        next = entry.tag_ids.filter((id) => !tagIds.includes(id));
      } else {
        next = tagIds;
      }
      return patchEntry(entry.id, { tag_ids: next });
    },
  }));
}
