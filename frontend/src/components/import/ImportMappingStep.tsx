import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import type { components } from "../../api/schema";
import { flattenCategoryTree } from "../../lib/categoryTree";
import {
  collectFieldExamples,
  truncateExample,
} from "../../lib/import/formatExample";
import type {
  DecimalSeparator,
  ThousandsSeparator,
} from "../../lib/import/parseAmount";
import type { ParsedRow } from "../../lib/import/parseFile";
import { TagInput } from "../TagInput";
import { type MappingState, toRowMapping } from "./types";

type Category = components["schemas"]["Category"];
type Tag = components["schemas"]["Tag"];

const inputClass =
  "rounded-md border border-black/15 bg-transparent px-3 py-2 text-sm font-normal outline-none transition-colors focus:border-black/40 dark:border-white/15 dark:focus:border-white/40";

const DECIMAL_OPTIONS: DecimalSeparator[] = ["auto", ".", ","];
const THOUSANDS_OPTIONS: ThousandsSeparator[] = [
  "auto",
  "none",
  ",",
  ".",
  " ",
  "'",
];

function FieldSelect({
  label,
  value,
  fields,
  fieldExamples,
  onChange,
  required,
  allowUnmapped,
}: {
  label: string;
  value: string;
  fields: string[];
  fieldExamples: Record<string, string>;
  onChange: (value: string) => void;
  required?: boolean;
  allowUnmapped?: boolean;
}) {
  const { t } = useTranslation();
  return (
    <label className="flex flex-col gap-1.5 text-sm font-medium">
      {label}
      <select
        value={value}
        onChange={(e) => onChange(e.target.value)}
        className={inputClass}
        required={required}
      >
        <option value="" disabled={!allowUnmapped}>
          {allowUnmapped
            ? t("entries.import.notMapped")
            : t("entries.import.steps.mapping.selectColumn")}
        </option>
        {fields.map((f) => {
          const example = fieldExamples[f];
          return (
            <option key={f} value={f}>
              {example ? `${f} — ${truncateExample(example)}` : f}
            </option>
          );
        })}
      </select>
    </label>
  );
}

/**
 * Step 3: map source columns/fields to entry fields (each option showing an
 * example value from the file, for orientation), set the batch category/
 * tags, and tune amount and date parsing. "Continue" moves to the dry-run
 * step (`ImportDryRunStep`), which computes and shows results — this step
 * only gates that transition on every required field being mapped and a
 * category being selected.
 */
export function ImportMappingStep({
  fields,
  rows,
  ignoredFields,
  categories,
  tags,
  mapping,
  onMappingChange,
  onContinue,
  onBack,
}: {
  fields: string[];
  rows: ParsedRow[];
  /** Top-level JSON fields skipped because their value was a complex
   * structure we don't know how to map — see `ParsedFile.ignoredFields`.
   * Always empty for a CSV import. */
  ignoredFields: string[];
  categories: Category[];
  tags: Tag[];
  mapping: MappingState;
  onMappingChange: (next: MappingState) => void;
  onContinue: () => void;
  onBack: () => void;
}) {
  const { t } = useTranslation();

  function update<K extends keyof MappingState>(
    key: K,
    value: MappingState[K],
  ) {
    onMappingChange({ ...mapping, [key]: value });
  }

  const fieldExamples = useMemo(
    () => collectFieldExamples(fields, rows),
    [fields, rows],
  );

  const categoryOptions = flattenCategoryTree(
    categories
      .filter((c) => !c.disabled && c.permission !== "view")
      .map((c) => {
        if (!c.shared) return c;
        const { parent_id, ...rest } = c;
        return rest;
      }),
  );

  const canContinue = toRowMapping(mapping) !== null && rows.length > 0;

  return (
    <div className="flex flex-col gap-6">
      {ignoredFields.length > 0 && (
        <p className="rounded-md border border-amber-600/30 bg-amber-50 px-3 py-2 text-sm text-amber-800 dark:border-amber-400/30 dark:bg-amber-950/40 dark:text-amber-300">
          {t("entries.import.steps.mapping.ignoredFieldsHint", {
            fields: ignoredFields.join(", "),
          })}
        </p>
      )}

      <div className="flex flex-col gap-4">
        <h2 className="text-sm font-semibold text-zinc-500 dark:text-zinc-400">
          {t("entries.import.steps.mapping.fieldsHeading")}
        </h2>
        <FieldSelect
          label={t("entries.form.title")}
          value={mapping.titleColumn}
          fields={fields}
          fieldExamples={fieldExamples}
          onChange={(v) => update("titleColumn", v)}
          required
        />

        <fieldset className="flex flex-col gap-2 text-sm font-medium">
          {t("entries.import.steps.mapping.amountHeading")}
          <div className="flex gap-4 text-sm font-normal">
            <label className="flex items-center gap-1.5">
              <input
                type="radio"
                checked={mapping.amountMode === "single"}
                onChange={() => update("amountMode", "single")}
              />
              {t("entries.import.steps.mapping.amountSingle")}
            </label>
            <label className="flex items-center gap-1.5">
              <input
                type="radio"
                checked={mapping.amountMode === "split"}
                onChange={() => update("amountMode", "split")}
              />
              {t("entries.import.steps.mapping.amountSplit")}
            </label>
          </div>

          {mapping.amountMode === "single" ? (
            <>
              <FieldSelect
                label={t("entries.import.steps.mapping.amountColumn")}
                value={mapping.amountColumn}
                fields={fields}
                fieldExamples={fieldExamples}
                onChange={(v) => update("amountColumn", v)}
                required
              />
              <label className="flex items-center gap-1.5 text-sm font-normal">
                <input
                  type="checkbox"
                  checked={mapping.flipSign}
                  onChange={(e) => update("flipSign", e.target.checked)}
                />
                {t("entries.import.steps.mapping.flipSign")}
              </label>
            </>
          ) : (
            <>
              <FieldSelect
                label={t("entries.import.steps.mapping.debitColumn")}
                value={mapping.debitColumn}
                fields={fields}
                fieldExamples={fieldExamples}
                onChange={(v) => update("debitColumn", v)}
                required
              />
              <FieldSelect
                label={t("entries.import.steps.mapping.creditColumn")}
                value={mapping.creditColumn}
                fields={fields}
                fieldExamples={fieldExamples}
                onChange={(v) => update("creditColumn", v)}
                required
              />
            </>
          )}

          <div className="flex flex-wrap gap-4">
            <label className="flex flex-col gap-1.5 text-sm font-normal">
              {t("entries.import.steps.mapping.decimalSeparator")}
              <select
                value={mapping.decimalSeparator}
                onChange={(e) =>
                  update("decimalSeparator", e.target.value as DecimalSeparator)
                }
                className={inputClass}
              >
                {DECIMAL_OPTIONS.map((s) => (
                  <option key={s} value={s}>
                    {s === "auto" ? t("entries.import.steps.mapping.auto") : s}
                  </option>
                ))}
              </select>
            </label>
            <label className="flex flex-col gap-1.5 text-sm font-normal">
              {t("entries.import.steps.mapping.thousandsSeparator")}
              <select
                value={mapping.thousandsSeparator}
                onChange={(e) =>
                  update(
                    "thousandsSeparator",
                    e.target.value as ThousandsSeparator,
                  )
                }
                className={inputClass}
              >
                {THOUSANDS_OPTIONS.map((s) => (
                  <option key={s} value={s}>
                    {s === "auto"
                      ? t("entries.import.steps.mapping.auto")
                      : s === "none"
                        ? t("entries.import.steps.mapping.none")
                        : s}
                  </option>
                ))}
              </select>
            </label>
          </div>
        </fieldset>

        <FieldSelect
          label={t("entries.form.bookingTimestamp")}
          value={mapping.bookingColumn}
          fields={fields}
          fieldExamples={fieldExamples}
          onChange={(v) => update("bookingColumn", v)}
          required
        />
        <label className="flex flex-col gap-1.5 text-sm font-medium">
          {t("entries.import.steps.mapping.dateFormat")}
          <input
            value={mapping.dateFormat}
            onChange={(e) => update("dateFormat", e.target.value)}
            placeholder="auto"
            className={inputClass}
          />
          <span className="text-xs font-normal text-zinc-500 dark:text-zinc-400">
            {t("entries.import.steps.mapping.dateFormatHint")}
          </span>
        </label>

        <FieldSelect
          label={t("entries.form.description")}
          value={mapping.descriptionColumn}
          fields={fields}
          fieldExamples={fieldExamples}
          onChange={(v) => update("descriptionColumn", v)}
          allowUnmapped
        />
        <FieldSelect
          label={t("entries.form.counterparty")}
          value={mapping.counterpartyColumn}
          fields={fields}
          fieldExamples={fieldExamples}
          onChange={(v) => update("counterpartyColumn", v)}
          allowUnmapped
        />
        <FieldSelect
          label={t("entries.form.location")}
          value={mapping.locationColumn}
          fields={fields}
          fieldExamples={fieldExamples}
          onChange={(v) => update("locationColumn", v)}
          allowUnmapped
        />
      </div>

      <div className="flex flex-col gap-4">
        <h2 className="text-sm font-semibold text-zinc-500 dark:text-zinc-400">
          {t("entries.import.steps.mapping.batchHeading")}
        </h2>
        <label className="flex flex-col gap-1.5 text-sm font-medium">
          {t("entries.form.category")}
          <select
            value={mapping.categoryId}
            onChange={(e) => update("categoryId", e.target.value)}
            className={inputClass}
            required
          >
            <option value="" disabled>
              {t("entries.form.categoryPlaceholder")}
            </option>
            {categoryOptions.map((c) => (
              <option key={c.id} value={c.id}>
                {c.label}
                {c.shared &&
                  ` — ${t("categories.shared.badgeTitle", { owner: c.ownerName ?? "" })}`}
              </option>
            ))}
          </select>
        </label>

        <div className="flex flex-col gap-1.5 text-sm font-medium">
          {t("entries.form.tags")}
          <TagInput
            value={mapping.tagNames}
            onChange={(names) => update("tagNames", names)}
            existingTags={tags.filter(
              (tg) => !tg.disabled && tg.permission !== "view",
            )}
          />
        </div>
      </div>

      <div className="flex gap-3">
        <button
          type="button"
          onClick={onBack}
          className="text-sm font-medium text-zinc-600 underline underline-offset-2 hover:text-zinc-900 dark:text-zinc-400 dark:hover:text-zinc-100"
        >
          {t("entries.import.back")}
        </button>
        <button
          type="button"
          onClick={onContinue}
          disabled={!canContinue}
          className="rounded-md bg-zinc-900 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-zinc-700 disabled:opacity-60 dark:bg-white dark:text-zinc-900 dark:hover:bg-zinc-200"
        >
          {t("entries.import.continue")}
        </button>
      </div>
    </div>
  );
}
