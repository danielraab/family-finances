import { type FormEvent, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { api } from "../api/client";
import type { components } from "../api/schema";
import { compact } from "../lib/compact";

type AccountType = components["schemas"]["AccountType"];
type AccountCreate = components["schemas"]["AccountCreate"];

export type AccountFormValues = {
  title: string;
  description: string;
  type_id: string;
  currency: string;
  financial_institute: string;
  opening_date: string;
  closing_date: string;
};

export const emptyAccountForm: AccountFormValues = {
  title: "",
  description: "",
  type_id: "",
  currency: "",
  financial_institute: "",
  opening_date: "",
  closing_date: "",
};

const inputClass =
  "rounded-md border border-black/15 bg-transparent px-3 py-2 text-sm font-normal outline-none transition-colors focus:border-black/40 dark:border-white/15 dark:focus:border-white/40";

function validate(values: AccountFormValues): string | null {
  if (values.title.trim() === "") return "title";
  if (values.type_id === "") return "type_id";
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
  const [types, setTypes] = useState<AccountType[]>([]);
  const [invalidField, setInvalidField] = useState<string | null>(null);

  useEffect(() => {
    api.GET("/api/account-types").then(({ data }) => {
      if (data) setTypes(data);
    });
  }, []);

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
    onSubmit({
      title: values.title.trim(),
      type_id: values.type_id,
      currency: values.currency,
      opening_date: values.opening_date,
      ...compact({
        description: values.description.trim() || undefined,
        financial_institute: values.financial_institute.trim() || undefined,
        closing_date: values.closing_date || undefined,
      }),
    });
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

      <label className="flex flex-col gap-1.5 text-sm font-medium">
        {t("accounts.form.type")}
        <select
          value={values.type_id}
          onChange={(event) => set("type_id", event.target.value)}
          className={inputClass}
          required
        >
          <option value="" disabled>
            {t("accounts.form.typePlaceholder")}
          </option>
          {types.map((type) => (
            <option key={type.id} value={type.id}>
              {type.name}
            </option>
          ))}
        </select>
        {invalidField === "type_id" && (
          <span className="text-xs font-normal text-red-600 dark:text-red-400">
            {t("accounts.form.typeRequired")}
          </span>
        )}
      </label>

      <div className="flex gap-4">
        <label className="flex flex-1 flex-col gap-1.5 text-sm font-medium">
          {t("accounts.form.currency")}
          <input
            value={values.currency}
            maxLength={3}
            onChange={(event) =>
              set("currency", event.target.value.toUpperCase())
            }
            className={`${inputClass} uppercase`}
            required
          />
          {invalidField === "currency" && (
            <span className="text-xs font-normal text-red-600 dark:text-red-400">
              {t("accounts.form.currencyInvalid")}
            </span>
          )}
        </label>

        <label className="flex flex-1 flex-col gap-1.5 text-sm font-medium">
          {t("accounts.form.financialInstitute")}
          <input
            value={values.financial_institute}
            onChange={(event) => set("financial_institute", event.target.value)}
            className={inputClass}
          />
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
