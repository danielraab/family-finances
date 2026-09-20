import { Link } from "@tanstack/react-router";
import { Plus } from "lucide-react";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { api } from "../../api/client";
import type { components } from "../../api/schema";
import { amountColorClass, formatAmount } from "../../lib/amount";
import { CUSTOM_PRESET_KEY, matchPreset } from "../../lib/recurrence";
import { AccountLabel } from "../AccountLabel";
import { CategoryLabel } from "../CategoryLabel";
import { Modal } from "../Modal";
import { TagLabel } from "../TagLabel";
import type { SummaryLookups } from "./lookups";
import {
  SummaryActions,
  SummaryField,
  SummaryFields,
  summaryLinkClass,
  summaryOutlineLinkClass,
  summaryQuietButtonClass,
} from "./SummaryField";

type RecurringTransaction = components["schemas"]["RecurringTransaction"];

/**
 * A read-only view of one recurring transaction.
 *
 * Unlike `EntrySummaryModal`, this one fetches: it is opened from an
 * entry's badge, which carries only a `recurring_transaction_id`, and no
 * page in the app holds the recurring transactions themselves.
 */
export function RecurringSummaryModal({
  recurringTransactionId,
  bookingTimestamp,
  lookups,
  onBack,
  onClose,
}: {
  recurringTransactionId: string;
  /** The occurrence this summary was opened for, when it was opened from
   * one — an Upcoming row names a specific projected date, and Create
   * transaction here should book that date rather than the template's
   * next suggested one. Absent everywhere else (a badge on a linked entry,
   * a `/recurring` row), where the form's own default applies. */
  bookingTimestamp?: string | undefined;
  lookups: SummaryLookups;
  /** Set only when this was reached from an entry's summary. */
  onBack?: (() => void) | undefined;
  onClose: () => void;
}) {
  const { t, i18n } = useTranslation();
  const locale = i18n.resolvedLanguage ?? "en";
  const [item, setItem] = useState<RecurringTransaction | null>(null);
  const [failed, setFailed] = useState(false);

  useEffect(() => {
    let cancelled = false;
    setItem(null);
    setFailed(false);
    api
      .GET("/api/recurring-transactions/{id}", {
        params: { path: { id: recurringTransactionId } },
      })
      .then(({ data }) => {
        if (cancelled) return;
        if (data) setItem(data);
        else setFailed(true);
      });
    return () => {
      cancelled = true;
    };
  }, [recurringTransactionId]);

  const account = item ? lookups.accountById.get(item.account_id) : undefined;
  const toAccount = item?.to_account_id
    ? lookups.accountById.get(item.to_account_id)
    : undefined;
  const category =
    item?.category_id !== null && item?.category_id !== undefined
      ? lookups.categoryById.get(item.category_id)
      : undefined;
  const tags = (item?.tag_ids ?? [])
    .map((id) => lookups.tagById.get(id))
    .filter((tag) => tag !== undefined);

  function frequency(rt: RecurringTransaction): string {
    const presetKey = matchPreset(rt.interval_unit, rt.interval_count);
    return presetKey === CUSTOM_PRESET_KEY
      ? t("recurring.customFrequency", {
          count: rt.interval_count,
          unit: t(`recurring.units.${rt.interval_unit}`),
        })
      : t(`recurring.presets.${presetKey}`);
  }

  return (
    <Modal
      open
      onClose={onClose}
      size="md"
      title={item?.title ?? t("summary.recurring.title")}
    >
      {failed ? (
        <p className="text-sm text-red-600 dark:text-red-400">
          {t("summary.recurring.loadError")}
        </p>
      ) : item === null ? (
        <p className="text-sm text-zinc-500 dark:text-zinc-400">
          {t("summary.loading")}
        </p>
      ) : (
        <SummaryFields>
          <SummaryField label={t("summary.recurring.amount")}>
            <span
              className={`font-mono tabular-nums ${amountColorClass(item.amount)}`}
            >
              {formatAmount(
                item.amount,
                item.account_currency ?? "",
                lookups.displayedDecimalPlaces,
                locale,
              )}
            </span>
          </SummaryField>
          <SummaryField label={t("summary.recurring.frequency")}>
            {frequency(item)}
          </SummaryField>
          <SummaryField label={t("summary.recurring.perYearAmount")}>
            <span
              className={`font-mono tabular-nums ${amountColorClass(item.per_year_amount)}`}
            >
              {formatAmount(
                item.per_year_amount,
                item.account_currency ?? "",
                lookups.displayedDecimalPlaces,
                locale,
              )}
            </span>
          </SummaryField>
          <SummaryField label={t("summary.recurring.account")}>
            {account ? (
              <AccountLabel account={account} iconSize={16} />
            ) : (
              <span className="italic text-zinc-500 dark:text-zinc-400">
                {t("entries.notShared")}
              </span>
            )}
          </SummaryField>
          {item.kind === "self_transfer" && (
            <SummaryField label={t("summary.entry.toAccount")}>
              {toAccount ? (
                <AccountLabel account={toAccount} iconSize={16} />
              ) : (
                (item.to_account_name ?? (
                  <span className="italic text-zinc-500 dark:text-zinc-400">
                    {t("entries.notShared")}
                  </span>
                ))
              )}
            </SummaryField>
          )}
          <SummaryField label={t("summary.recurring.startsOn")}>
            {new Date(item.starts_on).toLocaleDateString(locale)}
          </SummaryField>
          {item.ends_on && (
            <SummaryField label={t("summary.recurring.endsOn")}>
              {new Date(item.ends_on).toLocaleDateString(locale)}
            </SummaryField>
          )}
          {item.ended && (
            <SummaryField label={t("summary.recurring.status")}>
              {t("recurring.ended")}
            </SummaryField>
          )}
          {!item.ended && item.next_suggested_date && (
            <SummaryField label={t("summary.recurring.nextSuggested")}>
              {new Date(item.next_suggested_date).toLocaleDateString(locale)}
            </SummaryField>
          )}
          <SummaryField label={t("summary.recurring.description")}>
            {item.description}
          </SummaryField>
          {item.category_id && (
            <SummaryField label={t("summary.recurring.category")}>
              {category ? (
                <CategoryLabel category={category} iconSize={16} />
              ) : (
                <span className="italic text-zinc-500 dark:text-zinc-400">
                  {t("entries.notShared")}
                </span>
              )}
            </SummaryField>
          )}
          {tags.length > 0 && (
            <SummaryField label={t("summary.recurring.tags")}>
              <span className="flex flex-wrap gap-1">
                {tags.map((tag) => (
                  <TagLabel
                    key={tag.id}
                    tag={tag}
                    className="rounded-full bg-black/[.06] px-2 py-0.5 text-xs font-medium dark:bg-white/10"
                  />
                ))}
              </span>
            </SummaryField>
          )}
          <SummaryField label={t("summary.recurring.counterparty")}>
            {item.counterparty}
          </SummaryField>
          <SummaryField label={t("summary.recurring.linkedEntries")}>
            {t("summary.recurring.linkedEntriesCount", {
              count: item.linked_entry_count,
            })}
          </SummaryField>
        </SummaryFields>
      )}

      <SummaryActions>
        {onBack && (
          <button
            type="button"
            onClick={onBack}
            className={`mr-auto ${summaryQuietButtonClass}`}
          >
            {t("summary.back")}
          </button>
        )}
        <button
          type="button"
          onClick={onClose}
          className={summaryQuietButtonClass}
        >
          {t("summary.close")}
        </button>
        <Link
          to="/entries/new"
          search={{
            recurring_transaction_id: recurringTransactionId,
            ...(bookingTimestamp
              ? { booking_timestamp: bookingTimestamp }
              : {}),
          }}
          className={summaryOutlineLinkClass}
        >
          <Plus size={16} aria-hidden="true" />
          {t("recurring.createTransaction")}
        </Link>
        <Link
          to="/recurring/$id/edit"
          params={{ id: recurringTransactionId }}
          className={summaryLinkClass}
        >
          {t("summary.edit")}
        </Link>
      </SummaryActions>
    </Modal>
  );
}
