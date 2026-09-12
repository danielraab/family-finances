import {
  type AmountMapping,
  parseSingleAmount,
  parseSplitAmount,
  type ResolvedSeparators,
} from "./parseAmount";
import {
  type DateFormatSetting,
  isAmbiguousDate,
  parseBookingDate,
} from "./parseDate";
import type { ParsedRow } from "./parseFile";

export type RowMapping = {
  titleColumn: string;
  amount: AmountMapping;
  bookingColumn: string;
  dateFormat: DateFormatSetting;
  /** null means "not mapped" — the entry field stays empty. */
  descriptionColumn: string | null;
  counterpartyColumn: string | null;
  locationColumn: string | null;
  categoryId: string;
};

/** A ready `POST /api/entries` body for a transaction, minus `account_id`
 * and `tag_ids` — filled in by the caller (see ImportRunStep). */
export type MappedEntryDraft = {
  kind: "transaction";
  amount: number;
  booking_timestamp: string;
  title: string;
  description?: string;
  counterparty?: string;
  location?: string;
  category_id: string;
};

/** field/reason pairs — reason is a short code, translated by the caller. */
export type RowIssue = { field: string; reason: string };

export type MapRowResult =
  | { ok: true; entry: MappedEntryDraft }
  | { ok: false; errors: RowIssue[] };

/**
 * The single implementation of "is this row valid and what entry does it
 * produce" — used identically by the dry run and the real import loop, so
 * the two can never disagree (see design.md's "one shared mapRow function"
 * decision).
 */
export function mapRow(
  row: ParsedRow,
  mapping: RowMapping,
  amountSeparators: ResolvedSeparators,
): MapRowResult {
  const errors: RowIssue[] = [];

  const title = (row[mapping.titleColumn] ?? "").trim();
  if (!title) errors.push({ field: "title", reason: "missingTitle" });

  const amount =
    mapping.amount.mode === "single"
      ? parseSingleAmount(
          row[mapping.amount.column] ?? "",
          amountSeparators,
          mapping.amount.flipSign,
        )
      : parseSplitAmount(
          row[mapping.amount.debitColumn] ?? "",
          row[mapping.amount.creditColumn] ?? "",
          amountSeparators,
        );
  if (amount === null)
    errors.push({ field: "amount", reason: "invalidAmount" });

  const rawDate = row[mapping.bookingColumn] ?? "";
  const parsedDate = parseBookingDate(rawDate, mapping.dateFormat);
  if (!parsedDate) {
    errors.push({ field: "booking_timestamp", reason: "invalidDate" });
  }

  if (!mapping.categoryId) {
    errors.push({ field: "category_id", reason: "missingCategory" });
  }

  if (errors.length > 0 || amount === null || !parsedDate) {
    return { ok: false, errors };
  }

  const entry: MappedEntryDraft = {
    kind: "transaction",
    amount,
    booking_timestamp: parsedDate.toISOString(),
    title,
    category_id: mapping.categoryId,
  };
  if (mapping.descriptionColumn) {
    const v = (row[mapping.descriptionColumn] ?? "").trim();
    if (v) entry.description = v;
  }
  if (mapping.counterpartyColumn) {
    const v = (row[mapping.counterpartyColumn] ?? "").trim();
    if (v) entry.counterparty = v;
  }
  if (mapping.locationColumn) {
    const v = (row[mapping.locationColumn] ?? "").trim();
    if (v) entry.location = v;
  }
  return { ok: true, entry };
}

export type RowClassification = "ok" | "suspicious" | "failed";

export type ClassifiedRow = {
  row: ParsedRow;
  index: number;
  classification: RowClassification;
  issues: RowIssue[];
  entry: MappedEntryDraft | null;
};

/**
 * Layers the suspicious checks (zero amount; an ambiguous date when date
 * format is "auto") on top of an already-successful mapRow result — see
 * design.md's dry-run classification diagram. A failed row is never also
 * checked for suspicion; its mapRow errors are the whole story.
 */
export function classifyRow(
  row: ParsedRow,
  index: number,
  mapping: RowMapping,
  amountSeparators: ResolvedSeparators,
): ClassifiedRow {
  const result = mapRow(row, mapping, amountSeparators);
  if (!result.ok) {
    return {
      row,
      index,
      classification: "failed",
      issues: result.errors,
      entry: null,
    };
  }

  const issues: RowIssue[] = [];
  if (result.entry.amount === 0) {
    issues.push({ field: "amount", reason: "zeroAmount" });
  }
  if (
    mapping.dateFormat === "auto" &&
    isAmbiguousDate(row[mapping.bookingColumn] ?? "")
  ) {
    issues.push({ field: "booking_timestamp", reason: "ambiguousDate" });
  }

  return {
    row,
    index,
    classification: issues.length > 0 ? "suspicious" : "ok",
    issues,
    entry: result.entry,
  };
}
