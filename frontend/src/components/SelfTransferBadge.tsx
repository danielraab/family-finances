import { useTranslation } from "react-i18next";
import { SelfTransferIcon } from "./SelfTransferIcon";

/**
 * A small badge shown on any entry that is a self-transfer
 * (`kind === "self_transfer"`), in the ledger (`/entries`) and the
 * `/reports` results table — opens that same entry's read-only summary
 * modal, the same gesture as clicking the entry's title (a self-transfer
 * is listed once per account it touches, sharing one `id` — see
 * account-entries — so the badge and the title always resolve to the
 * exact same entry, never a counterpart). Renders null for any other
 * kind, so call sites can render it unconditionally.
 *
 * `stopPropagation` because the badge sits inside a row that responds to
 * clicks of its own; opening a modal rather than navigating doesn't
 * change that.
 */
export function SelfTransferBadge({
  kind,
  onOpen,
}: {
  kind: string;
  onOpen: () => void;
}) {
  const { t } = useTranslation();
  if (kind !== "self_transfer") {
    return null;
  }
  return (
    <button
      type="button"
      onClick={(e) => {
        e.stopPropagation();
        onOpen();
      }}
      aria-label={t("entries.selfTransferBadge")}
      title={t("entries.selfTransferBadge")}
      className="ml-1.5 inline-flex align-middle text-zinc-500 hover:text-zinc-700 dark:text-zinc-400 dark:hover:text-zinc-200"
    >
      <SelfTransferIcon width={14} height={14} />
    </button>
  );
}
