/**
 * Amounts are stored (and travel over the API) as integers at a fixed 4
 * decimal places — never configurable, see backend/internal/entry. Display
 * rounding is a separate, per-user preference (`displayed_decimal_places`,
 * default 2) that only affects how an amount is *shown*; the create/edit
 * forms always work at full stored precision so a display preference can
 * never truncate what's saved.
 */
export const STORED_DECIMAL_PLACES = 4;

/** Converts a stored integer amount to its numeric value in major units. */
export function amountToNumber(amount: number): number {
  return amount / 10 ** STORED_DECIMAL_PLACES;
}

/**
 * Formats a stored integer amount as a localized currency string, rounded
 * to `displayedDecimalPlaces` — for read-only display only (account
 * balances, entry list rows).
 */
export function formatAmount(
  amount: number,
  currency: string,
  displayedDecimalPlaces: number,
  locale: string,
): string {
  try {
    return new Intl.NumberFormat(locale, {
      style: "currency",
      currency,
      minimumFractionDigits: displayedDecimalPlaces,
      maximumFractionDigits: displayedDecimalPlaces,
    }).format(amountToNumber(amount));
  } catch {
    return amountToNumber(amount).toFixed(displayedDecimalPlaces);
  }
}

/** Renders a stored integer amount at full precision for an edit input. */
export function amountToInput(amount: number): string {
  return amountToNumber(amount).toFixed(STORED_DECIMAL_PLACES);
}

/** Parses a full-precision edit input back into a stored integer amount. */
export function inputToAmount(input: string): number | null {
  const trimmed = input.trim();
  if (trimmed === "") return null;
  const n = Number(trimmed);
  if (!Number.isFinite(n)) return null;
  return Math.round(n * 10 ** STORED_DECIMAL_PLACES);
}
