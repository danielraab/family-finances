import { Fragment, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { runDryRun } from "../../lib/import/dryRun";
import {
  collectFieldExamples,
  truncateExample,
} from "../../lib/import/formatExample";
import {
  type ClassifiedRow,
  classifyRow,
  type RowMapping,
} from "../../lib/import/mapRow";
import type { ParsedRow } from "../../lib/import/parseFile";

const inputClass =
  "rounded-md border border-black/15 bg-transparent px-2 py-1 text-xs font-normal outline-none transition-colors focus:border-black/40 dark:border-white/15 dark:focus:border-white/40";

/** A per-row, per-field remap: overrides which source column feeds one
 * mapped field, for exactly this row — never the whole file's mapping.
 * `undefined` means "use the mapping this step started with." */
type RowOverride = Partial<{
  titleColumn: string;
  amountColumn: string;
  debitColumn: string;
  creditColumn: string;
  bookingColumn: string;
}>;

/** The bulk panel's local draft: one entry per remappable field, `""`
 * meaning "keep current" — the resting value every control returns to,
 * since a selection spanning many rows has no single mapping to display.
 * Only the non-empty entries become a `RowOverride` when Apply is pressed. */
type BulkDraft = {
  titleColumn: string;
  amountColumn: string;
  debitColumn: string;
  creditColumn: string;
  bookingColumn: string;
};

const EMPTY_BULK_DRAFT: BulkDraft = {
  titleColumn: "",
  amountColumn: "",
  debitColumn: "",
  creditColumn: "",
  bookingColumn: "",
};

function draftToOverride(draft: BulkDraft): RowOverride {
  const override: RowOverride = {};
  if (draft.titleColumn) override.titleColumn = draft.titleColumn;
  if (draft.amountColumn) override.amountColumn = draft.amountColumn;
  if (draft.debitColumn) override.debitColumn = draft.debitColumn;
  if (draft.creditColumn) override.creditColumn = draft.creditColumn;
  if (draft.bookingColumn) override.bookingColumn = draft.bookingColumn;
  return override;
}

function applyOverride(base: RowMapping, override: RowOverride): RowMapping {
  const amount =
    base.amount.mode === "single"
      ? { ...base.amount, column: override.amountColumn ?? base.amount.column }
      : {
          ...base.amount,
          debitColumn: override.debitColumn ?? base.amount.debitColumn,
          creditColumn: override.creditColumn ?? base.amount.creditColumn,
        };
  return {
    ...base,
    titleColumn: override.titleColumn ?? base.titleColumn,
    amount,
    bookingColumn: override.bookingColumn ?? base.bookingColumn,
  };
}

/** The raw source values behind one row's mapped fields, for the per-row
 * expansion — labeled reuse of the same field labels the mapping controls
 * use, so "why did this become that" is traceable. */
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

function RemapSelect({
  label,
  fields,
  row,
  value,
  onChange,
}: {
  label: string;
  fields: string[];
  row: ParsedRow;
  value: string;
  onChange: (value: string) => void;
}) {
  return (
    <label className="flex flex-col gap-1 text-xs font-medium">
      {label}
      <select
        value={value}
        onChange={(e) => onChange(e.target.value)}
        className={inputClass}
      >
        {fields.map((f) => {
          const example = row[f];
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

/** The bulk panel's counterpart to `RemapSelect`: same column list, but a
 * file-wide example (there is no single row to draw one from) and an
 * explicit "keep current" option instead of the current mapping as the
 * resting value. */
function BulkRemapSelect({
  label,
  fields,
  fieldExamples,
  value,
  onChange,
}: {
  label: string;
  fields: string[];
  fieldExamples: Record<string, string>;
  value: string;
  onChange: (value: string) => void;
}) {
  const { t } = useTranslation();
  return (
    <label className="flex flex-col gap-1 text-xs font-medium">
      {label}
      <select
        value={value}
        onChange={(e) => onChange(e.target.value)}
        className={inputClass}
      >
        <option value="">
          {t("entries.import.steps.dryRun.bulkRemapKeep")}
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
 * Step 4: computes the offline dry run once (from the mapping the previous
 * step produced) and lists every row, each independently click-to-expand
 * to show its source values and — for ok/suspicious rows — the entry it
 * would create. A **failed** row's expansion additionally offers a remap
 * control per failing field (title/amount/booking date): picking a
 * different source column re-classifies *only that row*, immediately,
 * against the file's already-resolved amount separators — the row-file
 * mapping everywhere else is untouched.
 *
 * Rows also carry a checkbox. A non-empty selection reveals a bulk remap
 * panel that writes the same per-row overrides across every selected row
 * at once (see design.md's "one selection, two uses"), and drives the
 * "Import selected" action beside "Import all" — each handing the wizard
 * the rows of its chosen scope, whose non-failed members are what the run
 * step submits and whose failed members are what the result step reports.
 */
export function ImportDryRunStep({
  sourceRows,
  rowMapping,
  fields,
  currency,
  onContinue,
  onBack,
}: {
  sourceRows: ParsedRow[];
  rowMapping: RowMapping;
  fields: string[];
  currency: string;
  onContinue: (rows: ClassifiedRow[]) => void;
  onBack: () => void;
}) {
  const { t } = useTranslation();

  // Lazy initializers run exactly once, on mount — the dry run is computed
  // from the mapping this step was entered with; a mapping change means
  // going Back and re-entering this step fresh, which remounts it.
  const [initial] = useState(() => runDryRun(sourceRows, rowMapping));
  const [rows, setRows] = useState<ClassifiedRow[]>(initial.rows);
  const [overrides, setOverrides] = useState<Record<number, RowOverride>>({});
  const [expandedRows, setExpandedRows] = useState<Set<number>>(new Set());
  const [hideSuccessful, setHideSuccessful] = useState(false);
  const [selected, setSelected] = useState<Set<number>>(new Set());
  const [bulkDraft, setBulkDraft] = useState<BulkDraft>(EMPTY_BULK_DRAFT);
  const selectAllRef = useRef<HTMLInputElement>(null);

  const fieldExamples = useMemo(
    () => collectFieldExamples(fields, sourceRows),
    [fields, sourceRows],
  );

  const counts = useMemo(() => {
    let ok = 0;
    let suspicious = 0;
    let failed = 0;
    for (const r of rows) {
      if (r.classification === "ok") ok++;
      else if (r.classification === "suspicious") suspicious++;
      else failed++;
    }
    return { ok, suspicious, failed };
  }, [rows]);

  function toggleExpanded(index: number) {
    setExpandedRows((current) => {
      const next = new Set(current);
      if (next.has(index)) next.delete(index);
      else next.add(index);
      return next;
    });
  }

  function handleRemap(rowIndex: number, patch: RowOverride) {
    const merged = { ...(overrides[rowIndex] ?? {}), ...patch };
    setOverrides((prev) => ({ ...prev, [rowIndex]: merged }));
    const effectiveMapping = applyOverride(rowMapping, merged);
    setRows((prev) =>
      prev.map((r) =>
        r.index === rowIndex
          ? classifyRow(r.row, r.index, effectiveMapping, initial.separators)
          : r,
      ),
    );
  }

  function toggleSelected(index: number) {
    setSelected((current) => {
      const next = new Set(current);
      if (next.has(index)) next.delete(index);
      else next.add(index);
      return next;
    });
  }

  /** Writes every *changed* bulk control onto every selected row's
   * override and re-classifies those rows in one pass — the same
   * `applyOverride` + `classifyRow` path `handleRemap` takes for a single
   * row, so a bulk remap and a per-row remap leave identical state. */
  function applyBulkRemap() {
    const patch = draftToOverride(bulkDraft);
    if (Object.keys(patch).length === 0 || selected.size === 0) return;

    const nextOverrides = { ...overrides };
    for (const index of selected) {
      nextOverrides[index] = { ...(overrides[index] ?? {}), ...patch };
    }
    setOverrides(nextOverrides);
    setRows((prev) =>
      prev.map((r) =>
        selected.has(r.index)
          ? classifyRow(
              r.row,
              r.index,
              applyOverride(rowMapping, nextOverrides[r.index] ?? {}),
              initial.separators,
            )
          : r,
      ),
    );
    setBulkDraft(EMPTY_BULK_DRAFT);
  }

  const visibleRows = hideSuccessful
    ? rows.filter((r) => r.classification !== "ok")
    : rows;

  // The header checkbox reflects the *listed* rows — what the visitor can
  // see — while the selection itself deliberately keeps rows hidden by the
  // "hide successful" toggle (see design.md).
  const listedSelectedCount = visibleRows.filter((r) =>
    selected.has(r.index),
  ).length;
  const allListedSelected =
    visibleRows.length > 0 && listedSelectedCount === visibleRows.length;

  useEffect(() => {
    if (selectAllRef.current) {
      selectAllRef.current.indeterminate =
        listedSelectedCount > 0 && !allListedSelected;
    }
  }, [listedSelectedCount, allListedSelected]);

  function toggleSelectAll() {
    // Clearing only the listed rows would leave "3 rows selected" standing
    // over a list where nothing is checked — clear the lot instead.
    if (allListedSelected) {
      setSelected(new Set());
      return;
    }
    setSelected((current) => {
      const next = new Set(current);
      for (const r of visibleRows) next.add(r.index);
      return next;
    });
  }

  const importAllCount = counts.ok + counts.suspicious;
  const selectedRows = rows.filter((r) => selected.has(r.index));
  const importSelectedCount = selectedRows.filter(
    (r) => r.classification !== "failed",
  ).length;
  const bulkDraftTouched = Object.keys(draftToOverride(bulkDraft)).length > 0;

  return (
    <div className="flex flex-col gap-4">
      <h2 className="text-sm font-semibold text-zinc-500 dark:text-zinc-400">
        {t("entries.import.steps.dryRun.heading")}
      </h2>
      <div className="flex flex-wrap items-center justify-between gap-3">
        <p className="text-sm">
          {t("entries.import.steps.dryRun.summary", {
            ok: counts.ok,
            suspicious: counts.suspicious,
            failed: counts.failed,
          })}
        </p>
        <label className="flex items-center gap-1.5 text-sm font-normal">
          <input
            type="checkbox"
            checked={hideSuccessful}
            onChange={(e) => setHideSuccessful(e.target.checked)}
          />
          {t("entries.import.steps.dryRun.hideSuccessful")}
        </label>
      </div>

      {selected.size > 0 && (
        <div className="flex flex-col gap-3 rounded-md border border-black/10 bg-black/[.02] px-3 py-3 dark:border-white/10 dark:bg-white/[.03]">
          <div className="flex flex-wrap items-center gap-3">
            <span className="text-sm font-medium">
              {t("entries.import.steps.dryRun.selectedCount", {
                count: selected.size,
              })}
            </span>
            <button
              type="button"
              onClick={() => setSelected(new Set())}
              className="text-xs font-medium text-zinc-600 underline underline-offset-2 hover:text-zinc-900 dark:text-zinc-400 dark:hover:text-zinc-100"
            >
              {t("entries.import.steps.dryRun.clearSelection")}
            </button>
          </div>
          <span className="text-xs font-semibold text-zinc-500 dark:text-zinc-400">
            {t("entries.import.steps.dryRun.bulkRemapHeading")}
          </span>
          <p className="max-w-prose text-xs text-zinc-500 dark:text-zinc-400">
            {t("entries.import.steps.dryRun.bulkRemapNote", {
              count: selected.size,
            })}
          </p>
          <div className="flex flex-wrap items-end gap-3">
            <BulkRemapSelect
              label={t("entries.form.title")}
              fields={fields}
              fieldExamples={fieldExamples}
              value={bulkDraft.titleColumn}
              onChange={(v) =>
                setBulkDraft((prev) => ({ ...prev, titleColumn: v }))
              }
            />
            {rowMapping.amount.mode === "single" ? (
              <BulkRemapSelect
                label={t("entries.import.steps.mapping.amountColumn")}
                fields={fields}
                fieldExamples={fieldExamples}
                value={bulkDraft.amountColumn}
                onChange={(v) =>
                  setBulkDraft((prev) => ({ ...prev, amountColumn: v }))
                }
              />
            ) : (
              <>
                <BulkRemapSelect
                  label={t("entries.import.steps.mapping.debitColumn")}
                  fields={fields}
                  fieldExamples={fieldExamples}
                  value={bulkDraft.debitColumn}
                  onChange={(v) =>
                    setBulkDraft((prev) => ({ ...prev, debitColumn: v }))
                  }
                />
                <BulkRemapSelect
                  label={t("entries.import.steps.mapping.creditColumn")}
                  fields={fields}
                  fieldExamples={fieldExamples}
                  value={bulkDraft.creditColumn}
                  onChange={(v) =>
                    setBulkDraft((prev) => ({ ...prev, creditColumn: v }))
                  }
                />
              </>
            )}
            <BulkRemapSelect
              label={t("entries.form.bookingTimestamp")}
              fields={fields}
              fieldExamples={fieldExamples}
              value={bulkDraft.bookingColumn}
              onChange={(v) =>
                setBulkDraft((prev) => ({ ...prev, bookingColumn: v }))
              }
            />
            <button
              type="button"
              onClick={applyBulkRemap}
              disabled={!bulkDraftTouched}
              className="rounded-md border border-black/15 px-3 py-1.5 text-xs font-medium transition-colors hover:bg-black/[.04] disabled:opacity-60 dark:border-white/15 dark:hover:bg-white/[.06]"
            >
              {t("entries.import.steps.dryRun.bulkRemapApply", {
                count: selected.size,
              })}
            </button>
          </div>
        </div>
      )}

      <div className="max-h-[32rem] overflow-y-auto overflow-x-auto rounded-md border border-black/10 dark:border-white/10">
        <table className="w-full text-left text-sm">
          <thead>
            <tr className="text-xs font-medium text-zinc-500 dark:text-zinc-400">
              <th className="py-1 pl-2 pr-2">
                <input
                  ref={selectAllRef}
                  type="checkbox"
                  checked={allListedSelected}
                  onChange={toggleSelectAll}
                  aria-label={t("entries.import.steps.dryRun.selectAll")}
                />
              </th>
              <th className="pr-4 py-1">
                {t("entries.import.steps.dryRun.rowColumn")}
              </th>
              <th className="pr-4 py-1">
                {t("entries.import.steps.dryRun.statusColumn")}
              </th>
              <th className="py-1">
                {t("entries.import.steps.dryRun.reasonColumn")}
              </th>
            </tr>
          </thead>
          <tbody>
            {visibleRows.map((r) => {
              const expanded = expandedRows.has(r.index);
              const override = overrides[r.index] ?? {};
              return (
                <Fragment key={r.index}>
                  <tr
                    onClick={() => toggleExpanded(r.index)}
                    className="cursor-pointer border-t border-black/5 hover:bg-black/[.03] dark:border-white/5 dark:hover:bg-white/[.05]"
                  >
                    <td
                      className="py-1 pl-2 pr-2"
                      onClick={(e) => e.stopPropagation()}
                      onKeyDown={(e) => e.stopPropagation()}
                    >
                      <input
                        type="checkbox"
                        checked={selected.has(r.index)}
                        onChange={() => toggleSelected(r.index)}
                        aria-label={t("entries.import.steps.dryRun.selectRow", {
                          row: r.index + 1,
                        })}
                      />
                    </td>
                    <td className="pr-4 py-1">
                      <span className="mr-1 inline-block w-3 text-zinc-400">
                        {expanded ? "▾" : "▸"}
                      </span>
                      {r.index + 1}
                    </td>
                    <td className="pr-4 py-1">
                      {r.classification === "failed" ? (
                        <span className="text-red-600 dark:text-red-400">
                          {t("entries.import.steps.dryRun.failed")}
                        </span>
                      ) : r.classification === "suspicious" ? (
                        <span className="text-amber-600 dark:text-amber-400">
                          {t("entries.import.steps.dryRun.suspicious")}
                        </span>
                      ) : (
                        <span className="text-emerald-600 dark:text-emerald-400">
                          {t("entries.import.steps.dryRun.ready")}
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
                  {expanded && (
                    <tr className="border-t border-black/5 dark:border-white/5">
                      <td
                        colSpan={4}
                        className="bg-black/[.02] px-2 py-3 dark:bg-white/[.03]"
                      >
                        <div className="flex flex-col gap-4 text-xs sm:flex-row sm:gap-8">
                          <div className="flex flex-col gap-1">
                            <span className="font-semibold text-zinc-500 dark:text-zinc-400">
                              {t(
                                "entries.import.steps.dryRun.sourceDataHeading",
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
                                  "entries.import.steps.dryRun.mappedEntryHeading",
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
                                <dd>{(r.entry.amount / 10000).toFixed(4)}</dd>
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
                          {r.classification === "failed" && (
                            <div className="flex flex-col gap-2">
                              <span className="font-semibold text-zinc-500 dark:text-zinc-400">
                                {t("entries.import.steps.dryRun.remapHeading")}
                              </span>
                              <p className="max-w-xs text-zinc-500 dark:text-zinc-400">
                                {t("entries.import.steps.dryRun.remapNote")}
                              </p>
                              <div className="flex flex-wrap gap-3">
                                {r.issues.some((i) => i.field === "title") && (
                                  <RemapSelect
                                    label={t("entries.form.title")}
                                    fields={fields}
                                    row={r.row}
                                    value={
                                      override.titleColumn ??
                                      rowMapping.titleColumn
                                    }
                                    onChange={(v) =>
                                      handleRemap(r.index, { titleColumn: v })
                                    }
                                  />
                                )}
                                {r.issues.some((i) => i.field === "amount") &&
                                  (rowMapping.amount.mode === "single" ? (
                                    <RemapSelect
                                      label={t(
                                        "entries.import.steps.mapping.amountColumn",
                                      )}
                                      fields={fields}
                                      row={r.row}
                                      value={
                                        override.amountColumn ??
                                        rowMapping.amount.column
                                      }
                                      onChange={(v) =>
                                        handleRemap(r.index, {
                                          amountColumn: v,
                                        })
                                      }
                                    />
                                  ) : (
                                    <>
                                      <RemapSelect
                                        label={t(
                                          "entries.import.steps.mapping.debitColumn",
                                        )}
                                        fields={fields}
                                        row={r.row}
                                        value={
                                          override.debitColumn ??
                                          rowMapping.amount.debitColumn
                                        }
                                        onChange={(v) =>
                                          handleRemap(r.index, {
                                            debitColumn: v,
                                          })
                                        }
                                      />
                                      <RemapSelect
                                        label={t(
                                          "entries.import.steps.mapping.creditColumn",
                                        )}
                                        fields={fields}
                                        row={r.row}
                                        value={
                                          override.creditColumn ??
                                          rowMapping.amount.creditColumn
                                        }
                                        onChange={(v) =>
                                          handleRemap(r.index, {
                                            creditColumn: v,
                                          })
                                        }
                                      />
                                    </>
                                  ))}
                                {r.issues.some(
                                  (i) => i.field === "booking_timestamp",
                                ) && (
                                  <RemapSelect
                                    label={t("entries.form.bookingTimestamp")}
                                    fields={fields}
                                    row={r.row}
                                    value={
                                      override.bookingColumn ??
                                      rowMapping.bookingColumn
                                    }
                                    onChange={(v) =>
                                      handleRemap(r.index, { bookingColumn: v })
                                    }
                                  />
                                )}
                              </div>
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
          onClick={() => onContinue(rows)}
          disabled={importAllCount === 0}
          className="rounded-md bg-zinc-900 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-zinc-700 disabled:opacity-60 dark:bg-white dark:text-zinc-900 dark:hover:bg-zinc-200"
        >
          {t("entries.import.steps.dryRun.importAll", {
            count: importAllCount,
          })}
        </button>
        <button
          type="button"
          onClick={() => onContinue(selectedRows)}
          disabled={importSelectedCount === 0}
          className="rounded-md border border-black/15 px-4 py-2 text-sm font-medium transition-colors hover:bg-black/[.04] disabled:opacity-60 dark:border-white/15 dark:hover:bg-white/[.06]"
        >
          {t("entries.import.steps.dryRun.importSelected", {
            count: importSelectedCount,
          })}
        </button>
      </div>
    </div>
  );
}
