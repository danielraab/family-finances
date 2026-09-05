/**
 * Strips undefined-valued keys from obj. Under this project's strict
 * `exactOptionalPropertyTypes`, an optional field typed `field?: X` (as
 * openapi-typescript generates) rejects an object that explicitly sets
 * `field: undefined` — only an *absent* key satisfies it. Building request
 * bodies/query objects with `field: condition ? value : undefined` is much
 * easier to read than a conditional-spread chain, so this bridges the two:
 * call it once around the object literal before handing it to `api.*` or a
 * route's `search`.
 */
export function compact<T extends object>(
  obj: T,
): { [K in keyof T]?: Exclude<T[K], undefined> } {
  const out: Record<string, unknown> = {};
  for (const [key, value] of Object.entries(obj)) {
    if (value !== undefined) {
      out[key] = value;
    }
  }
  return out as { [K in keyof T]?: Exclude<T[K], undefined> };
}
