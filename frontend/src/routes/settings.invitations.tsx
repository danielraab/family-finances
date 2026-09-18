import { Description } from "@headlessui/react";
import { createFileRoute } from "@tanstack/react-router";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { api } from "../api/client";
import type { components } from "../api/schema";
import { useAuth } from "../components/AuthProvider";
import { InviteList } from "../components/InviteList";
import { Modal } from "../components/Modal";

export const Route = createFileRoute("/settings/invitations")({
  component: MyInvitationsTab,
});

type Invite = components["schemas"]["Invite"];

/**
 * The My Invitations tab: every authenticated visitor's own view of the
 * invitations they've personally sent (GET /api/auth/invites/mine) — the
 * non-admin counterpart of the admin Users tab's all-invitations list. Only
 * the inviter or an admin may revoke an invitation; this tab is where a
 * non-admin inviter gets to do that themselves.
 */
function MyInvitationsTab() {
  const { user } = useAuth();
  const { t } = useTranslation();
  const [invites, setInvites] = useState<Invite[] | null>(null);
  const [revoking, setRevoking] = useState<Invite | null>(null);

  useEffect(() => {
    if (!user) return;
    let cancelled = false;
    api.GET("/api/auth/invites/mine").then(({ data }) => {
      if (cancelled || !data) return;
      setInvites(data);
    });
    return () => {
      cancelled = true;
    };
  }, [user]);

  if (!user) {
    return null;
  }

  async function confirmRevoke() {
    if (!revoking) return;
    const target = revoking;
    setRevoking(null);
    const { data } = await api.POST("/api/auth/invites/{id}/revoke", {
      params: { path: { id: target.id } },
    });
    if (data) {
      setInvites(
        (prev) => prev?.map((i) => (i.id === data.id ? data : i)) ?? null,
      );
    }
  }

  return (
    <div className="flex flex-col gap-3">
      <h2 className="text-sm font-semibold text-zinc-500 dark:text-zinc-400">
        {t("settings.myInvitations.heading")}
      </h2>
      {invites === null ? null : invites.length === 0 ? (
        <p className="text-sm text-zinc-500 dark:text-zinc-400">
          {t("settings.myInvitations.noInvites")}
        </p>
      ) : (
        <InviteList
          invites={invites}
          showInviter={false}
          onRevoke={setRevoking}
        />
      )}

      <Modal
        open={revoking !== null}
        onClose={() => setRevoking(null)}
        title={
          revoking
            ? t("settings.invite.confirmRevokeTitle", { email: revoking.email })
            : ""
        }
      >
        {revoking && (
          <>
            <Description className="text-sm text-zinc-600 dark:text-zinc-400">
              {t("settings.invite.confirmRevokeBody")}
            </Description>
            <div className="flex justify-end gap-2">
              <button
                type="button"
                onClick={() => setRevoking(null)}
                className="rounded-md px-3 py-2 text-sm font-medium text-zinc-600 hover:bg-black/[.04] dark:text-zinc-400 dark:hover:bg-white/[.06]"
              >
                {t("settings.users.confirm.cancel")}
              </button>
              <button
                type="button"
                onClick={confirmRevoke}
                className="rounded-md bg-zinc-900 px-3 py-2 text-sm font-medium text-white hover:bg-zinc-700 dark:bg-white dark:text-zinc-900 dark:hover:bg-zinc-200"
              >
                {t("settings.users.confirm.confirmAction")}
              </button>
            </div>
          </>
        )}
      </Modal>
    </div>
  );
}
