import { type FormEvent, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { api } from "../api/client";
import type { components } from "../api/schema";
import { inputToAmount } from "../lib/amount";
import { flattenCategoryTree } from "../lib/categoryTree";
import {
  CUSTOM_PRESET_KEY,
  matchPreset,
  RECURRENCE_PRESETS,
} from "../lib/recurrence";
import { resolveTagIds } from "../lib/resolveTags";
import { LocationField } from "./LocationField";
import { SignedAmountInput } from "./SignedAmountInput";
import { TagInput } from "./TagInput";

type Account = components["schemas"]["Account"];
type Category = components["schemas"]["Category"];
type Tag = components["schemas"]["Tag"];
type IntervalUnit = components["schemas"]["IntervalUnit"];
type RecurringKind = components["schemas"]["RecurringTransactionKind"];

export type RecurringTransactionFormValues = {
  account_id: string;
  to_account_id: string;
  kind: RecurringKind;
  title: string;
  description: string;
  category_id: string;
  counterparty: string;
  location: string;
  tagNames: string[];
  amountMagnitude: string;
  negative: boolean;
  interval_unit: IntervalUnit;
  interval_count: string;
  starts_on: string;
  ends_on: string;
};

export const emptyRecurringTransactionForm: RecurringTransactionFormValues = {
  account_id: "",
  to_account_id: "",
  kind: "transaction",
  title: "",
  description: "",
  category_id: "",
  counterparty: "",
  location: "",
  tagNames: [],
  amountMagnitude: "",
  negative: true,
  interval_unit: "month",
  interval_count: "1",
  starts_on: "",
  ends_on: "",
};

/** The values RecurringTransactionForm hands its parent on submit, after
 * local validation and tag-name-to-id resolution — the parent builds the
 * actual POST/PATCH body, since create/update differ slightly in how an
 * empty ends_on is represented (omitted vs. explicit null). */
export type RecurringTransactionSubmitValues = {
  account_id: string;
  /** Empty unless kind is self_transfer — the parent omits it otherwise. */
  to_account_id: string;
  kind: RecurringKind;
  title: string;
  description: string;
  category_id: string;
  counterparty: string;
  location: string;
  tag_ids: string[];
  amount: number;
  interval_unit: IntervalUnit;
  interval_count: number;
  starts_on: string;
  ends_on: string;
};

const inputClass =
  "rounded-md border border-black/15 bg-transparent px-3 py-2 text-sm font-normal outline-none transition-colors focus:border-black/40 dark:border-white/15 dark:focus:border-white/40";

/** Shared create/edit form for recurring.new.tsx and recurring.$id.edit.tsx. */
export function RecurringTransactionForm({
  initial,
  submitLabel,
  submitting,
  serverError,
  onSubmit,
  accountLocked,
  kindLocked,
}: {
  initial: RecurringTransactionFormValues;
  submitLabel: string;
  submitting: boolean;
  serverError: string | null;
  onSubmit: (values: RecurringTransactionSubmitValues) => void;
  /** True once the caller has append+ on at most one account — locks the
   * field the same way entries.new.tsx locks a preset account_id. */
  accountLocked?: boolean;
  /** True on the edit form: kind and to_account_id are immutable after
   * creation (the backend rejects either on PATCH), so both are shown as
   * read-only rather than as controls that would fail on save. */
  kindLocked?: boolean;
}) {
  const { t } = useTranslation();
  const [values, setValues] = useState(initial);
  const [presetKey, setPresetKey] = useState(() =>
    matchPreset(initial.interval_unit, Number(initial.interval_count) || 1),
  );
  const [accounts, setAccounts] = useState<Account[]>([]);
  const [categories, setCategories] = useState<Category[]>([]);
  const [tags, setTags] = useState<Tag[]>([]);
  const [counterparties, setCounterparties] = useState<string[]>([]);
  const [invalidField, setInvalidField] = useState<string | null>(null);

  useEffect(() => {
    Promise.all([
      api.GET("/api/accounts"),
      api.GET("/api/categories"),
      api.GET("/api/tags"),
      api.GET("/api/entries/counterparties"),
    ]).then(([a, c, tg, cp]) => {
      setAccounts(a.data ?? []);
      setCategories(c.data ?? []);
      setTags(tg.data ?? []);
      setCounterparties(cp.data ?? []);
    });
  }, []);

  function set<K extends keyof RecurringTransactionFormValues>(
    key: K,
    value: RecurringTransactionFormValues[K],
  ) {
    setValues((prev) => ({ ...prev, [key]: value }));
  }

  function selectPreset(key: string) {
    setPresetKey(key);
    const preset = RECURRENCE_PRESETS.find((p) => p.key === key);
    if (preset) {
      setValues((prev) => ({
        ...prev,
        interval_unit: preset.unit,
        interval_count: String(preset.count),
      }));
    }
  }

  const selectableAccounts = accounts.filter((a) => a.permission !== "view");
  const account = accounts.find((a) => a.id === values.account_id);
  // The same predicate entries.new.tsx applies to its own to-account
  // picker: append+, not disabled, not the source account, same currency.
  const toAccountOptions = accounts.filter(
    (a) =>
      a.permission !== "view" &&
      !a.disabled &&
      a.id !== values.account_id &&
      (account === undefined || a.currency === account.currency),
  );
  const isSelfTransfer = values.kind === "self_transfer";
  // Editing a self-transfer template requires append+ on both of its
  // accounts, still held now — the backend rejects the save outright
  // otherwise, so the form says why up front rather than letting the
  // visitor fill it in and fail. An account the caller has no permission
  // on at all is simply absent from /api/accounts.
  const canAppend = (id: string) =>
    accounts.some((a) => a.id === id && a.permission !== "view");
  const readOnly =
    kindLocked === true &&
    isSelfTransfer &&
    accounts.length > 0 &&
    (!canAppend(values.account_id) || !canAppend(values.to_account_id));
  const categoryOptions = flattenCategoryTree(
    categories
      .filter((c) => !c.disabled && c.permission !== "view")
      .map((c) => {
        if (!c.shared) return c;
        const { parent_id, ...rest } = c;
        return rest;
      }),
  );

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!values.account_id) {
      setInvalidField("account_id");
      return;
    }
    if (values.title.trim() === "") {
      setInvalidField("title");
      return;
    }
    if (isSelfTransfer && !values.to_account_id) {
      setInvalidField("to_account_id");
      return;
    }
    // A category is required for a transaction and optional for a
    // self-transfer, mirroring what the entry form already enforces.
    if (!isSelfTransfer && !values.category_id) {
      setInvalidField("category_id");
      return;
    }
    const magnitude = inputToAmount(values.amountMagnitude);
    if (magnitude === null || magnitude === 0) {
      setInvalidField("amount");
      return;
    }
    const intervalCount = Number(values.interval_count);
    if (!Number.isInteger(intervalCount) || intervalCount <= 0) {
      setInvalidField("interval_count");
      return;
    }
    if (values.starts_on === "") {
      setInvalidField("starts_on");
      return;
    }
    if (values.ends_on !== "" && values.ends_on < values.starts_on) {
      setInvalidField("ends_on");
      return;
    }
    setInvalidField(null);

    const tagIds = await resolveTagIds(values.tagNames, tags);

    onSubmit({
      account_id: values.account_id,
      to_account_id: isSelfTransfer ? values.to_account_id : "",
      kind: values.kind,
      title: values.title.trim(),
      description: values.description.trim(),
      category_id: values.category_id,
      // A self-transfer rejects both outright, so they are never sent for
      // that kind however the form was filled in before switching.
      counterparty: isSelfTransfer ? "" : values.counterparty.trim(),
      location: isSelfTransfer ? "" : values.location.trim(),
      tag_ids: tagIds,
      amount: values.negative ? -magnitude : magnitude,
      interval_unit: values.interval_unit,
      interval_count: intervalCount,
      starts_on: values.starts_on,
      ends_on: values.ends_on,
    });
  }

  return (
    <form onSubmit={handleSubmit} className="flex flex-col gap-4">
      <label className="flex flex-col gap-1.5 text-sm font-medium">
        {t("entries.form.account")}
        <select
          value={values.account_id}
          onChange={(e) =>
            setValues((prev) => ({
              ...prev,
              account_id: e.target.value,
              // A new source account may invalidate the chosen target one
              // (different currency, or now the same account), so the
              // choice is reset rather than preserved — the entry form
              // does exactly this for the same reason.
              to_account_id: "",
            }))
          }
          className={inputClass}
          disabled={accountLocked}
          required
        >
          <option value="" disabled>
            {t("entries.form.accountPlaceholder")}
          </option>
          {selectableAccounts.map((a) => (
            <option key={a.id} value={a.id}>
              {a.title} ({a.currency})
            </option>
          ))}
        </select>
        {invalidField === "account_id" && (
          <span className="text-xs font-normal text-red-600 dark:text-red-400">
            {t("entries.form.accountRequired")}
          </span>
        )}
      </label>

      {kindLocked ? (
        <div className="flex flex-col gap-1.5 text-sm font-medium">
          {t("entries.form.kind")}
          <p className="text-sm font-normal text-zinc-500 dark:text-zinc-400">
            {isSelfTransfer
              ? t("entries.kind.selfTransfer")
              : t("entries.kind.transaction")}
          </p>
        </div>
      ) : (
        <fieldset className="flex flex-col gap-1.5 text-sm font-medium">
          {t("entries.form.kind")}
          <div className="flex flex-wrap gap-x-4 gap-y-1 text-sm font-normal">
            <label className="flex items-center gap-1.5">
              <input
                type="radio"
                checked={values.kind === "transaction"}
                onChange={() => set("kind", "transaction")}
              />
              {t("entries.kind.transaction")}
            </label>
            <label className="flex items-center gap-1.5">
              <input
                type="radio"
                checked={isSelfTransfer}
                onChange={() => set("kind", "self_transfer")}
              />
              {t("entries.kind.selfTransfer")}
            </label>
          </div>
        </fieldset>
      )}

      {isSelfTransfer &&
        (kindLocked ? (
          <div className="flex flex-col gap-1.5 text-sm font-medium">
            {t("entries.form.toAccount")}
            <p className="text-sm font-normal text-zinc-500 dark:text-zinc-400">
              {accounts.find((a) => a.id === values.to_account_id)?.title ??
                "—"}
            </p>
          </div>
        ) : (
          <label className="flex flex-col gap-1.5 text-sm font-medium">
            {t("entries.form.toAccount")}
            <select
              value={values.to_account_id}
              onChange={(e) => set("to_account_id", e.target.value)}
              className={inputClass}
              required
            >
              <option value="" disabled>
                {t("entries.form.toAccountPlaceholder")}
              </option>
              {toAccountOptions.map((a) => (
                <option key={a.id} value={a.id}>
                  {a.title} ({a.currency})
                </option>
              ))}
            </select>
            {invalidField === "to_account_id" && (
              <span className="text-xs font-normal text-red-600 dark:text-red-400">
                {t("entries.form.toAccountRequired")}
              </span>
            )}
          </label>
        ))}

      <div className="flex flex-col gap-1.5 text-sm font-medium">
        {t("entries.form.amount", { currency: account?.currency ?? "" })}
        <SignedAmountInput
          magnitude={values.amountMagnitude}
          onMagnitudeChange={(v) => set("amountMagnitude", v)}
          negative={values.negative}
          onNegativeChange={(v) => set("negative", v)}
          currency={account?.currency ?? ""}
          invalid={invalidField === "amount"}
        />
        {invalidField === "amount" && (
          <span className="text-xs font-normal text-red-600 dark:text-red-400">
            {t("entries.form.amountInvalid")}
          </span>
        )}
      </div>

      <label className="flex flex-col gap-1.5 text-sm font-medium">
        {t("entries.form.title")}
        <input
          value={values.title}
          onChange={(e) => set("title", e.target.value)}
          className={inputClass}
          required
        />
        {invalidField === "title" && (
          <span className="text-xs font-normal text-red-600 dark:text-red-400">
            {t("entries.form.titleRequired")}
          </span>
        )}
      </label>

      <label className="flex flex-col gap-1.5 text-sm font-medium">
        {t("entries.form.description")}
        <textarea
          value={values.description}
          onChange={(e) => set("description", e.target.value)}
          className={`${inputClass} min-h-16`}
        />
      </label>

      <label className="flex flex-col gap-1.5 text-sm font-medium">
        {t("entries.form.category")}
        <select
          value={values.category_id}
          onChange={(e) => set("category_id", e.target.value)}
          className={inputClass}
        >
          <option value="">{t("entries.form.categoryPlaceholder")}</option>
          {categoryOptions.map((c) => (
            <option key={c.id} value={c.id}>
              {c.label}
              {c.shared &&
                ` — ${t("categories.shared.badgeTitle", { owner: c.ownerName ?? "" })}`}
            </option>
          ))}
        </select>
        {invalidField === "category_id" && (
          <span className="text-xs font-normal text-red-600 dark:text-red-400">
            {t("entries.form.categoryRequired")}
          </span>
        )}
      </label>

      {!isSelfTransfer && (
        <label className="flex flex-col gap-1.5 text-sm font-medium">
          {t("entries.form.counterparty")}
          <input
            list="recurring-counterparty-suggestions"
            value={values.counterparty}
            onChange={(e) => set("counterparty", e.target.value)}
            placeholder={t("entries.form.counterpartyPlaceholder")}
            className={inputClass}
          />
          <datalist id="recurring-counterparty-suggestions">
            {counterparties.map((c) => (
              <option key={c} value={c} />
            ))}
          </datalist>
        </label>
      )}

      {!isSelfTransfer && (
        <div className="flex flex-col gap-1.5 text-sm font-medium">
          {t("entries.form.location")}
          <LocationField
            value={values.location}
            onChange={(v) => set("location", v)}
            inputClassName={inputClass}
          />
        </div>
      )}

      <div className="flex flex-col gap-1.5 text-sm font-medium">
        {t("entries.form.tags")}
        <TagInput
          value={values.tagNames}
          onChange={(v) => set("tagNames", v)}
          existingTags={tags.filter(
            (tg) => !tg.disabled && tg.permission !== "view",
          )}
        />
      </div>

      <fieldset className="flex flex-col gap-3 rounded-md border border-black/10 p-3 dark:border-white/10">
        <legend className="px-1 text-sm font-medium">
          {t("recurring.form.recurrenceLegend")}
        </legend>

        <label className="flex flex-col gap-1.5 text-sm font-medium">
          {t("recurring.form.frequency")}
          <select
            value={presetKey}
            onChange={(e) => selectPreset(e.target.value)}
            className={inputClass}
          >
            {RECURRENCE_PRESETS.map((p) => (
              <option key={p.key} value={p.key}>
                {t(`recurring.presets.${p.key}`)}
              </option>
            ))}
            <option value={CUSTOM_PRESET_KEY}>
              {t("recurring.presets.custom")}
            </option>
          </select>
        </label>

        {presetKey === CUSTOM_PRESET_KEY && (
          <div className="flex items-end gap-2 text-sm font-medium">
            <label className="flex flex-col gap-1.5">
              {t("recurring.form.everyLabel")}
              <input
                type="number"
                min={1}
                step={1}
                value={values.interval_count}
                onChange={(e) => set("interval_count", e.target.value)}
                className={`${inputClass} w-20`}
              />
            </label>
            <label className="flex flex-col gap-1.5">
              {t("recurring.form.intervalUnitLabel")}
              <select
                value={values.interval_unit}
                onChange={(e) =>
                  set("interval_unit", e.target.value as IntervalUnit)
                }
                className={inputClass}
              >
                <option value="day">{t("recurring.units.day")}</option>
                <option value="week">{t("recurring.units.week")}</option>
                <option value="month">{t("recurring.units.month")}</option>
                <option value="year">{t("recurring.units.year")}</option>
              </select>
            </label>
          </div>
        )}
        {invalidField === "interval_count" && (
          <span className="text-xs font-normal text-red-600 dark:text-red-400">
            {t("recurring.form.intervalCountInvalid")}
          </span>
        )}

        <label className="flex flex-col gap-1.5 text-sm font-medium">
          {t("recurring.form.startsOn")}
          <input
            type="date"
            value={values.starts_on}
            onChange={(e) => set("starts_on", e.target.value)}
            className={inputClass}
            required
          />
          {invalidField === "starts_on" && (
            <span className="text-xs font-normal text-red-600 dark:text-red-400">
              {t("recurring.form.startsOnRequired")}
            </span>
          )}
        </label>

        <label className="flex flex-col gap-1.5 text-sm font-medium">
          {t("recurring.form.endsOn")}
          <input
            type="date"
            value={values.ends_on}
            onChange={(e) => set("ends_on", e.target.value)}
            className={inputClass}
          />
          {invalidField === "ends_on" && (
            <span className="text-xs font-normal text-red-600 dark:text-red-400">
              {t("recurring.form.endsOnBeforeStarts")}
            </span>
          )}
        </label>
      </fieldset>

      {readOnly && (
        <p className="text-sm text-amber-700 dark:text-amber-400">
          {t("recurring.form.selfTransferReadOnly")}
        </p>
      )}

      {serverError && (
        <p className="text-sm text-red-600 dark:text-red-400">{serverError}</p>
      )}

      <div>
        <button
          type="submit"
          disabled={submitting || readOnly}
          className="rounded-md bg-zinc-900 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-zinc-700 disabled:opacity-60 dark:bg-white dark:text-zinc-900 dark:hover:bg-zinc-200"
        >
          {submitLabel}
        </button>
      </div>
    </form>
  );
}
