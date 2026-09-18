import { useTranslation } from "react-i18next";
import { RecurringIcon } from "./RecurringIcon";

/**
 * A small badge shown on any entry linked to a recurring transaction
 * (`recurring_transaction_id` set), in the ledger (`/entries`) and the
 * `/reports` results table — opens that recurring transaction's read-only
 * summary modal. Renders null when unlinked, so call sites can render it
 * unconditionally.
 *
 * `stopPropagation` because the badge sits inside a row that responds to
 * clicks of its own; opening a modal rather than navigating doesn't change
 * that.
 */
export function RecurringTransactionBadge({
  recurringTransactionId,
  onOpen,
}: {
  recurringTransactionId: string | null | undefined;
  onOpen: (recurringTransactionId: string) => void;
}) {
  const { t } = useTranslation();
  if (!recurringTransactionId) {
    return null;
  }
  return (
    <button
      type="button"
      onClick={(e) => {
        e.stopPropagation();
        onOpen(recurringTransactionId);
      }}
      aria-label={t("entries.recurringBadge")}
      title={t("entries.recurringBadge")}
      className="ml-1.5 inline-flex align-middle text-zinc-500 hover:text-zinc-700 dark:text-zinc-400 dark:hover:text-zinc-200"
    >
      <RecurringIcon width={14} height={14} />
    </button>
  );
}
