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

export const Route = createFileRoute("/accounts/$accountId/sharing")({
  component: SharingPage,
});

type Account = components["schemas"]["Account"];
type AccountShare = components["schemas"]["AccountShare"];
type AccountPermission = components["schemas"]["AccountPermission"];

const PERMISSIONS: AccountPermission[] = [
  "view",
  "append",
  "entry_admin",
  "owner",
];

const inputClass =
  "rounded-md border border-black/15 bg-transparent px-3 py-2 text-sm font-normal outline-none transition-colors focus:border-black/40 dark:border-white/15 dark:focus:border-white/40";

function formatDate(iso: string, lang: string): string {
  return new Intl.DateTimeFormat(lang, { dateStyle: "medium" }).format(
    new Date(iso),
  );
}

function SharingPage() {
  const { accountId } = Route.useParams();
  const { t, i18n } = useTranslation();
  const { user } = useAuth();
  const navigate = useNavigate();

  const [account, setAccount] = useState<Account | null | undefined>(undefined);
  const [shares, setShares] = useState<AccountShare[] | null>(null);

  const [inviteEmail, setInviteEmail] = useState("");
  const [invitePermission, setInvitePermission] =
    useState<AccountPermission>("view");
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
    share: AccountShare;
  } | null>(null);

  const load = useCallback(() => {
    let cancelled = false;
    api
      .GET("/api/accounts/{id}", { params: { path: { id: accountId } } })
      .then(({ data }) => {
        if (!cancelled) setAccount(data ?? null);
      });
    api
      .GET("/api/accounts/{id}/shares", {
        params: { path: { id: accountId } },
      })
      .then(({ data }) => {
        if (!cancelled) setShares(data ?? []);
      });
    return () => {
      cancelled = true;
    };
  }, [accountId]);

  useEffect(() => load(), [load]);

  async function handleInvite(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setInviting(true);
    setInviteError(null);
    setUnmatched(null);
    setAppInviteSent(false);
    const email = inviteEmail.trim();
    const { data, response } = await api.POST("/api/accounts/{id}/shares", {
      params: { path: { id: accountId } },
      body: { email, permission: invitePermission },
    });
    setInviting(false);
    if (!response.ok || !data) {
      setInviteError(t("accounts.sharing.inviteError"));
      return;
    }
    if (data.matched && data.share) {
      setShares((prev) => [...(prev ?? []), data.share as AccountShare]);
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

  async function updatePermission(
    userId: string,
    permission: AccountPermission,
  ) {
    const { data } = await api.PATCH("/api/accounts/{id}/shares/{userId}", {
      params: { path: { id: accountId, userId } },
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
    const { response } = await api.DELETE(
      "/api/accounts/{id}/shares/{userId}",
      { params: { path: { id: accountId, userId: share.user_id } } },
    );
    if (!response.ok) return;
    if (share.user_id === user?.id) {
      navigate({ to: "/accounts" });
      return;
    }
    setShares(
      (prev) => prev?.filter((s) => s.user_id !== share.user_id) ?? null,
    );
  }

  if (account === undefined) {
    return null;
  }
  if (account === null) {
    return (
      <section className="mx-auto flex w-full max-w-2xl flex-col gap-4 px-6 py-12 sm:px-10">
        <p className="text-sm text-zinc-500 dark:text-zinc-400">
          {t("accounts.notFound")}
        </p>
      </section>
    );
  }

  const canManage = account.permission === "owner";

  return (
    <section className="mx-auto flex w-full max-w-2xl flex-col gap-8 px-6 py-12 sm:px-10">
      <div className="flex flex-col gap-1">
        <h1 className="text-2xl font-semibold tracking-tight">
          {t("accounts.sharing.title", { account: account.title })}
        </h1>
        <p className="text-sm text-zinc-500 dark:text-zinc-400">
          {t("accounts.sharing.subtitle")}
        </p>
      </div>

      <ul className="flex flex-col gap-2">
        <li className="flex flex-wrap items-center justify-between gap-3 rounded-lg border border-black/10 px-4 py-3 dark:border-white/10">
          <div className="flex flex-col gap-0.5">
            <span className="font-medium">
              {account.owner_name || t("accounts.sharing.unknownUser")}
            </span>
            <span className="text-xs text-zinc-500 dark:text-zinc-400">
              {t("accounts.sharing.realOwner")}
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
                {t("accounts.sharing.grantedBy", {
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
                      e.target.value as AccountPermission,
                    )
                  }
                  className={inputClass}
                >
                  {PERMISSIONS.map((p) => (
                    <option key={p} value={p}>
                      {t(`accounts.sharing.permission.${p}`)}
                    </option>
                  ))}
                </select>
              ) : (
                <span className="text-sm text-zinc-600 dark:text-zinc-400">
                  {t(`accounts.sharing.permission.${share.permission}`)}
                </span>
              )}
              {share.user_id === user?.id ? (
                <button
                  type="button"
                  onClick={() => setConfirming({ kind: "leave", share })}
                  className="rounded-md border border-red-200 px-3 py-2 text-sm font-medium text-red-600 transition-colors hover:bg-red-50 dark:border-red-900/50 dark:text-red-400 dark:hover:bg-red-950/30"
                >
                  {t("accounts.sharing.leave")}
                </button>
              ) : (
                canManage && (
                  <button
                    type="button"
                    onClick={() => setConfirming({ kind: "revoke", share })}
                    className="rounded-md border border-red-200 px-3 py-2 text-sm font-medium text-red-600 transition-colors hover:bg-red-50 dark:border-red-900/50 dark:text-red-400 dark:hover:bg-red-950/30"
                  >
                    {t("accounts.sharing.revoke")}
                  </button>
                )
              )}
            </div>
          </li>
        ))}

        {shares?.length === 0 && (
          <li className="rounded-lg border border-dashed border-black/15 px-4 py-6 text-center text-sm text-zinc-500 dark:border-white/15 dark:text-zinc-400">
            {t("accounts.sharing.noShares")}
          </li>
        )}
      </ul>

      {canManage && (
        <section className="flex flex-col gap-3 border-t border-black/10 pt-6 dark:border-white/10">
          <h2 className="text-sm font-semibold text-zinc-500 dark:text-zinc-400">
            {t("accounts.sharing.inviteTitle")}
          </h2>
          <form
            onSubmit={handleInvite}
            className="flex flex-wrap items-end gap-3"
          >
            <label className="flex flex-1 flex-col gap-1.5 text-sm font-medium">
              {t("accounts.sharing.email")}
              <input
                type="email"
                value={inviteEmail}
                onChange={(e) => setInviteEmail(e.target.value)}
                className={inputClass}
                required
              />
            </label>
            <label className="flex flex-col gap-1.5 text-sm font-medium">
              {t("accounts.sharing.permissionLabel")}
              <select
                value={invitePermission}
                onChange={(e) =>
                  setInvitePermission(e.target.value as AccountPermission)
                }
                className={inputClass}
              >
                {PERMISSIONS.map((p) => (
                  <option key={p} value={p}>
                    {t(`accounts.sharing.permission.${p}`)}
                  </option>
                ))}
              </select>
            </label>
            <button
              type="submit"
              disabled={inviting}
              className="rounded-md bg-zinc-900 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-zinc-700 disabled:opacity-60 dark:bg-white dark:text-zinc-900 dark:hover:bg-zinc-200"
            >
              {t("accounts.sharing.inviteSubmit")}
            </button>
          </form>
          {inviteError && (
            <p className="text-sm text-red-600 dark:text-red-400">
              {inviteError}
            </p>
          )}
          {unmatched && (
            <div className="flex flex-col gap-2 rounded-md bg-black/[.04] px-4 py-3 text-sm dark:bg-white/[.06]">
              <p>
                {t("accounts.sharing.unmatched", { email: unmatched.email })}
              </p>
              {unmatched.inviteAllowed ? (
                appInviteSent ? (
                  <p className="text-emerald-600 dark:text-emerald-400">
                    {t("accounts.sharing.appInviteSent")}
                  </p>
                ) : (
                  <button
                    type="button"
                    onClick={sendAppInvite}
                    disabled={sendingAppInvite}
                    className="self-start rounded-md border border-black/15 px-3 py-1.5 text-sm font-medium transition-colors hover:bg-black/[.04] disabled:opacity-60 dark:border-white/15 dark:hover:bg-white/[.06]"
                  >
                    {t("accounts.sharing.sendAppInvite")}
                  </button>
                )
              ) : (
                <p className="text-zinc-500 dark:text-zinc-400">
                  {t("accounts.sharing.inviteUnavailable")}
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
                  {t(`accounts.sharing.confirm.${confirming.kind}Title`)}
                </DialogTitle>
                <Description className="text-sm text-zinc-600 dark:text-zinc-400">
                  {t(`accounts.sharing.confirm.${confirming.kind}Body`, {
                    name: confirming.share.name || confirming.share.email,
                  })}
                </Description>
                <div className="flex justify-end gap-2">
                  <button
                    type="button"
                    onClick={() => setConfirming(null)}
                    className="rounded-md px-3 py-2 text-sm font-medium text-zinc-600 hover:bg-black/[.04] dark:text-zinc-400 dark:hover:bg-white/[.06]"
                  >
                    {t("accounts.edit.confirm.cancel")}
                  </button>
                  <button
                    type="button"
                    onClick={performConfirmed}
                    className="rounded-md bg-red-600 px-3 py-2 text-sm font-medium text-white hover:bg-red-700"
                  >
                    {t("accounts.edit.confirm.confirmAction")}
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
