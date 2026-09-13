import { Link } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";

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
      <svg
        xmlns="http://www.w3.org/2000/svg"
        viewBox="0 0 24 24"
        width={14}
        height={14}
        fill="none"
        stroke="currentColor"
        strokeWidth={2}
        strokeLinecap="round"
        strokeLinejoin="round"
        aria-hidden="true"
      >
        <path d="M17 2.1 21 6l-4 3.9" />
        <path d="M3 11V9a4 4 0 0 1 4-4h14" />
        <path d="M7 21.9 3 18l4-3.9" />
        <path d="M21 13v2a4 4 0 0 1-4 4H3" />
      </svg>
    </Link>
  );
}
