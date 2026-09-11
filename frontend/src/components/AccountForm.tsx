import { type FormEvent, useEffect, useId, useState } from "react";
import { useTranslation } from "react-i18next";
import { api } from "../api/client";
import type { components } from "../api/schema";
import { compact } from "../lib/compact";
import { CurrencySelect } from "./CurrencySelect";
import { IconColorPicker } from "./IconColorPicker";

type AccountCreate = components["schemas"]["AccountCreate"];
type Account = components["schemas"]["Account"];

/**
 * Starter `type` autocomplete suggestions offered to every visitor, as
 * `accounts.form.types.*` i18n keys so the suggested (and, once picked,
 * submitted) text is in the visitor's own language. Merged with the
 * visitor's own in-use values from `GET /api/account-types`.
 */
const DEFAULT_ACCOUNT_TYPE_KEYS = [
  "checking",
  "savings",
  "cash",
  "creditCard",
  "prepaidCard",
  "loan",
  "mortgage",
  "investment",
  "brokerage",
  "retirement",
  "business",
  "insurance",
  "misc",
] as const;

export type AccountFormValues = {
  title: string;
  description: string;
  icon: string;
  color: string;
  type: string;
  currency: string;
  financial_institute: string;
  opening_date: string;
  closing_date: string;
};

export const emptyAccountForm: AccountFormValues = {
  title: "",
  description: "",
  icon: "",
  color: "",
  type: "",
  currency: "",
  financial_institute: "",
  opening_date: "",
  closing_date: "",
};

const inputClass =
  "rounded-md border border-black/15 bg-transparent px-3 py-2 text-sm font-normal outline-none transition-colors focus:border-black/40 dark:border-white/15 dark:focus:border-white/40";

function validate(values: AccountFormValues): string | null {
  if (values.title.trim() === "") return "title";
  if (values.type.trim() === "") return "type";
  if (!/^[A-Z]{3}$/.test(values.currency)) return "currency";
  if (values.opening_date === "") return "opening_date";
  if (values.closing_date !== "" && values.closing_date < values.opening_date) {
    return "closing_date";
  }
  return null;
}

/** Shared create/edit form for accounts.new.tsx and accounts.$accountId.edit.tsx. */
export function AccountForm({
  initial,
  submitLabel,
  submitting,
  onSubmit,
  serverError,
}: {
  initial: AccountFormValues;
  submitLabel: string;
  submitting: boolean;
  onSubmit: (values: AccountCreate) => void;
  serverError: string | null;
}) {
  const { t } = useTranslation();
  const [values, setValues] = useState(initial);
  const [typeSuggestions, setTypeSuggestions] = useState<string[]>([]);
  const [accounts, setAccounts] = useState<Account[]>([]);
  const [invalidField, setInvalidField] = useState<string | null>(null);
  const typeListId = useId();
  const currencyId = useId();

  useEffect(() => {
    Promise.all([api.GET("/api/account-types"), api.GET("/api/accounts")]).then(
      ([typesRes, accountsRes]) => {
        if (typesRes.data) setTypeSuggestions(typesRes.data);
        if (accountsRes.data) setAccounts(accountsRes.data);
      },
    );
  }, []);

  const institutes = Array.from(
    new Set(
      accounts
        .map((account) => account.financial_institute?.trim())
        .filter((value): value is string => !!value),
    ),
  ).sort((a, b) => a.localeCompare(b));
  const instituteSuggestions = institutes.filter((institute) =>
    institute
      .toLowerCase()
      .includes(values.financial_institute.trim().toLowerCase()),
  );

  // Default labels first, then any distinct in-use values not already in
  // that list — deduped by exact string match.
  const defaultAccountTypes = DEFAULT_ACCOUNT_TYPE_KEYS.map((key) =>
    t(`accounts.form.types.${key}`),
  );
  const typeOptions = Array.from(
    new Set([...defaultAccountTypes, ...typeSuggestions]),
  );

  function set<K extends keyof AccountFormValues>(
    key: K,
    value: AccountFormValues[K],
  ) {
    setValues((prev) => ({ ...prev, [key]: value }));
  }

  function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const invalid = validate(values);
    setInvalidField(invalid);
    if (invalid) return;
    const body: AccountCreate = {
      title: values.title.trim(),
      type: values.type.trim(),
      currency: values.currency,
      opening_date: values.opening_date,
      // Always sent: "" leaves the field unset on create and clears it on
      // edit, so the picker can reset a previously-chosen icon/colour.
      icon: values.icon,
      color: values.color,
      ...compact({
        description: values.description.trim() || undefined,
        financial_institute: values.financial_institute.trim() || undefined,
        closing_date: values.closing_date || undefined,
      }),
    };
    onSubmit(body);
  }

  return (
    <form onSubmit={handleSubmit} className="flex flex-col gap-4">
      <label className="flex flex-col gap-1.5 text-sm font-medium">
        {t("accounts.form.title")}
        <input
          value={values.title}
          onChange={(event) => set("title", event.target.value)}
          className={inputClass}
          required
        />
        {invalidField === "title" && (
          <span className="text-xs font-normal text-red-600 dark:text-red-400">
            {t("accounts.form.titleRequired")}
          </span>
        )}
      </label>

      <label className="flex flex-col gap-1.5 text-sm font-medium">
        {t("accounts.form.description")}
        <textarea
          value={values.description}
          onChange={(event) => set("description", event.target.value)}
          className={`${inputClass} min-h-16`}
        />
      </label>

      <IconColorPicker
        value={{ icon: values.icon, color: values.color }}
        onChange={(next) => {
          set("icon", next.icon);
          set("color", next.color);
        }}
      />

      <label className="flex flex-col gap-1.5 text-sm font-medium">
        {t("accounts.form.type")}
        <input
          value={values.type}
          onChange={(event) => set("type", event.target.value)}
          className={inputClass}
          list={typeListId}
          required
        />
        <datalist id={typeListId}>
          {typeOptions.map((option) => (
            <option key={option} value={option} />
          ))}
        </datalist>
        {invalidField === "type" && (
          <span className="text-xs font-normal text-red-600 dark:text-red-400">
            {t("accounts.form.typeRequired")}
          </span>
        )}
      </label>

      <div className="flex flex-col gap-4 sm:flex-row">
        <label
          htmlFor={currencyId}
          className="flex flex-1 flex-col gap-1.5 text-sm font-medium"
        >
          {t("accounts.form.currency")}
          <CurrencySelect
            id={currencyId}
            value={values.currency}
            onChange={(value) => set("currency", value)}
            className={inputClass}
            placeholder={t("accounts.form.currencyPlaceholder")}
            required
          />
          {invalidField === "currency" && (
            <span className="text-xs font-normal text-red-600 dark:text-red-400">
              {t("accounts.form.currencyInvalid")}
            </span>
          )}
        </label>

        <label className="group flex flex-1 flex-col gap-1.5 text-sm font-medium">
          {t("accounts.form.financialInstitute")}
          <input
            value={values.financial_institute}
            onChange={(event) => set("financial_institute", event.target.value)}
            className={inputClass}
          />
          {instituteSuggestions.length > 0 && (
            <div className="hidden flex-wrap gap-1.5 group-focus-within:flex">
              {instituteSuggestions.map((institute) => (
                <button
                  key={institute}
                  type="button"
                  onClick={() => set("financial_institute", institute)}
                  className="rounded-full border border-black/10 px-2 py-0.5 text-xs text-zinc-600 hover:bg-black/[.04] dark:border-white/10 dark:text-zinc-400 dark:hover:bg-white/[.06]"
                >
                  {institute}
                </button>
              ))}
            </div>
          )}
        </label>
      </div>

      <div className="flex gap-4">
        <label className="flex flex-1 flex-col gap-1.5 text-sm font-medium">
          {t("accounts.form.openingDate")}
          <input
            type="date"
            value={values.opening_date}
            onChange={(event) => set("opening_date", event.target.value)}
            className={inputClass}
            required
          />
        </label>

        <label className="flex flex-1 flex-col gap-1.5 text-sm font-medium">
          {t("accounts.form.closingDate")}
          <input
            type="date"
            value={values.closing_date}
            onChange={(event) => set("closing_date", event.target.value)}
            className={inputClass}
          />
          {invalidField === "closing_date" && (
            <span className="text-xs font-normal text-red-600 dark:text-red-400">
              {t("accounts.form.closingDateInvalid")}
            </span>
          )}
        </label>
      </div>

      {serverError && (
        <p className="text-sm text-red-600 dark:text-red-400">{serverError}</p>
      )}

      <div>
        <button
          type="submit"
          disabled={submitting}
          className="rounded-md bg-zinc-900 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-zinc-700 disabled:opacity-60 dark:bg-white dark:text-zinc-900 dark:hover:bg-zinc-200"
        >
          {submitLabel}
        </button>
      </div>
    </form>
  );
}
