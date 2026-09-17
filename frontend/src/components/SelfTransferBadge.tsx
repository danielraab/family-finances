import { Link } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";
import { SelfTransferIcon } from "./SelfTransferIcon";

/**
 * A small badge shown on any entry that is a self-transfer
 * (`kind === "self_transfer"`), in the ledger (`/entries`) and the
 * `/reports` results table — links to that same entry's own edit page,
 * viewed from its other account (see account-entries: a self-transfer is
 * listed once per account it touches). Renders null for any other kind, so
 * call sites can render it unconditionally.
 */
export function SelfTransferBadge({
  entryId,
  kind,
}: {
  entryId: string;
  kind: string;
}) {
  const { t } = useTranslation();
  if (kind !== "self_transfer") {
    return null;
  }
  return (
    <Link
      to="/entries/$entryId/edit"
      params={{ entryId }}
      onClick={(e) => e.stopPropagation()}
      aria-label={t("entries.selfTransferBadge")}
      title={t("entries.selfTransferBadge")}
      className="ml-1.5 inline-flex align-middle text-zinc-500 hover:text-zinc-700 dark:text-zinc-400 dark:hover:text-zinc-200"
    >
      <SelfTransferIcon width={14} height={14} />
    </Link>
  );
}
