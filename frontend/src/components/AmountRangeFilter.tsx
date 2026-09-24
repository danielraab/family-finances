import { Popover, PopoverButton, PopoverPanel } from "@headlessui/react";
import { ChevronDown } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { amountToNumber, inputToAmount } from "../lib/amount";
import { filterControlClass } from "./FilterPanel";

type AmountRange = {
  amount_from?: number | undefined;
  amount_to?: number | undefined;
};

function amountDraft(amount: number | undefined): string {
  if (amount === undefined) return "";
  return amountToNumber(amount)
    .toFixed(4)
    .replace(/\.?0+$/, "");
}

function positiveDraft(value: string): string {
  return value.replaceAll("-", "").replaceAll("−", "");
}

function trimDraft(value: string): string {
  const amount = inputToAmount(value);
  return amount === null ? value : amountDraft(amount);
}

function summaryFor(
  amountFrom: number | undefined,
  amountTo: number | undefined,
  t: ReturnType<typeof useTranslation>["t"],
): string {
  const from = amountDraft(amountFrom);
  const to = amountDraft(amountTo);
  if (from && to) return `${from} – ${to}`;
  if (from) return t("entries.filters.amountFromOnly", { amount: from });
  if (to) return t("entries.filters.amountToOnly", { amount: to });
  return t("entries.filters.amountPlaceholder");
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
  const summary = summaryFor(amountFrom, amountTo, t);
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
    <div className="flex flex-col gap-1 text-xs font-medium text-zinc-500 dark:text-zinc-400">
      {t("entries.filters.amountRange")}
      <Popover className="relative">
        <PopoverButton
          className={`${filterControlClass} flex min-w-40 items-center gap-2 text-left data-[open]:border-black/40 dark:data-[open]:border-white/40`}
        >
          <span className="flex-1 truncate text-zinc-900 dark:text-zinc-100">
            {summary}
          </span>
          <ChevronDown
            size={14}
            className="shrink-0 text-zinc-400"
            aria-hidden="true"
          />
        </PopoverButton>
        <PopoverPanel
          anchor="bottom start"
          className="z-[60] mt-1 w-72 rounded-lg border border-black/10 bg-white p-3 shadow-xl dark:border-white/15 dark:bg-neutral-900"
        >
          <div className="grid grid-cols-2 gap-3">
            <label className="flex flex-col gap-1 text-xs font-medium text-zinc-500 dark:text-zinc-400">
              {t("entries.filters.amountFrom")}
              <input
                type="text"
                inputMode="decimal"
                className={filterControlClass}
                value={amountFromDraft}
                onChange={(e) => changeFrom(e.target.value)}
                onBlur={() => setAmountFromDraft((draft) => trimDraft(draft))}
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
                onBlur={() => setAmountToDraft((draft) => trimDraft(draft))}
              />
            </label>
          </div>
        </PopoverPanel>
      </Popover>
    </div>
  );
}
