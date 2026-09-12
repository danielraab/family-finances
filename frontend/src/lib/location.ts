// A location field's value is either a typed address string or a
// JSON-encoded coordinate object produced by device GPS or a dropped map
// pin — the backend never distinguishes the two (see
// backend/AGENTS.md's "Entries" section). This is the single parse
// attempt both the ledger's globe-icon condition and the entry form's own
// live preview use, so the two never disagree about what counts as a
// valid coordinate.

export type Coordinates = { lat: number; lng: number };

export function parseLocation(
  value: string | null | undefined,
): Coordinates | null {
  if (!value) return null;
  let parsed: unknown;
  try {
    parsed = JSON.parse(value);
  } catch {
    return null;
  }
  if (typeof parsed !== "object" || parsed === null) return null;
  const record = parsed as Record<string, unknown>;
  const lat = record["lat"];
  const lng = record["lng"];
  if (typeof lat !== "number" || typeof lng !== "number") return null;
  if (!Number.isFinite(lat) || !Number.isFinite(lng)) return null;
  if (lat < -90 || lat > 90 || lng < -180 || lng > 180) return null;
  return { lat, lng };
}

export function formatCoordinates(coords: Coordinates): string {
  return JSON.stringify({ lat: coords.lat, lng: coords.lng });
}
