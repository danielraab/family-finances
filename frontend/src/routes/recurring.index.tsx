import { createFileRoute, Link } from "@tanstack/react-router";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { api } from "../api/client";
import type { components } from "../api/schema";
import { CategoryLabel } from "../components/CategoryLabel";
import { amountColorClass, formatAmount } from "../lib/amount";
import { CUSTOM_PRESET_KEY, matchPreset } from "../lib/recurrence";
import { useDisplayedDecimalPlaces } from "../lib/useDisplayedDecimalPlaces";

type Category = components["schemas"]["Category"];
type RecurringTransaction = components["schemas"]["RecurringTransaction"];
type CurrencySum = components["schemas"]["CurrencySum"];

export const Route = createFileRoute("/recurring/")({
  component: RecurringTransactionsList,
});

function RecurringTransactionsList() {
  const { t, i18n } = useTranslation();
  const displayedDecimalPlaces = useDisplayedDecimalPlaces();

  const [items, setItems] = useState<RecurringTransaction[]>([]);
  const [categories, setCategories] = useState<Category[]>([]);
  const [sums, setSums] = useState<CurrencySum[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    Promise.all([
      api.GET("/api/recurring-transactions"),
      api.GET("/api/recurring-transactions/summary"),
      api.GET("/api/categories"),
    ]).then(([list, summary, cats]) => {
      setItems(list.data ?? []);
      setSums(summary.data?.sums ?? []);
      setCategories(cats.data ?? []);
      setLoading(false);
    });
  }, []);

  const categoryById = new Map(categories.map((c) => [c.id, c]));

  return (
    <section className="mx-auto flex w-full max-w-5xl flex-col gap-6 px-6 py-12 sm:px-10">
      <header className="flex items-center justify-between gap-4">
        <h1 className="text-2xl font-semibold tracking-tight">
          {t("recurring.title")}
        </h1>
        <Link
          to="/recurring/new"
          className="rounded-md bg-zinc-900 px-3 py-2 text-sm font-medium text-white transition-colors hover:bg-zinc-700 dark:bg-white dark:text-zinc-900 dark:hover:bg-zinc-200"
        >
          {t("recurring.create")}
        </Link>
      </header>

      <div className="overflow-x-auto rounded-lg border border-black/10 dark:border-white/10">
        <table className="w-full text-left text-sm">
          <thead className="border-b border-black/10 text-xs uppercase text-zinc-500 dark:border-white/10 dark:text-zinc-400">
            <tr>
              <th className="px-3 py-2 font-medium">
                {t("recurring.columns.title")}
              </th>
              <th className="px-3 py-2 font-medium">
                {t("recurring.columns.category")}
              </th>
              <th className="px-3 py-2 font-medium">
                {t("recurring.columns.frequency")}
              </th>
              <th className="px-3 py-2 text-right font-medium">
                {t("recurring.columns.amount")}
              </th>
              <th className="px-3 py-2 text-right font-medium">
                {t("recurring.columns.perYearAmount")}
              </th>
              <th className="px-3 py-2 font-medium" />
            </tr>
          </thead>
          <tbody>
            {items.map((rt) => {
              const category = rt.category_id
                ? categoryById.get(rt.category_id)
                : undefined;
              return (
                <tr
                  key={rt.id}
                  className={`border-b border-black/5 last:border-0 dark:border-white/5 ${
                    rt.ended ? "opacity-50" : ""
                  }`}
                >
                  <td className="px-3 py-2">
                    <Link
                      to="/recurring/$id/edit"
                      params={{ id: rt.id }}
                      className="font-medium underline-offset-2 hover:underline"
                    >
                      {rt.title}
                    </Link>
                    {rt.ended && (
                      <span className="ml-2 rounded-full bg-black/[.06] px-2 py-0.5 text-xs font-medium text-zinc-500 dark:bg-white/10 dark:text-zinc-400">
                        {t("recurring.ended")}
                      </span>
                    )}
                  </td>
                  <td className="px-3 py-2 text-zinc-500 dark:text-zinc-400">
                    {category ? (
                      <CategoryLabel category={category} iconSize={16} />
                    ) : (
                      <span aria-hidden>—</span>
                    )}
                  </td>
                  <td className="px-3 py-2 text-zinc-500 dark:text-zinc-400">
                    {(() => {
                      const presetKey = matchPreset(
                        rt.interval_unit,
                        rt.interval_count,
                      );
                      return presetKey === CUSTOM_PRESET_KEY
                        ? t("recurring.customFrequency", {
                            count: rt.interval_count,
                            unit: t(`recurring.units.${rt.interval_unit}`),
                          })
                        : t(`recurring.presets.${presetKey}`);
                    })()}
                  </td>
                  <td
                    className={`px-3 py-2 text-right font-mono tabular-nums ${amountColorClass(rt.amount)}`}
                  >
                    {formatAmount(
                      rt.amount,
                      rt.account_currency ?? "",
                      displayedDecimalPlaces,
                      i18n.resolvedLanguage ?? "en",
                    )}
                  </td>
                  <td
                    className={`px-3 py-2 text-right font-mono tabular-nums ${amountColorClass(rt.per_year_amount)}`}
                  >
                    {formatAmount(
                      rt.per_year_amount,
                      rt.account_currency ?? "",
                      displayedDecimalPlaces,
                      i18n.resolvedLanguage ?? "en",
                    )}
                  </td>
                  <td className="px-3 py-2 text-right">
                    <Link
                      to="/entries/new"
                      search={{ recurring_transaction_id: rt.id }}
                      className="rounded-md border border-black/15 px-2.5 py-1.5 text-xs font-medium transition-colors hover:bg-black/[.04] dark:border-white/15 dark:hover:bg-white/[.06]"
                    >
                      {t("recurring.createTransaction")}
                    </Link>
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>

      {!loading && items.length === 0 && (
        <p className="text-sm text-zinc-500 dark:text-zinc-400">
          {t("recurring.empty")}
        </p>
      )}

      {sums.length > 0 && (
        <div className="flex flex-col gap-1 rounded-lg border border-black/10 p-4 text-sm dark:border-white/10">
          <span className="font-medium text-zinc-500 dark:text-zinc-400">
            {t("recurring.totalPerYear")}
          </span>
          <div className="flex flex-wrap gap-x-6 gap-y-1">
            {sums.map((sum) => (
              <span
                key={sum.currency}
                className={`font-mono text-base tabular-nums ${amountColorClass(sum.amount)}`}
              >
                {formatAmount(
                  sum.amount,
                  sum.currency,
                  displayedDecimalPlaces,
                  i18n.resolvedLanguage ?? "en",
                )}
              </span>
            ))}
          </div>
        </div>
      )}
    </section>
  );
}
