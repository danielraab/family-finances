import { Fragment, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import type { components } from "../../api/schema";
import { flattenCategoryTree } from "../../lib/categoryTree";
import { type DryRunSummary, runDryRun } from "../../lib/import/dryRun";
import type { RowMapping } from "../../lib/import/mapRow";
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

const EXAMPLE_MAX_LENGTH = 28;

function truncate(value: string): string {
  return value.length > EXAMPLE_MAX_LENGTH
    ? `${value.slice(0, EXAMPLE_MAX_LENGTH)}…`
    : value;
}

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
              {example ? `${f} — ${truncate(example)}` : f}
            </option>
          );
        })}
      </select>
    </label>
  );
}

/** The raw source values behind one row's mapped fields, for the dry-run
 * table's per-row expansion — labeled reuse of the same field labels the
 * mapping controls use, so "why did this become that" is traceable. */
function rawFieldsForRow(
  t: (key: string) => string,
  rowMapping: RowMapping,
  row: ParsedRow,
): { label: string; value: string }[] {
  const out: { label: string; value: string }[] = [
    {
      label: t("entries.form.title"),
      value: row[rowMapping.titleColumn] ?? "",
    },
  ];
  if (rowMapping.amount.mode === "single") {
    out.push({
      label: t("entries.import.steps.mapping.amountColumn"),
      value: row[rowMapping.amount.column] ?? "",
    });
  } else {
    out.push({
      label: t("entries.import.steps.mapping.debitColumn"),
      value: row[rowMapping.amount.debitColumn] ?? "",
    });
    out.push({
      label: t("entries.import.steps.mapping.creditColumn"),
      value: row[rowMapping.amount.creditColumn] ?? "",
    });
  }
  out.push({
    label: t("entries.form.bookingTimestamp"),
    value: row[rowMapping.bookingColumn] ?? "",
  });
  if (rowMapping.descriptionColumn) {
    out.push({
      label: t("entries.form.description"),
      value: row[rowMapping.descriptionColumn] ?? "",
    });
  }
  if (rowMapping.counterpartyColumn) {
    out.push({
      label: t("entries.form.counterparty"),
      value: row[rowMapping.counterpartyColumn] ?? "",
    });
  }
  if (rowMapping.locationColumn) {
    out.push({
      label: t("entries.form.location"),
      value: row[rowMapping.locationColumn] ?? "",
    });
  }
  return out;
}

/**
 * Step 3: map source columns/fields to entry fields (each option showing an
 * example value from the file, for orientation), set the batch category/
 * tags, tune amount and date parsing, and run the offline dry run over the
 * whole file. The dry run's results list every row, each expandable on
 * click to show its source values and (when it mapped successfully) the
 * resulting entry — there is no separate single-row preview. "Continue"
 * only unlocks once a dry run has completed against the *current* mapping —
 * any further mapping change clears the dry run result, forcing a re-run
 * (see design.md's "settings can be adjusted and re-validated" decision).
 */
export function ImportMappingStep({
  fields,
  rows,
  ignoredFields,
  categories,
  tags,
  currency,
  mapping,
  onMappingChange,
  dryRun,
  onDryRun,
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
  currency: string;
  mapping: MappingState;
  onMappingChange: (next: MappingState) => void;
  dryRun: DryRunSummary | null;
  onDryRun: (result: DryRunSummary) => void;
  onContinue: () => void;
  onBack: () => void;
}) {
  const { t } = useTranslation();
  const [expandedRows, setExpandedRows] = useState<Set<number>>(new Set());

  function update<K extends keyof MappingState>(
    key: K,
    value: MappingState[K],
  ) {
    onMappingChange({ ...mapping, [key]: value });
  }

  function toggleExpanded(index: number) {
    setExpandedRows((current) => {
      const next = new Set(current);
      if (next.has(index)) next.delete(index);
      else next.add(index);
      return next;
    });
  }

  const fieldExamples = useMemo(() => {
    const examples: Record<string, string> = {};
    for (const field of fields) {
      for (const row of rows) {
        const value = row[field];
        if (value && value.trim() !== "") {
          examples[field] = value;
          break;
        }
      }
    }
    return examples;
  }, [fields, rows]);

  const categoryOptions = flattenCategoryTree(
    categories
      .filter((c) => !c.disabled && c.permission !== "view")
      .map((c) => {
        if (!c.shared) return c;
        const { parent_id, ...rest } = c;
        return rest;
      }),
  );

  const rowMapping = toRowMapping(mapping);

  function handleRunDryRun() {
    if (!rowMapping) return;
    setExpandedRows(new Set());
    onDryRun(runDryRun(rows, rowMapping));
  }

  const canRunDryRun = rowMapping !== null && rows.length > 0;

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

      <div className="flex flex-col gap-3">
        <div>
          <button
            type="button"
            onClick={handleRunDryRun}
            disabled={!canRunDryRun}
            className="rounded-md border border-black/15 px-4 py-2 text-sm font-medium transition-colors hover:bg-black/[.04] disabled:opacity-60 dark:border-white/15 dark:hover:bg-white/[.06]"
          >
            {t("entries.import.steps.mapping.runDryRun")}
          </button>
        </div>

        {dryRun && (
          <div className="flex flex-col gap-3 rounded-md border border-black/10 p-4 dark:border-white/10">
            <p className="text-sm">
              {t("entries.import.steps.mapping.dryRunSummary", {
                ok: dryRun.okCount,
                suspicious: dryRun.suspiciousCount,
                failed: dryRun.failedCount,
              })}
            </p>
            <div className="max-h-[28rem] overflow-y-auto overflow-x-auto">
              <table className="w-full text-left text-sm">
                <thead>
                  <tr className="text-xs font-medium text-zinc-500 dark:text-zinc-400">
                    <th className="pr-4 py-1">
                      {t("entries.import.steps.mapping.rowColumn")}
                    </th>
                    <th className="pr-4 py-1">
                      {t("entries.import.steps.mapping.statusColumn")}
                    </th>
                    <th className="py-1">
                      {t("entries.import.steps.mapping.reasonColumn")}
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {dryRun.rows.map((r) => {
                    const expanded = expandedRows.has(r.index);
                    return (
                      <Fragment key={r.index}>
                        <tr
                          onClick={() => toggleExpanded(r.index)}
                          className="cursor-pointer border-t border-black/5 hover:bg-black/[.03] dark:border-white/5 dark:hover:bg-white/[.05]"
                        >
                          <td className="pr-4 py-1">
                            <span className="mr-1 inline-block w-3 text-zinc-400">
                              {expanded ? "▾" : "▸"}
                            </span>
                            {r.index + 1}
                          </td>
                          <td className="pr-4 py-1">
                            {r.classification === "failed" ? (
                              <span className="text-red-600 dark:text-red-400">
                                {t("entries.import.steps.mapping.failed")}
                              </span>
                            ) : r.classification === "suspicious" ? (
                              <span className="text-amber-600 dark:text-amber-400">
                                {t("entries.import.steps.mapping.suspicious")}
                              </span>
                            ) : (
                              <span className="text-emerald-600 dark:text-emerald-400">
                                {t("entries.import.steps.mapping.ready")}
                              </span>
                            )}
                          </td>
                          <td className="py-1">
                            {r.issues.length > 0
                              ? r.issues
                                  .map((issue) =>
                                    t(`entries.import.reasons.${issue.reason}`),
                                  )
                                  .join(", ")
                              : "—"}
                          </td>
                        </tr>
                        {expanded && rowMapping && (
                          <tr className="border-t border-black/5 dark:border-white/5">
                            <td
                              colSpan={3}
                              className="bg-black/[.02] px-2 py-3 dark:bg-white/[.03]"
                            >
                              <div className="flex flex-col gap-4 text-xs sm:flex-row sm:gap-8">
                                <div className="flex flex-col gap-1">
                                  <span className="font-semibold text-zinc-500 dark:text-zinc-400">
                                    {t(
                                      "entries.import.steps.mapping.sourceDataHeading",
                                    )}
                                  </span>
                                  <dl className="grid grid-cols-[auto_1fr] gap-x-3 gap-y-0.5">
                                    {rawFieldsForRow(t, rowMapping, r.row).map(
                                      (f) => (
                                        <Fragment key={f.label}>
                                          <dt className="text-zinc-500 dark:text-zinc-400">
                                            {f.label}
                                          </dt>
                                          <dd>{f.value || "—"}</dd>
                                        </Fragment>
                                      ),
                                    )}
                                  </dl>
                                </div>
                                {r.entry && (
                                  <div className="flex flex-col gap-1">
                                    <span className="font-semibold text-zinc-500 dark:text-zinc-400">
                                      {t(
                                        "entries.import.steps.mapping.previewHeading",
                                      )}
                                    </span>
                                    <dl className="grid grid-cols-[auto_1fr] gap-x-3 gap-y-0.5">
                                      <dt className="text-zinc-500 dark:text-zinc-400">
                                        {t("entries.form.title")}
                                      </dt>
                                      <dd>{r.entry.title}</dd>
                                      <dt className="text-zinc-500 dark:text-zinc-400">
                                        {t("entries.form.amount", { currency })}
                                      </dt>
                                      <dd>
                                        {(r.entry.amount / 10000).toFixed(4)}
                                      </dd>
                                      <dt className="text-zinc-500 dark:text-zinc-400">
                                        {t("entries.form.bookingTimestamp")}
                                      </dt>
                                      <dd>{r.entry.booking_timestamp}</dd>
                                      {r.entry.description && (
                                        <>
                                          <dt className="text-zinc-500 dark:text-zinc-400">
                                            {t("entries.form.description")}
                                          </dt>
                                          <dd>{r.entry.description}</dd>
                                        </>
                                      )}
                                      {r.entry.counterparty && (
                                        <>
                                          <dt className="text-zinc-500 dark:text-zinc-400">
                                            {t("entries.form.counterparty")}
                                          </dt>
                                          <dd>{r.entry.counterparty}</dd>
                                        </>
                                      )}
                                      {r.entry.location && (
                                        <>
                                          <dt className="text-zinc-500 dark:text-zinc-400">
                                            {t("entries.form.location")}
                                          </dt>
                                          <dd>{r.entry.location}</dd>
                                        </>
                                      )}
                                    </dl>
                                  </div>
                                )}
                              </div>
                            </td>
                          </tr>
                        )}
                      </Fragment>
                    );
                  })}
                </tbody>
              </table>
            </div>
          </div>
        )}
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
          disabled={!dryRun || dryRun.okCount + dryRun.suspiciousCount === 0}
          className="rounded-md bg-zinc-900 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-zinc-700 disabled:opacity-60 dark:bg-white dark:text-zinc-900 dark:hover:bg-zinc-200"
        >
          {t("entries.import.steps.mapping.startImport")}
        </button>
      </div>
    </div>
  );
}
