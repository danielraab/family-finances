/** "auto" tries ISO/`Date.parse`; anything else is a token format string
 * (`YYYY`, `MM`, `DD`, `HH`, `mm`, `ss`) — see design.md's date-mapping
 * decision. */
export type DateFormatSetting = "auto" | string;

const FORMAT_TOKEN = /YYYY|MM|DD|HH|mm|ss/g;

function escapeRegExp(s: string): string {
  return s.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}

function parseAuto(raw: string): Date | null {
  const s = raw.trim();
  if (!s) return null;
  const ms = Date.parse(s);
  if (Number.isNaN(ms)) return null;
  return new Date(ms);
}

/** Parses raw against an explicit token format, rejecting anything that
 * doesn't match the format's shape or isn't a real calendar date (e.g.
 * `31.02.2026`). */
function parseWithFormat(raw: string, format: string): Date | null {
  const tokens: string[] = [];
  let pattern = "";
  let lastIndex = 0;
  let match: RegExpExecArray | null;
  FORMAT_TOKEN.lastIndex = 0;
  // biome-ignore lint/suspicious/noAssignInExpressions: standard exec loop
  while ((match = FORMAT_TOKEN.exec(format)) !== null) {
    pattern += escapeRegExp(format.slice(lastIndex, match.index));
    const token = match[0];
    tokens.push(token);
    pattern += token === "YYYY" ? "(\\d{4})" : "(\\d{1,2})";
    lastIndex = FORMAT_TOKEN.lastIndex;
  }
  pattern += escapeRegExp(format.slice(lastIndex));
  if (tokens.length === 0 || !tokens.includes("YYYY")) return null;

  const m = new RegExp(`^${pattern}$`).exec(raw.trim());
  if (!m) return null;

  const parts: Partial<Record<string, number>> = {};
  tokens.forEach((token, i) => {
    parts[token] = Number(m[i + 1]);
  });

  const year = parts["YYYY"];
  if (year === undefined) return null;
  const monthIndex = (parts["MM"] ?? 1) - 1;
  const day = parts["DD"] ?? 1;
  const hours = parts["HH"] ?? 0;
  const minutes = parts["mm"] ?? 0;
  const seconds = parts["ss"] ?? 0;

  const date = new Date(year, monthIndex, day, hours, minutes, seconds);
  if (
    date.getFullYear() !== year ||
    date.getMonth() !== monthIndex ||
    date.getDate() !== day ||
    date.getHours() !== hours ||
    date.getMinutes() !== minutes
  ) {
    return null;
  }
  return date;
}

export function parseBookingDate(
  raw: string,
  format: DateFormatSetting,
): Date | null {
  return format === "auto" ? parseAuto(raw) : parseWithFormat(raw, format);
}

const AMBIGUOUS_DATE_RE = /^(\d{1,2})[./-](\d{1,2})[./-]\d{2,4}$/;

/** True when raw looks like a `D?D<sep>M?M<sep>YYYY`-shaped date where both
 * day-first and month-first readings are plausible (both parts <= 12) —
 * flagged as suspicious only when date format is "auto", since an explicit
 * format already resolves the ambiguity. */
export function isAmbiguousDate(raw: string): boolean {
  const m = AMBIGUOUS_DATE_RE.exec(raw.trim());
  if (!m) return false;
  const a = Number(m[1]);
  const b = Number(m[2]);
  return a >= 1 && a <= 12 && b >= 1 && b <= 12 && a !== b;
}
