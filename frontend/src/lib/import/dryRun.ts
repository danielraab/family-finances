import { type ClassifiedRow, classifyRow, type RowMapping } from "./mapRow";
import { type ResolvedSeparators, resolveSeparators } from "./parseAmount";
import type { ParsedRow } from "./parseFile";

export type DryRunSummary = {
  rows: ClassifiedRow[];
  okCount: number;
  suspiciousCount: number;
  failedCount: number;
  separators: ResolvedSeparators;
};

function amountSampleColumns(mapping: RowMapping): string[] {
  return mapping.amount.mode === "single"
    ? [mapping.amount.column]
    : [mapping.amount.debitColumn, mapping.amount.creditColumn];
}

/** Resolves any "auto" amount separator setting from a sample of the mapped
 * amount column(s) across every row — the one resolution both the dry run
 * and the mapping step's live single-row preview use, so they can never
 * disagree about what a given value parses to. */
export function resolveAmountSeparatorsForRows(
  rows: ParsedRow[],
  mapping: RowMapping,
): ResolvedSeparators {
  const samples = rows
    .flatMap((row) =>
      amountSampleColumns(mapping).map((column) => row[column] ?? ""),
    )
    .filter((value) => value.trim() !== "");
  return resolveSeparators(
    mapping.amount.decimalSeparator,
    mapping.amount.thousandsSeparator,
    samples,
  );
}

/**
 * Resolves any "auto" amount separator setting from a sample of the mapped
 * amount column(s) across every row, then classifies every row against
 * that one resolved setting — entirely client-side, no network. This is
 * the whole dry run: `ClassifiedRow.entry` (set on every non-failed row)
 * is the exact payload the import step later submits, so there is no
 * second pass that could disagree with this one (see design.md's shared
 * `mapRow` decision).
 */
export function runDryRun(
  rows: ParsedRow[],
  mapping: RowMapping,
): DryRunSummary {
  const separators = resolveAmountSeparatorsForRows(rows, mapping);

  let okCount = 0;
  let suspiciousCount = 0;
  let failedCount = 0;
  const classified = rows.map((row, index) => {
    const result = classifyRow(row, index, mapping, separators);
    if (result.classification === "ok") okCount++;
    else if (result.classification === "suspicious") suspiciousCount++;
    else failedCount++;
    return result;
  });

  return {
    rows: classified,
    okCount,
    suspiciousCount,
    failedCount,
    separators,
  };
}
