import { useTranslation } from "react-i18next";
import type { components } from "../api/schema";
import { EntityIcon } from "./EntityIcon";

type Account = components["schemas"]["Account"];

type AccountLabelProps = {
  account: Pick<Account, "title" | "icon" | "color" | "shared" | "owner_name">;
  /** Badge size in px. Default 20. */
  iconSize?: number | undefined;
  /** Classes for the wrapper (e.g. `font-medium`). */
  className?: string | undefined;
};

function SharedGlyph() {
  return (
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
      <path d="M16 3.5a3 3 0 1 0 0 6 3 3 0 0 0 0-6ZM8 9.5a3 3 0 1 0 0 6 3 3 0 0 0 0-6ZM16 14.5a3 3 0 1 0 0 6 3 3 0 0 0 0-6Z" />
      <path d="m10.7 11 4.6-2.6M10.7 13l4.6 2.6" />
    </svg>
  );
}

/**
 * An account's name, preceded by its icon/colour badge when it has one, and
 * followed by a shared indicator plus the real owner's name whenever the
 * account is shared (i.e. the viewer isn't its real owner) — never shown to
 * the real owner themselves, even when they've shared it with others. Used
 * everywhere an account is shown by name outside a native `<select>`.
 *
 * The shared badge sits inline next to the name when there's room, but wraps
 * onto its own full-width line in tight containers (narrow cards, table
 * cells) so the name and the owner name stay readable instead of both
 * truncating to compete for one line.
 */
export function AccountLabel({
  account,
  iconSize,
  className,
}: AccountLabelProps) {
  const { t } = useTranslation();
  return (
    <span
      className={`inline-flex min-w-0 flex-wrap items-center gap-x-1.5 gap-y-1 ${
        className ?? ""
      }`}
    >
      <span className="inline-flex min-w-32 grow items-center gap-1.5">
        <EntityIcon icon={account.icon} color={account.color} size={iconSize} />
        <span className="min-w-0 truncate">{account.title}</span>
      </span>
      {account.shared && (
        <span
          className="inline-flex max-w-full min-w-0 shrink-0 items-center gap-1 rounded-full bg-black/[.06] px-1.5 py-0.5 text-xs font-normal text-zinc-500 dark:bg-white/[.08] dark:text-zinc-400"
          title={t("accounts.shared.badgeTitle", {
            owner: account.owner_name ?? "",
          })}
        >
          <span className="shrink-0">
            <SharedGlyph />
          </span>
          {account.owner_name && (
            <span className="truncate">{account.owner_name}</span>
          )}
        </span>
      )}
    </span>
  );
}
