import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { amountToInput, inputToAmount } from "../lib/amount";
import { filterControlClass } from "./FilterPanel";

type AmountRange = {
  amount_from?: number | undefined;
  amount_to?: number | undefined;
};

function amountDraft(amount: number | undefined): string {
  return amount === undefined ? "" : amountToInput(amount);
}

function positiveDraft(value: string): string {
  return value.replaceAll("-", "").replaceAll("−", "");
}

/**
 * The paired entry-ledger amount controls. Amounts are entered as positive
 * magnitudes; the backend applies the resulting stored-scale bounds to either
 * sign of an entry amount.
 */
export function AmountRangeFilter({
  amountFrom,
  amountTo,
  onChange,
}: {
  amountFrom: number | undefined;
  amountTo: number | undefined;
  onChange: (range: AmountRange) => void;
}) {
  const { t } = useTranslation();
  const [amountFromDraft, setAmountFromDraft] = useState(() =>
    amountDraft(amountFrom),
  );
  const [amountToDraft, setAmountToDraft] = useState(() =>
    amountDraft(amountTo),
  );
  const lastEmitted = useRef<AmountRange>({
    amount_from: amountFrom,
    amount_to: amountTo,
  });

  useEffect(() => {
    if (amountFrom !== lastEmitted.current.amount_from) {
      setAmountFromDraft(amountDraft(amountFrom));
    }
    if (amountTo !== lastEmitted.current.amount_to) {
      setAmountToDraft(amountDraft(amountTo));
    }
    lastEmitted.current = { amount_from: amountFrom, amount_to: amountTo };
  }, [amountFrom, amountTo]);

  function changeFrom(value: string) {
    const draft = positiveDraft(value);
    const nextAmountFrom = inputToAmount(draft) ?? undefined;
    setAmountFromDraft(draft);
    const range = { amount_from: nextAmountFrom, amount_to: amountTo };
    lastEmitted.current = range;
    onChange(range);
  }

  function changeTo(value: string) {
    const draft = positiveDraft(value);
    const nextAmountTo = inputToAmount(draft) ?? undefined;
    setAmountToDraft(draft);
    const range = { amount_from: amountFrom, amount_to: nextAmountTo };
    lastEmitted.current = range;
    onChange(range);
  }

  return (
    <div className="grid grid-cols-2 gap-3 sm:col-span-2">
      <label className="flex flex-col gap-1 text-xs font-medium text-zinc-500 dark:text-zinc-400">
        {t("entries.filters.amountFrom")}
        <input
          type="text"
          inputMode="decimal"
          className={filterControlClass}
          value={amountFromDraft}
          onChange={(e) => changeFrom(e.target.value)}
        />
      </label>

      <label className="flex flex-col gap-1 text-xs font-medium text-zinc-500 dark:text-zinc-400">
        {t("entries.filters.amountTo")}
        <input
          type="text"
          inputMode="decimal"
          className={filterControlClass}
          value={amountToDraft}
          onChange={(e) => changeTo(e.target.value)}
        />
      </label>
    </div>
  );
}
