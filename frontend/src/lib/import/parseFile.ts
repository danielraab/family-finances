import Papa from "papaparse";

/** One row from a parsed import file, values always normalized to strings. */
export type ParsedRow = Record<string, string>;

export type ParsedFile = {
  rows: ParsedRow[];
  /** Mappable column/field names, in first-seen order. */
  fields: string[];
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
        resolve({ rows: results.data, fields });
      },
      error: (error) => resolve({ message: error.message }),
    });
  });
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

  for (const item of parsed) {
    if (typeof item !== "object" || item === null || Array.isArray(item)) {
      return { message: "jsonNotFlatObjects" };
    }
    const row: ParsedRow = {};
    for (const [key, value] of Object.entries(item)) {
      if (value !== null && typeof value === "object") {
        return { message: "jsonNotFlatObjects" };
      }
      row[key] = value === null || value === undefined ? "" : String(value);
      if (!seenFields.has(key) && fields.length < JSON_FIELD_SAMPLE_LIMIT) {
        seenFields.add(key);
        fields.push(key);
      }
    }
    rows.push(row);
  }

  return { rows, fields };
}
