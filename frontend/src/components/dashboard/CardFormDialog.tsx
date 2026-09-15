import { Dialog, DialogPanel, DialogTitle } from "@headlessui/react";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { api } from "../../api/client";
import type { components } from "../../api/schema";
import { compact } from "../../lib/compact";
import type {
  DashboardCard,
  DashboardCardType,
} from "../../lib/dashboardFilter";
import type { WeekStart } from "../../lib/dateRangePresets";
import type { Account } from "../../lib/useAccountsWithBalances";
import { DateRangeFilter } from "../DateRangeFilter";

type Category = components["schemas"]["Category"];
type Tag = components["schemas"]["Tag"];

const inputClass =
  "rounded-md border border-black/15 bg-transparent px-2.5 py-1.5 text-sm font-normal outline-none transition-colors focus:border-black/40 dark:border-white/15 dark:focus:border-white/40 disabled:opacity-60";

const CARD_TYPES: DashboardCardType[] = [
  "account_stat",
  "query_stat",
  "entry_list",
  "bar_chart",
];

/**
 * Edit mode's card form, shared by "Add card" (editingCard null — starts
 * from account_stat with an empty config) and a card's own "Edit" action
 * (editingCard set — the type picker locks to that card's type, since
 * type is immutable after creation per design.md, and every other field
 * seeds from its current config). Reuses /reports' own account/category/
 * tag <select>s and DateRangeFilter for the filter-bearing types rather
 * than rebuilding equivalent controls. Submitting creates or updates the
 * card and hands the result back to the caller via onSaved.
 */
export function CardFormDialog({
  open,
  onClose,
  editingCard,
  onSaved,
  accounts,
  categories,
  tags,
  weekStart,
}: {
  open: boolean;
  onClose: () => void;
  editingCard: DashboardCard | null;
  onSaved: (card: DashboardCard) => void;
  accounts: Account[];
  categories: Category[];
  tags: Tag[];
  weekStart: WeekStart;
}) {
  const { t } = useTranslation();
  const [type, setType] = useState<DashboardCardType>("account_stat");
  const [title, setTitle] = useState("");
  const [accountId, setAccountId] = useState("");
  const [categoryId, setCategoryId] = useState("");
  const [includeSubcategories, setIncludeSubcategories] = useState(true);
  const [tagId, setTagId] = useState("");
  const [range, setRange] = useState<{
    range?: string | undefined;
    from?: string | undefined;
    to?: string | undefined;
  }>({});
  const [unit, setUnit] = useState<"month" | "day">("month");
  const [columns, setColumns] = useState(2);
  const [showRecurringPreview, setShowRecurringPreview] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);

  // Seeds the form fresh every time the dialog opens — from editingCard's
  // current config when editing, or blank defaults when adding. Keyed on
  // editingCard's id (not the object itself, which is a new identity on
  // every parent render) so an unrelated re-render while the dialog is
  // open never clobbers what the visitor is mid-typing.
  // biome-ignore lint/correctness/useExhaustiveDependencies: intentionally keyed on editingCard's id, not the object identity — see comment above.
  useEffect(() => {
    if (!open) return;
    setError(null);
    if (editingCard) {
      setType(editingCard.type);
      setTitle(editingCard.config.title ?? "");
      setAccountId(editingCard.config.account_id ?? "");
      setCategoryId(editingCard.config.category_id ?? "");
      setIncludeSubcategories(editingCard.config.include_subcategories ?? true);
      setTagId(editingCard.config.tag_id ?? "");
      setRange({
        range: editingCard.config.range?.preset,
        from: editingCard.config.range?.from,
        to: editingCard.config.range?.to,
      });
      setUnit(editingCard.config.unit === "day" ? "day" : "month");
      setColumns(editingCard.config.columns ?? 2);
      setShowRecurringPreview(
        editingCard.config.show_recurring_preview ?? false,
      );
    } else {
      setType("account_stat");
      setTitle("");
      setAccountId("");
      setCategoryId("");
      setIncludeSubcategories(true);
      setTagId("");
      setRange({});
      setUnit("month");
      setColumns(2);
      setShowRecurringPreview(false);
    }
  }, [open, editingCard?.id]);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);

    if (type === "account_stat" && !accountId) {
      setError(t("dashboard.addCard.accountRequired"));
      return;
    }

    const config = compact({
      title: type !== "account_stat" ? title.trim() || undefined : undefined,
      account_id: accountId || undefined,
      category_id:
        type !== "account_stat" ? categoryId || undefined : undefined,
      include_subcategories:
        type !== "account_stat" && categoryId
          ? includeSubcategories
          : undefined,
      tag_id: type !== "account_stat" ? tagId || undefined : undefined,
      range:
        type === "query_stat" || type === "entry_list"
          ? compact({
              preset: range.range,
              from: range.from,
              to: range.to,
            })
          : undefined,
      unit: type === "bar_chart" ? unit : undefined,
      columns: type === "entry_list" ? columns : undefined,
      show_recurring_preview:
        type === "entry_list" || type === "bar_chart"
          ? showRecurringPreview
          : undefined,
    });

    setSaving(true);
    const { data, error: apiError } = editingCard
      ? await api.PATCH("/api/dashboard/cards/{id}", {
          params: { path: { id: editingCard.id } },
          body: { config },
        })
      : await api.POST("/api/dashboard/cards", {
          body: { type, config },
        });
    setSaving(false);
    if (apiError || !data) {
      setError(t("dashboard.addCard.saveError"));
      return;
    }
    onSaved(data);
    onClose();
  }

  return (
    <Dialog open={open} onClose={onClose} className="relative z-50">
      <div className="fixed inset-0 bg-black/40" aria-hidden="true" />
      <div className="fixed inset-0 flex items-center justify-center p-4">
        <DialogPanel className="flex w-full max-w-md flex-col gap-4 rounded-lg bg-white p-6 dark:bg-neutral-900">
          <form onSubmit={submit} className="flex flex-col gap-4">
            <DialogTitle className="text-base font-semibold">
              {editingCard
                ? t("dashboard.editCard.heading")
                : t("dashboard.addCard.heading")}
            </DialogTitle>

            <label className="flex flex-col gap-1.5 text-sm font-medium">
              {t("dashboard.addCard.typeLabel")}
              <select
                value={type}
                disabled={!!editingCard}
                onChange={(e) => setType(e.target.value as DashboardCardType)}
                className={inputClass}
              >
                {CARD_TYPES.map((ct) => (
                  <option key={ct} value={ct}>
                    {t(`dashboard.cardTypes.${ct}`)}
                  </option>
                ))}
              </select>
              {editingCard && (
                <span className="text-xs font-normal text-zinc-500 dark:text-zinc-400">
                  {t("dashboard.editCard.typeImmutableHint")}
                </span>
              )}
            </label>

            {type !== "account_stat" && (
              <label className="flex flex-col gap-1.5 text-sm font-medium">
                {t("dashboard.addCard.titleLabel")}
                <input
                  type="text"
                  value={title}
                  onChange={(e) => setTitle(e.target.value)}
                  placeholder={t("dashboard.addCard.titlePlaceholder")}
                  className={inputClass}
                />
              </label>
            )}

            <label className="flex flex-col gap-1.5 text-sm font-medium">
              {type === "account_stat"
                ? t("reports.filters.account")
                : t("dashboard.addCard.accountOptional")}
              <select
                value={accountId}
                onChange={(e) => setAccountId(e.target.value)}
                className={inputClass}
              >
                <option value="">
                  {type === "account_stat"
                    ? t("entries.form.accountPlaceholder")
                    : t("reports.filters.allAccounts")}
                </option>
                {accounts.map((a) => (
                  <option key={a.id} value={a.id}>
                    {a.title}
                  </option>
                ))}
              </select>
            </label>

            {type !== "account_stat" && (
              <>
                <label className="flex flex-col gap-1.5 text-sm font-medium">
                  {t("reports.filters.category")}
                  <select
                    value={categoryId}
                    onChange={(e) => setCategoryId(e.target.value)}
                    className={inputClass}
                  >
                    <option value="">
                      {t("reports.filters.selectCategory")}
                    </option>
                    {categories.map((c) => (
                      <option key={c.id} value={c.id}>
                        {c.name}
                      </option>
                    ))}
                  </select>
                </label>

                {categoryId && (
                  <label className="flex items-center gap-2 text-sm">
                    <input
                      type="checkbox"
                      checked={includeSubcategories}
                      onChange={(e) =>
                        setIncludeSubcategories(e.target.checked)
                      }
                    />
                    {t("reports.filters.includeSubcategories")}
                  </label>
                )}

                <label className="flex flex-col gap-1.5 text-sm font-medium">
                  {t("reports.filters.tag")}
                  <select
                    value={tagId}
                    onChange={(e) => setTagId(e.target.value)}
                    className={inputClass}
                  >
                    <option value="">{t("reports.filters.selectTag")}</option>
                    {tags.map((tag) => (
                      <option key={tag.id} value={tag.id}>
                        {tag.name}
                      </option>
                    ))}
                  </select>
                </label>
              </>
            )}

            {(type === "query_stat" || type === "entry_list") && (
              <DateRangeFilter
                value={range}
                weekStart={weekStart}
                onChange={(patch) =>
                  setRange((prev) => ({ ...prev, ...patch }))
                }
                fromLabel={t("reports.filters.from")}
                toLabel={t("reports.filters.to")}
              />
            )}

            {type === "bar_chart" && (
              <label className="flex flex-col gap-1.5 text-sm font-medium">
                {t("dashboard.addCard.unitLabel")}
                <select
                  value={unit}
                  onChange={(e) => setUnit(e.target.value as "month" | "day")}
                  className={inputClass}
                >
                  <option value="month">
                    {t("dashboard.addCard.unitMonth")}
                  </option>
                  <option value="day">{t("dashboard.addCard.unitDay")}</option>
                </select>
              </label>
            )}

            {type === "entry_list" && (
              <label className="flex flex-col gap-1.5 text-sm font-medium">
                {t("dashboard.addCard.columnsLabel")}
                <select
                  value={columns}
                  onChange={(e) => setColumns(Number(e.target.value))}
                  className={inputClass}
                >
                  <option value={2}>{t("dashboard.addCard.columns2")}</option>
                  <option value={3}>{t("dashboard.addCard.columns3")}</option>
                  <option value={4}>{t("dashboard.addCard.columns4")}</option>
                </select>
              </label>
            )}

            {(type === "entry_list" || type === "bar_chart") && (
              <label className="flex items-center gap-2 text-sm">
                <input
                  type="checkbox"
                  checked={showRecurringPreview}
                  onChange={(e) => setShowRecurringPreview(e.target.checked)}
                />
                {t("dashboard.addCard.showRecurringPreviewLabel")}
              </label>
            )}

            {error && (
              <p className="text-sm text-red-600 dark:text-red-400">{error}</p>
            )}

            <div className="flex justify-end gap-2">
              <button
                type="button"
                onClick={onClose}
                className="rounded-md px-3 py-2 text-sm font-medium text-zinc-600 hover:bg-black/[.04] dark:text-zinc-400 dark:hover:bg-white/[.06]"
              >
                {t("categories.edit.cancel")}
              </button>
              <button
                type="submit"
                disabled={saving}
                className="rounded-md bg-zinc-900 px-3 py-2 text-sm font-medium text-white hover:bg-zinc-700 disabled:opacity-60 dark:bg-white dark:text-zinc-900 dark:hover:bg-zinc-200"
              >
                {editingCard
                  ? saving
                    ? t("categories.edit.saving")
                    : t("categories.edit.save")
                  : t("dashboard.addCard.submit")}
              </button>
            </div>
          </form>
        </DialogPanel>
      </div>
    </Dialog>
  );
}
