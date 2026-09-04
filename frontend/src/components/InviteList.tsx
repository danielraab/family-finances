import { useTranslation } from "react-i18next";
import type { components } from "../api/schema";

type Invite = components["schemas"]["Invite"];

function formatDate(iso: string, lang: string): string {
  return new Intl.DateTimeFormat(lang, {
    dateStyle: "medium",
  }).format(new Date(iso));
}

/**
 * The invitation-row list shared by the admin Users tab and the My
 * Invitations tab: status (revoked takes precedence over accepted takes
 * precedence over pending/expires), an optional inviter line, and a Revoke
 * action for any not-yet-revoked row. Callers own fetching, the confirmation
 * step, and the empty-state text (rendered by the caller when `invites` is
 * empty, so each tab can word it for its own context).
 */
export function InviteList({
  invites,
  showInviter,
  onRevoke,
  onDelete,
}: {
  invites: Invite[];
  showInviter: boolean;
  onRevoke: (invite: Invite) => void;
  /**
   * Admin-only soft-delete. Omit this prop entirely on a non-admin surface
   * (e.g. the My Invitations tab) — the Delete button only ever renders when
   * it's provided, and only once an invite is already revoked, matching
   * DELETE /api/auth/invites/{id}'s requirement that revoked_at be set.
   */
  onDelete?: (invite: Invite) => void;
}) {
  const { t, i18n } = useTranslation();
  const lang = i18n.resolvedLanguage ?? "en";

  return (
    <ul className="flex flex-col gap-2">
      {invites.map((invite) => (
        <li
          key={invite.id}
          className="flex items-center justify-between gap-3 rounded-md border border-black/10 px-3 py-2 text-sm dark:border-white/10"
        >
          <span className="flex flex-col gap-0.5 leading-tight">
            <span className="font-medium">{invite.email}</span>
            <span className="text-xs text-zinc-500 dark:text-zinc-400">
              {showInviter && (
                <>
                  {t("settings.invite.invitedBy", {
                    name:
                      invite.invited_by.display_name || invite.invited_by.email,
                  })}
                  {" · "}
                </>
              )}
              {invite.revoked_at
                ? t("settings.invite.revoked", {
                    date: formatDate(invite.revoked_at, lang),
                  })
                : invite.accepted_at
                  ? t("settings.invite.accepted", {
                      date: formatDate(invite.accepted_at, lang),
                    })
                  : `${t("settings.invite.pending")} · ${t(
                      "settings.invite.expires",
                      { date: formatDate(invite.expires_at, lang) },
                    )}`}
            </span>
          </span>
          <div className="flex shrink-0 gap-2">
            {!invite.revoked_at && (
              <button
                type="button"
                onClick={() => onRevoke(invite)}
                className="text-xs font-medium text-red-600 underline underline-offset-2 hover:text-red-800 dark:text-red-400 dark:hover:text-red-300"
              >
                {t("settings.invite.revoke")}
              </button>
            )}
            {onDelete && invite.revoked_at && (
              <button
                type="button"
                onClick={() => onDelete(invite)}
                className="text-xs font-medium text-red-600 underline underline-offset-2 hover:text-red-800 dark:text-red-400 dark:hover:text-red-300"
              >
                {t("settings.invite.delete")}
              </button>
            )}
          </div>
        </li>
      ))}
    </ul>
  );
}
