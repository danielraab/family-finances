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
