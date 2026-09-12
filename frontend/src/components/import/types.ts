import type { RowMapping } from "../../lib/import/mapRow";
import type {
  AmountMapping,
  DecimalSeparator,
  ThousandsSeparator,
} from "../../lib/import/parseAmount";

/** UI-level mapping state the mapping step edits — a superset of
 * `RowMapping`/`AmountMapping` that also holds the not-yet-required-to-be-
 * mapped fields as plain strings (`""` meaning "unmapped") and the batch
 * tag names, before `toRowMapping` turns it into what `lib/import/` wants. */
export type MappingState = {
  titleColumn: string;
  amountMode: "single" | "split";
  amountColumn: string;
  debitColumn: string;
  creditColumn: string;
  decimalSeparator: DecimalSeparator;
  thousandsSeparator: ThousandsSeparator;
  flipSign: boolean;
  bookingColumn: string;
  dateFormat: string;
  descriptionColumn: string;
  counterpartyColumn: string;
  locationColumn: string;
  categoryId: string;
  tagNames: string[];
};

export const initialMappingState: MappingState = {
  titleColumn: "",
  amountMode: "single",
  amountColumn: "",
  debitColumn: "",
  creditColumn: "",
  decimalSeparator: "auto",
  thousandsSeparator: "auto",
  flipSign: false,
  bookingColumn: "",
  dateFormat: "auto",
  descriptionColumn: "",
  counterpartyColumn: "",
  locationColumn: "",
  categoryId: "",
  tagNames: [],
};

/** Builds a `RowMapping` from the UI state, or `null` while a required
 * field (title, amount column(s), booking date, category) isn't mapped yet
 * — gates the mapping step's "Run dry run" action. */
export function toRowMapping(state: MappingState): RowMapping | null {
  if (!state.titleColumn || !state.bookingColumn || !state.categoryId) {
    return null;
  }

  let amount: AmountMapping;
  if (state.amountMode === "single") {
    if (!state.amountColumn) return null;
    amount = {
      mode: "single",
      column: state.amountColumn,
      decimalSeparator: state.decimalSeparator,
      thousandsSeparator: state.thousandsSeparator,
      flipSign: state.flipSign,
    };
  } else {
    if (!state.debitColumn || !state.creditColumn) return null;
    amount = {
      mode: "split",
      debitColumn: state.debitColumn,
      creditColumn: state.creditColumn,
      decimalSeparator: state.decimalSeparator,
      thousandsSeparator: state.thousandsSeparator,
    };
  }

  return {
    titleColumn: state.titleColumn,
    amount,
    bookingColumn: state.bookingColumn,
    dateFormat:
      state.dateFormat.trim() === "" ? "auto" : state.dateFormat.trim(),
    descriptionColumn: state.descriptionColumn || null,
    counterpartyColumn: state.counterpartyColumn || null,
    locationColumn: state.locationColumn || null,
    categoryId: state.categoryId,
  };
}

export type RunFailure = { index: number; reason: string };

export type RunState =
  | { status: "idle" }
  | { status: "running"; total: number; created: number; failed: RunFailure[] }
  | {
      status: "done";
      total: number;
      created: number;
      failed: RunFailure[];
      canceled: boolean;
    };
