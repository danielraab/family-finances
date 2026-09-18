import { Link } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";
import type { components } from "../../api/schema";
import { amountColorClass, formatAmount } from "../../lib/amount";
import { canEditEntry } from "../../lib/entryPermissions";
import { parseLocation } from "../../lib/location";
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
  summaryQuietButtonClass,
} from "./SummaryField";

type Entry = components["schemas"]["Entry"];

/**
 * A read-only view of one entry, rendered from the data the listing that
 * opened it already holds — no request of its own, so it shows exactly
 * what the row behind it shows (see the change's design.md).
 *
 * The Edit action is gated by `canEditEntry`, the same predicate the edit
 * page itself uses, so the two can never disagree about who may edit.
 */
export function EntrySummaryModal({
  entry,
  lookups,
  onOpenRecurring,
  onClose,
}: {
  entry: Entry;
  lookups: SummaryLookups;
  /** Omitted when the host has no way to show a recurring summary. */
  onOpenRecurring?: ((recurringTransactionId: string) => void) | undefined;
  onClose: () => void;
}) {
  const { t, i18n } = useTranslation();
  const locale = i18n.resolvedLanguage ?? "en";
  const account = lookups.accountById.get(entry.account_id);
  const toAccount = entry.to_account_id
    ? lookups.accountById.get(entry.to_account_id)
    : undefined;
  const category = entry.category_id
    ? lookups.categoryById.get(entry.category_id)
    : undefined;
  const tags = entry.tag_ids
    .map((id) => lookups.tagById.get(id))
    .filter((tag) => tag !== undefined);
  const coordinates = parseLocation(entry.location);
  const canEdit = canEditEntry(entry, account, toAccount, lookups.userId);

  return (
    <Modal open onClose={onClose} size="md" title={entry.title}>
      <SummaryFields>
        <SummaryField label={t("summary.entry.bookedAt")}>
          {new Date(entry.booking_timestamp).toLocaleString(locale, {
            dateStyle: "medium",
            timeStyle: "short",
          })}
        </SummaryField>
        <SummaryField label={t("summary.entry.amount")}>
          <span
            className={`font-mono tabular-nums ${amountColorClass(entry.amount)}`}
          >
            {formatAmount(
              entry.amount,
              entry.account_currency ?? "",
              lookups.displayedDecimalPlaces,
              locale,
            )}
          </span>
        </SummaryField>
        {entry.balance !== null && entry.balance !== undefined && (
          <SummaryField label={t("summary.entry.balance")}>
            <span className="font-mono tabular-nums">
              {formatAmount(
                entry.balance,
                entry.account_currency ?? "",
                lookups.displayedDecimalPlaces,
                locale,
              )}
            </span>
          </SummaryField>
        )}
        <SummaryField label={t("summary.entry.account")}>
          {account ? (
            <AccountLabel account={account} iconSize={16} />
          ) : (
            <span className="italic text-zinc-500 dark:text-zinc-400">
              {t("entries.notShared")}
            </span>
          )}
        </SummaryField>
        {entry.kind === "self_transfer" && (
          <SummaryField label={t("summary.entry.toAccount")}>
            {toAccount ? (
              <AccountLabel account={toAccount} iconSize={16} />
            ) : (
              (entry.to_account_name ?? (
                <span className="italic text-zinc-500 dark:text-zinc-400">
                  {t("entries.notShared")}
                </span>
              ))
            )}
          </SummaryField>
        )}
        <SummaryField label={t("summary.entry.description")}>
          {entry.description}
        </SummaryField>
        {entry.category_id && (
          <SummaryField label={t("summary.entry.category")}>
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
          <SummaryField label={t("summary.entry.tags")}>
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
        <SummaryField label={t("summary.entry.counterparty")}>
          {entry.counterparty}
        </SummaryField>
        <SummaryField label={t("summary.entry.location")}>
          {coordinates
            ? `${coordinates.lat.toFixed(5)}, ${coordinates.lng.toFixed(5)}`
            : entry.location}
        </SummaryField>
        {entry.created_by !== lookups.userId && entry.created_by_name && (
          <SummaryField label={t("summary.entry.createdBy")}>
            {entry.created_by_name}
          </SummaryField>
        )}
      </SummaryFields>

      <SummaryActions>
        {entry.recurring_transaction_id && onOpenRecurring && (
          <button
            type="button"
            onClick={() => {
              if (entry.recurring_transaction_id) {
                onOpenRecurring(entry.recurring_transaction_id);
              }
            }}
            className={`mr-auto ${summaryQuietButtonClass}`}
          >
            {t("summary.entry.viewRecurring")}
          </button>
        )}
        <button
          type="button"
          onClick={onClose}
          className={summaryQuietButtonClass}
        >
          {t("summary.close")}
        </button>
        {canEdit && (
          <Link
            to="/entries/$entryId/edit"
            params={{ entryId: entry.id }}
            className={summaryLinkClass}
          >
            {t("summary.edit")}
          </Link>
        )}
      </SummaryActions>
    </Modal>
  );
}
