import { Link } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";
import { RecurringIcon } from "./RecurringIcon";

/**
 * A small badge shown on any entry linked to a recurring transaction
 * (`recurring_transaction_id` set), in the ledger (`/entries`) and the
 * `/reports` results table — links to that recurring transaction's edit
 * page. Renders null when unlinked, so call sites can render it
 * unconditionally.
 */
export function RecurringTransactionBadge({
  recurringTransactionId,
}: {
  recurringTransactionId: string | null | undefined;
}) {
  const { t } = useTranslation();
  if (!recurringTransactionId) {
    return null;
  }
  return (
    <Link
      to="/recurring/$id/edit"
      params={{ id: recurringTransactionId }}
      onClick={(e) => e.stopPropagation()}
      aria-label={t("entries.recurringBadge")}
      title={t("entries.recurringBadge")}
      className="ml-1.5 inline-flex align-middle text-zinc-500 hover:text-zinc-700 dark:text-zinc-400 dark:hover:text-zinc-200"
    >
      <RecurringIcon width={14} height={14} />
    </Link>
  );
}
