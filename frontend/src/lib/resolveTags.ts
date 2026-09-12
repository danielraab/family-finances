import { api } from "../api/client";
import type { components } from "../api/schema";

type Tag = components["schemas"]["Tag"];

/**
 * Resolves free-text tag names to ids: reuses an existing tag by exact name
 * match, or creates it via `POST /api/tags`. Shared by the entry create/edit
 * forms (one name resolved per submitted entry) and the import wizard's
 * batch tag resolution (`names` resolved once for the whole import — see
 * `ImportRunStep.tsx`).
 */
export async function resolveTagIds(
  names: string[],
  existingTags: Tag[],
): Promise<string[]> {
  const ids: string[] = [];
  for (const name of names) {
    const existing = existingTags.find((tag) => tag.name === name);
    if (existing) {
      ids.push(existing.id);
      continue;
    }
    const { data } = await api.POST("/api/tags", { body: { name } });
    if (data) ids.push(data.id);
  }
  return ids;
}
