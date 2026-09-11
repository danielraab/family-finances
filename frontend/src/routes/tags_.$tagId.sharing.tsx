import {
  Description,
  Dialog,
  DialogPanel,
  DialogTitle,
} from "@headlessui/react";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { type FormEvent, useCallback, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { api } from "../api/client";
import type { components } from "../api/schema";
import { useAuth } from "../components/AuthProvider";

export const Route = createFileRoute("/tags_/$tagId/sharing")({
  component: SharingPage,
});

type Tag = components["schemas"]["Tag"];
type TagShare = components["schemas"]["TagShare"];
type TagPermission = components["schemas"]["TagPermission"];

const PERMISSIONS: Exclude<TagPermission, "owner">[] = ["view", "append"];

const inputClass =
  "rounded-md border border-black/15 bg-transparent px-3 py-2 text-sm font-normal outline-none transition-colors focus:border-black/40 dark:border-white/15 dark:focus:border-white/40";

function formatDate(iso: string, lang: string): string {
  return new Intl.DateTimeFormat(lang, { dateStyle: "medium" }).format(
    new Date(iso),
  );
}

/**
 * The /tags/{id}/sharing page — structurally identical to
 * /categories/{id}/sharing, with the same two-tier permission model. See
 * web-client-tag-sharing's spec.
 */
function SharingPage() {
  const { tagId } = Route.useParams();
  const { t, i18n } = useTranslation();
  const { user } = useAuth();
  const navigate = useNavigate();

  const [tag, setTag] = useState<Tag | null | undefined>(undefined);
  const [shares, setShares] = useState<TagShare[] | null>(null);

  const [inviteEmail, setInviteEmail] = useState("");
  const [invitePermission, setInvitePermission] =
    useState<TagPermission>("view");
  const [inviting, setInviting] = useState(false);
  const [inviteError, setInviteError] = useState<string | null>(null);
  const [unmatched, setUnmatched] = useState<{
    email: string;
    inviteAllowed: boolean;
  } | null>(null);
  const [sendingAppInvite, setSendingAppInvite] = useState(false);
  const [appInviteSent, setAppInviteSent] = useState(false);

  const [confirming, setConfirming] = useState<{
    kind: "revoke" | "leave";
    share: TagShare;
  } | null>(null);

  const load = useCallback(() => {
    let cancelled = false;
    api
      .GET("/api/tags/{id}", { params: { path: { id: tagId } } })
      .then(({ data }) => {
        if (!cancelled) setTag(data ?? null);
      });
    api
      .GET("/api/tags/{id}/shares", {
        params: { path: { id: tagId } },
      })
      .then(({ data }) => {
        if (!cancelled) setShares(data ?? []);
      });
    return () => {
      cancelled = true;
    };
  }, [tagId]);

  useEffect(() => load(), [load]);

  async function handleInvite(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setInviting(true);
    setInviteError(null);
    setUnmatched(null);
    setAppInviteSent(false);
    const email = inviteEmail.trim();
    const { data, response } = await api.POST("/api/tags/{id}/shares", {
      params: { path: { id: tagId } },
      body: { email, permission: invitePermission },
    });
    setInviting(false);
    if (!response.ok || !data) {
      setInviteError(t("tags.sharing.inviteError"));
      return;
    }
    if (data.matched && data.share) {
      setShares((prev) => [...(prev ?? []), data.share as TagShare]);
      setInviteEmail("");
    } else {
      setUnmatched({ email, inviteAllowed: data.invite_allowed });
    }
  }

  async function sendAppInvite() {
    if (!unmatched) return;
    setSendingAppInvite(true);
    const { response } = await api.POST("/api/auth/invites", {
      body: { email: unmatched.email },
    });
    setSendingAppInvite(false);
    if (response.ok) setAppInviteSent(true);
  }

  async function updatePermission(userId: string, permission: TagPermission) {
    const { data } = await api.PATCH("/api/tags/{id}/shares/{userId}", {
      params: { path: { id: tagId, userId } },
      body: { permission },
    });
    if (data) {
      setShares(
        (prev) =>
          prev?.map((share) => (share.user_id === userId ? data : share)) ??
          null,
      );
    }
  }

  async function performConfirmed() {
    if (!confirming) return;
    const { share } = confirming;
    setConfirming(null);
    const { response } = await api.DELETE("/api/tags/{id}/shares/{userId}", {
      params: { path: { id: tagId, userId: share.user_id } },
    });
    if (!response.ok) return;
    if (share.user_id === user?.id) {
      navigate({ to: "/tags" });
      return;
    }
    setShares(
      (prev) => prev?.filter((s) => s.user_id !== share.user_id) ?? null,
    );
  }

  if (tag === undefined) {
    return null;
  }
  if (tag === null) {
    return (
      <section className="mx-auto flex w-full max-w-2xl flex-col gap-4 px-6 py-12 sm:px-10">
        <p className="text-sm text-zinc-500 dark:text-zinc-400">
          {t("tags.notFound")}
        </p>
      </section>
    );
  }

  const canManage = tag.permission === "owner";

  return (
    <section className="mx-auto flex w-full max-w-2xl flex-col gap-8 px-6 py-12 sm:px-10">
      <div className="flex flex-col gap-1">
        <h1 className="text-2xl font-semibold tracking-tight">
          {t("tags.sharing.title", { tag: tag.name })}
        </h1>
        <p className="text-sm text-zinc-500 dark:text-zinc-400">
          {t("tags.sharing.subtitle")}
        </p>
      </div>

      <ul className="flex flex-col gap-2">
        <li className="flex flex-wrap items-center justify-between gap-3 rounded-lg border border-black/10 px-4 py-3 dark:border-white/10">
          <div className="flex flex-col gap-0.5">
            <span className="font-medium">
              {tag.owner_name || t("tags.sharing.unknownUser")}
            </span>
            <span className="text-xs text-zinc-500 dark:text-zinc-400">
              {t("tags.sharing.realOwner")}
            </span>
          </div>
        </li>

        {shares?.map((share) => (
          <li
            key={share.user_id}
            className="flex flex-wrap items-center justify-between gap-3 rounded-lg border border-black/10 px-4 py-3 dark:border-white/10"
          >
            <div className="flex flex-col gap-0.5">
              <span className="font-medium">{share.name || share.email}</span>
              <span className="text-xs text-zinc-500 dark:text-zinc-400">
                {t("tags.sharing.grantedBy", {
                  name: share.granted_by_name,
                  date: formatDate(
                    share.created_at,
                    i18n.resolvedLanguage ?? "en",
                  ),
                })}
              </span>
            </div>
            <div className="flex items-center gap-2">
              {canManage ? (
                <select
                  value={share.permission}
                  onChange={(e) =>
                    updatePermission(
                      share.user_id,
                      e.target.value as TagPermission,
                    )
                  }
                  className={inputClass}
                >
                  {PERMISSIONS.map((p) => (
                    <option key={p} value={p}>
                      {t(`tags.sharing.permission.${p}`)}
                    </option>
                  ))}
                </select>
              ) : (
                <span className="text-sm text-zinc-600 dark:text-zinc-400">
                  {t(`tags.sharing.permission.${share.permission}`)}
                </span>
              )}
              {share.user_id === user?.id ? (
                <button
                  type="button"
                  onClick={() => setConfirming({ kind: "leave", share })}
                  className="rounded-md border border-red-200 px-3 py-2 text-sm font-medium text-red-600 transition-colors hover:bg-red-50 dark:border-red-900/50 dark:text-red-400 dark:hover:bg-red-950/30"
                >
                  {t("tags.sharing.leave")}
                </button>
              ) : (
                canManage && (
                  <button
                    type="button"
                    onClick={() => setConfirming({ kind: "revoke", share })}
                    className="rounded-md border border-red-200 px-3 py-2 text-sm font-medium text-red-600 transition-colors hover:bg-red-50 dark:border-red-900/50 dark:text-red-400 dark:hover:bg-red-950/30"
                  >
                    {t("tags.sharing.revoke")}
                  </button>
                )
              )}
            </div>
          </li>
        ))}

        {shares?.length === 0 && (
          <li className="rounded-lg border border-dashed border-black/15 px-4 py-6 text-center text-sm text-zinc-500 dark:border-white/15 dark:text-zinc-400">
            {t("tags.sharing.noShares")}
          </li>
        )}
      </ul>

      {canManage && (
        <section className="flex flex-col gap-3 border-t border-black/10 pt-6 dark:border-white/10">
          <h2 className="text-sm font-semibold text-zinc-500 dark:text-zinc-400">
            {t("tags.sharing.inviteTitle")}
          </h2>
          <form
            onSubmit={handleInvite}
            className="flex flex-wrap items-end gap-3"
          >
            <label className="flex flex-1 flex-col gap-1.5 text-sm font-medium">
              {t("tags.sharing.email")}
              <input
                type="email"
                value={inviteEmail}
                onChange={(e) => setInviteEmail(e.target.value)}
                className={inputClass}
                required
              />
            </label>
            <label className="flex flex-col gap-1.5 text-sm font-medium">
              {t("tags.sharing.permissionLabel")}
              <select
                value={invitePermission}
                onChange={(e) =>
                  setInvitePermission(e.target.value as TagPermission)
                }
                className={inputClass}
              >
                {PERMISSIONS.map((p) => (
                  <option key={p} value={p}>
                    {t(`tags.sharing.permission.${p}`)}
                  </option>
                ))}
              </select>
            </label>
            <button
              type="submit"
              disabled={inviting}
              className="rounded-md bg-zinc-900 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-zinc-700 disabled:opacity-60 dark:bg-white dark:text-zinc-900 dark:hover:bg-zinc-200"
            >
              {t("tags.sharing.inviteSubmit")}
            </button>
          </form>
          {inviteError && (
            <p className="text-sm text-red-600 dark:text-red-400">
              {inviteError}
            </p>
          )}
          {unmatched && (
            <div className="flex flex-col gap-2 rounded-md bg-black/[.04] px-4 py-3 text-sm dark:bg-white/[.06]">
              <p>{t("tags.sharing.unmatched", { email: unmatched.email })}</p>
              {unmatched.inviteAllowed ? (
                appInviteSent ? (
                  <p className="text-emerald-600 dark:text-emerald-400">
                    {t("tags.sharing.appInviteSent")}
                  </p>
                ) : (
                  <button
                    type="button"
                    onClick={sendAppInvite}
                    disabled={sendingAppInvite}
                    className="self-start rounded-md border border-black/15 px-3 py-1.5 text-sm font-medium transition-colors hover:bg-black/[.04] disabled:opacity-60 dark:border-white/15 dark:hover:bg-white/[.06]"
                  >
                    {t("tags.sharing.sendAppInvite")}
                  </button>
                )
              ) : (
                <p className="text-zinc-500 dark:text-zinc-400">
                  {t("tags.sharing.inviteUnavailable")}
                </p>
              )}
            </div>
          )}
        </section>
      )}

      <Dialog
        open={confirming !== null}
        onClose={() => setConfirming(null)}
        className="relative z-50"
      >
        <div className="fixed inset-0 bg-black/40" aria-hidden="true" />
        <div className="fixed inset-0 flex items-center justify-center p-4">
          <DialogPanel className="flex w-full max-w-sm flex-col gap-4 rounded-lg bg-white p-6 dark:bg-neutral-900">
            {confirming && (
              <>
                <DialogTitle className="text-base font-semibold">
                  {t(`tags.sharing.confirm.${confirming.kind}Title`)}
                </DialogTitle>
                <Description className="text-sm text-zinc-600 dark:text-zinc-400">
                  {t(`tags.sharing.confirm.${confirming.kind}Body`, {
                    name: confirming.share.name || confirming.share.email,
                  })}
                </Description>
                <div className="flex justify-end gap-2">
                  <button
                    type="button"
                    onClick={() => setConfirming(null)}
                    className="rounded-md px-3 py-2 text-sm font-medium text-zinc-600 hover:bg-black/[.04] dark:text-zinc-400 dark:hover:bg-white/[.06]"
                  >
                    {t("tags.confirm.cancel")}
                  </button>
                  <button
                    type="button"
                    onClick={performConfirmed}
                    className="rounded-md bg-red-600 px-3 py-2 text-sm font-medium text-white hover:bg-red-700"
                  >
                    {t("tags.confirm.confirmAction")}
                  </button>
                </div>
              </>
            )}
          </DialogPanel>
        </div>
      </Dialog>
    </section>
  );
}
