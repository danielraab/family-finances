import Papa from "papaparse";

/** One row from a parsed import file, values always normalized to strings. */
export type ParsedRow = Record<string, string>;

export type ParsedFile = {
  rows: ParsedRow[];
  /** Mappable column/field names, in first-seen order. */
  fields: string[];
  /** Top-level JSON keys whose value was a complex structure we don't know
   * how to map (an array, or an object that isn't a recognized amount
   * shape — see `asAmountObject`) — skipped rather than rejecting the
   * whole file; surfaced as a non-blocking hint. Always empty for CSV. */
  ignoredFields: string[];
};

export type ParseFileError = { message: string };

/** A JSON file's field list is capped to a sample so one pathological file
 * with thousands of distinct keys can't make the mapping UI unusable — see
 * design.md's "JSON input shape" decision. */
const JSON_FIELD_SAMPLE_LIMIT = 200;

export function isParseFileError(
  result: ParsedFile | ParseFileError,
): result is ParseFileError {
  return "message" in result;
}

/** True for a name ending in `.csv` or `.json` (case-insensitive) — checked
 * before parsing so a visitor picking the wrong file gets a clear message
 * instead of a confusing parse failure (relevant now that the file input
 * no longer restricts by `accept`, for mobile file-picker compatibility —
 * see ImportFileStep.tsx). */
export function hasSupportedExtension(fileName: string): boolean {
  const lower = fileName.toLowerCase();
  return lower.endsWith(".csv") || lower.endsWith(".json");
}

/** Detects CSV vs. JSON by extension and parses accordingly. */
export async function parseImportFile(
  file: File,
): Promise<ParsedFile | ParseFileError> {
  if (file.name.toLowerCase().endsWith(".json")) {
    return parseJSONFile(file);
  }
  return parseCSVFile(file);
}

function parseCSVFile(file: File): Promise<ParsedFile | ParseFileError> {
  return new Promise((resolve) => {
    Papa.parse<ParsedRow>(file, {
      header: true,
      skipEmptyLines: true,
      complete: (results) => {
        const fields = results.meta.fields ?? [];
        if (fields.length === 0) {
          resolve({ message: "csvNoHeaderRow" });
          return;
        }
        resolve({ rows: results.data, fields, ignoredFields: [] });
      },
      error: (error) => resolve({ message: error.message }),
    });
  });
}

/** A `{ value, precision[, currency] }` shaped amount, e.g.
 * `{"value": -3595, "precision": 2, "currency": "EUR"}` meaning `-35.95` —
 * a common structured-export shape. Recognized by shape, not by key name,
 * so it's matched under any field name. */
function asAmountObject(
  raw: unknown,
): { value: number; precision: number; currency?: string } | null {
  if (typeof raw !== "object" || raw === null || Array.isArray(raw)) {
    return null;
  }
  const obj = raw as Record<string, unknown>;
  const value = obj["value"];
  const precision = obj["precision"];
  if (typeof value !== "number" || typeof precision !== "number") {
    return null;
  }
  if (
    !Number.isFinite(value) ||
    !Number.isInteger(precision) ||
    precision < 0
  ) {
    return null;
  }
  return typeof obj["currency"] === "string"
    ? { value, precision, currency: obj["currency"] }
    : { value, precision };
}

async function parseJSONFile(file: File): Promise<ParsedFile | ParseFileError> {
  let text: string;
  try {
    text = await file.text();
  } catch {
    return { message: "fileReadError" };
  }

  let parsed: unknown;
  try {
    parsed = JSON.parse(text);
  } catch {
    return { message: "jsonInvalid" };
  }

  if (!Array.isArray(parsed)) {
    return { message: "jsonNotArray" };
  }

  const rows: ParsedRow[] = [];
  const fields: string[] = [];
  const seenFields = new Set<string>();
  const ignoredFields = new Set<string>();

  function addField(key: string) {
    if (!seenFields.has(key) && fields.length < JSON_FIELD_SAMPLE_LIMIT) {
      seenFields.add(key);
      fields.push(key);
    }
  }

  for (const item of parsed) {
    if (typeof item !== "object" || item === null || Array.isArray(item)) {
      return { message: "jsonNotFlatObjects" };
    }
    const row: ParsedRow = {};
    for (const [key, value] of Object.entries(item)) {
      if (value === null || value === undefined || typeof value !== "object") {
        row[key] = value === null || value === undefined ? "" : String(value);
        addField(key);
        continue;
      }
      const amount = asAmountObject(value);
      if (amount) {
        row[key] = (amount.value / 10 ** amount.precision).toFixed(
          amount.precision,
        );
        addField(key);
        if (amount.currency !== undefined) {
          const currencyKey = `${key}.currency`;
          row[currencyKey] = amount.currency;
          addField(currencyKey);
        }
        continue;
      }
      // An array or an unrecognized nested object: skip just this field
      // (not the whole file) and flag it for the mapping-step hint.
      ignoredFields.add(key);
    }
    rows.push(row);
  }

  return { rows, fields, ignoredFields: [...ignoredFields].sort() };
}
