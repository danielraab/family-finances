import type { ParsedRow } from "./parseFile";

/** Shared by the mapping step's column pickers (a global example — the
 * first non-empty value found anywhere in the file) and the dry-run step's
 * per-row remap pickers (a row-specific example — that row's own value) so
 * both truncate the same way. */
export const EXAMPLE_MAX_LENGTH = 28;

export function truncateExample(value: string): string {
  return value.length > EXAMPLE_MAX_LENGTH
    ? `${value.slice(0, EXAMPLE_MAX_LENGTH)}…`
    : value;
}

/** The file-wide example per column: the first non-empty value found
 * anywhere in the file. Used by the mapping step's column pickers and by
 * the dry-run step's *bulk* remap panel, which spans many rows and so has
 * no single row to draw an example from. The per-row remap pickers
 * deliberately don't use this — they show the row being remapped its own
 * value, since another row's would misrepresent it. */
export function collectFieldExamples(
  fields: string[],
  rows: ParsedRow[],
): Record<string, string> {
  const examples: Record<string, string> = {};
  for (const field of fields) {
    for (const row of rows) {
      const value = row[field];
      if (value && value.trim() !== "") {
        examples[field] = value;
        break;
      }
    }
  }
  return examples;
}
