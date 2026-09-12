import { inputToAmount } from "../amount";

export type DecimalSeparator = "auto" | "." | ",";
export type ThousandsSeparator = "auto" | "none" | "," | "." | " " | "'";

type ResolvedDecimalSeparator = "." | ",";
type ResolvedThousandsSeparator = "none" | "," | "." | " " | "'";

export type ResolvedSeparators = {
  decimalSeparator: ResolvedDecimalSeparator;
  thousandsSeparator: ResolvedThousandsSeparator;
};

export type AmountMapping =
  | {
      mode: "single";
      column: string;
      decimalSeparator: DecimalSeparator;
      thousandsSeparator: ThousandsSeparator;
      flipSign: boolean;
    }
  | {
      mode: "split";
      debitColumn: string;
      creditColumn: string;
      decimalSeparator: DecimalSeparator;
      thousandsSeparator: ThousandsSeparator;
    };

/**
 * Strips the thousands separator and rewrites the decimal separator to `.`,
 * so the result can be handed to `inputToAmount` — the same stored-integer
 * conversion the manual entry form already uses (see design.md's amount
 * mapping decision).
 */
function normalizeAmountString(
  raw: string,
  { decimalSeparator, thousandsSeparator }: ResolvedSeparators,
): string {
  let s = raw.trim();
  if (thousandsSeparator !== "none") {
    s = s.split(thousandsSeparator).join("");
  }
  if (decimalSeparator === ",") {
    s = s.replace(",", ".");
  }
  return s;
}

export function parseSingleAmount(
  raw: string,
  separators: ResolvedSeparators,
  flipSign: boolean,
): number | null {
  const amount = inputToAmount(normalizeAmountString(raw, separators));
  if (amount === null) return null;
  return flipSign ? -amount : amount;
}

/** `credit − debit`; a blank cell in either column counts as zero. */
export function parseSplitAmount(
  debitRaw: string,
  creditRaw: string,
  separators: ResolvedSeparators,
): number | null {
  const debit =
    debitRaw.trim() === ""
      ? 0
      : inputToAmount(normalizeAmountString(debitRaw, separators));
  const credit =
    creditRaw.trim() === ""
      ? 0
      : inputToAmount(normalizeAmountString(creditRaw, separators));
  if (debit === null || credit === null) return null;
  return credit - debit;
}

/**
 * Majority vote across sample values: whichever of `.`/`,` appears in the
 * rightmost position followed by 1-2 digits most often is taken as the
 * decimal separator, the other as the thousands separator — see
 * design.md's amount-mapping decision. Ties default to `.`/`,`.
 */
export function autoDetectSeparators(samples: string[]): ResolvedSeparators {
  let dotVotes = 0;
  let commaVotes = 0;
  for (const raw of samples) {
    const s = raw.trim();
    const dotIndex = s.lastIndexOf(".");
    const commaIndex = s.lastIndexOf(",");
    const sep =
      dotIndex > commaIndex ? "." : commaIndex > dotIndex ? "," : null;
    if (!sep) continue;
    const rightmostIndex = sep === "." ? dotIndex : commaIndex;
    const trailingDigits = s.length - rightmostIndex - 1;
    if (trailingDigits < 1 || trailingDigits > 2) continue;
    if (sep === ".") dotVotes++;
    else commaVotes++;
  }
  return commaVotes > dotVotes
    ? { decimalSeparator: ",", thousandsSeparator: "." }
    : { decimalSeparator: ".", thousandsSeparator: "," };
}

/** Resolves an "auto" decimal/thousands setting against sample values;
 * a non-"auto" setting always wins over detection. */
export function resolveSeparators(
  decimalSeparator: DecimalSeparator,
  thousandsSeparator: ThousandsSeparator,
  samples: string[],
): ResolvedSeparators {
  if (decimalSeparator !== "auto" && thousandsSeparator !== "auto") {
    return { decimalSeparator, thousandsSeparator };
  }
  const detected = autoDetectSeparators(samples);
  return {
    decimalSeparator:
      decimalSeparator === "auto"
        ? detected.decimalSeparator
        : decimalSeparator,
    thousandsSeparator:
      thousandsSeparator === "auto"
        ? detected.thousandsSeparator
        : thousandsSeparator,
  };
}
