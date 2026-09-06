import { useTranslation } from "react-i18next";

const baseInputClass =
  "w-full rounded-r-md border border-l-0 border-black/15 bg-transparent px-3 py-2 text-sm font-normal outline-none transition-colors focus:border-black/40 dark:border-white/15 dark:focus:border-white/40";

const negativeInputClass =
  "border-red-200 bg-red-100 text-red-800 focus:border-red-400 dark:border-red-900/50 dark:bg-red-900/40 dark:text-red-300 dark:focus:border-red-700";

const positiveInputClass =
  "border-emerald-200 bg-emerald-100 text-emerald-800 focus:border-emerald-400 dark:border-emerald-900/50 dark:bg-emerald-900/40 dark:text-emerald-300 dark:focus:border-emerald-700";

const negativeButtonClass = "bg-red-600 text-white hover:bg-red-700";
const positiveButtonClass = "bg-emerald-600 text-white hover:bg-emerald-700";

/**
 * A magnitude-only amount input with a sign toggle button in front of it.
 * The sign lives entirely in `negative`/`onNegativeChange` — a typed or
 * pasted "-" is intercepted in `onChange` (forcing the sign to negative
 * rather than toggling it, so it's a no-op when already negative) and never
 * reaches the field's text.
 */
export function SignedAmountInput({
  magnitude,
  onMagnitudeChange,
  negative,
  onNegativeChange,
  currency,
  invalid,
}: {
  magnitude: string;
  onMagnitudeChange: (value: string) => void;
  negative: boolean;
  onNegativeChange: (negative: boolean) => void;
  currency: string;
  invalid?: boolean;
}) {
  const { t } = useTranslation();
  const isZero = (Number(magnitude) || 0) === 0;

  function handleChange(raw: string) {
    if (raw.includes("-")) {
      onNegativeChange(true);
      raw = raw.replaceAll("-", "");
    }
    onMagnitudeChange(raw);
  }

  return (
    <div className="flex">
      <button
        type="button"
        onClick={() => onNegativeChange(!negative)}
        aria-label={
          negative
            ? t("entries.form.amountSignSwitchToPositive")
            : t("entries.form.amountSignSwitchToNegative")
        }
        className={`flex w-10 flex-none items-center justify-center rounded-l-md text-base font-semibold transition-colors ${
          negative ? negativeButtonClass : positiveButtonClass
        }`}
      >
        {negative ? "−" : "+"}
      </button>
      <input
        value={magnitude}
        onChange={(e) => handleChange(e.target.value)}
        inputMode="decimal"
        placeholder="0.00"
        aria-label={t("entries.form.amount", { currency })}
        aria-invalid={invalid}
        className={`${baseInputClass} ${
          isZero ? "" : negative ? negativeInputClass : positiveInputClass
        }`}
      />
    </div>
  );
}
