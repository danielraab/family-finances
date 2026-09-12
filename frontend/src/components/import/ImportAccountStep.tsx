import type { FormEvent } from "react";
import { useTranslation } from "react-i18next";
import type { components } from "../../api/schema";

type Account = components["schemas"]["Account"];

const inputClass =
  "rounded-md border border-black/15 bg-transparent px-3 py-2 text-sm font-normal outline-none transition-colors focus:border-black/40 dark:border-white/15 dark:focus:border-white/40";

/**
 * Step 1: choose the target account. Offered accounts mirror
 * `entries.new.tsx`'s `selectableAccounts` — the visitor's own non-deleted
 * accounts plus every shared account they hold at least `append`
 * permission on; a `view`-only account is never offered, since import
 * creates entries.
 */
export function ImportAccountStep({
  accounts,
  value,
  onChange,
  onContinue,
}: {
  accounts: Account[];
  value: string;
  onChange: (accountId: string) => void;
  onContinue: () => void;
}) {
  const { t } = useTranslation();
  const selectable = accounts.filter((a) => a.permission !== "view");

  function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (value) onContinue();
  }

  return (
    <form onSubmit={handleSubmit} className="flex flex-col gap-4">
      <label className="flex flex-col gap-1.5 text-sm font-medium">
        {t("entries.import.steps.account.label")}
        <select
          value={value}
          onChange={(e) => onChange(e.target.value)}
          className={inputClass}
          required
        >
          <option value="" disabled>
            {t("entries.form.accountPlaceholder")}
          </option>
          {selectable.map((a) => (
            <option key={a.id} value={a.id}>
              {a.title} ({a.currency})
            </option>
          ))}
        </select>
      </label>

      {selectable.length === 0 && (
        <p className="text-sm text-zinc-500 dark:text-zinc-400">
          {t("entries.import.steps.account.none")}
        </p>
      )}

      <div>
        <button
          type="submit"
          disabled={!value}
          className="rounded-md bg-zinc-900 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-zinc-700 disabled:opacity-60 dark:bg-white dark:text-zinc-900 dark:hover:bg-zinc-200"
        >
          {t("entries.import.continue")}
        </button>
      </div>
    </form>
  );
}
