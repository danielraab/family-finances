import {
  Description,
  Dialog,
  DialogPanel,
  DialogTitle,
} from "@headlessui/react";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { type FormEvent, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { api } from "../api/client";
import type { components } from "../api/schema";
import { useAuth } from "../components/AuthProvider";
import { Avatar } from "../components/Avatar";

export const Route = createFileRoute("/settings/users")({
  component: UsersSettingsTab,
});

type AdminUser = components["schemas"]["AdminUser"];
type Invite = components["schemas"]["Invite"];
type ConfirmKind = "disable" | "enable" | "delete";

const inputClass =
  "rounded-md border border-black/15 bg-transparent px-3 py-2 text-sm font-normal outline-none transition-colors focus:border-black/40 dark:border-white/15 dark:focus:border-white/40";

function formatDate(iso: string, lang: string): string {
  return new Intl.DateTimeFormat(lang, {
    dateStyle: "medium",
  }).format(new Date(iso));
}

/**
 * The admin-only Users tab: lists users and invitations, can invite, and can
 * disable/enable/(soft) delete a user. Guards itself against a direct link
 * from a non-admin — the tab link itself is already hidden by
 * settings.tsx's tab list, but a bookmarked/typed URL still needs this.
 */
function UsersSettingsTab() {
  const { user, logout } = useAuth();
  const navigate = useNavigate();
  const { t, i18n } = useTranslation();

  const [users, setUsers] = useState<AdminUser[] | null>(null);
  const [invites, setInvites] = useState<Invite[] | null>(null);
  const [inviteEmail, setInviteEmail] = useState("");
  const [inviteError, setInviteError] = useState<string | null>(null);
  const [inviting, setInviting] = useState(false);
  const [confirming, setConfirming] = useState<{
    kind: ConfirmKind;
    target: AdminUser;
  } | null>(null);

  useEffect(() => {
    if (user && !user.is_admin) {
      navigate({ to: "/settings", replace: true });
    }
  }, [user, navigate]);

  useEffect(() => {
    if (!user?.is_admin) return;
    let cancelled = false;
    Promise.all([api.GET("/api/auth/users"), api.GET("/api/auth/invites")])
      .then(([usersRes, invitesRes]) => {
        if (cancelled) return;
        if (usersRes.data) setUsers(usersRes.data);
        if (invitesRes.data) setInvites(invitesRes.data);
      })
      .catch(() => {
        /* left as loading; a real error surface can follow later */
      });
    return () => {
      cancelled = true;
    };
  }, [user?.is_admin]);

  if (!user?.is_admin) {
    return null;
  }

  async function onInvite(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setInviting(true);
    setInviteError(null);
    const { data, response } = await api.POST("/api/auth/invites", {
      body: { email: inviteEmail.trim() },
    });
    setInviting(false);
    if (!response.ok || !data) {
      setInviteError(t("settings.users.inviteError"));
      return;
    }
    setInvites((prev) => [data, ...(prev ?? [])]);
    setInviteEmail("");
  }

  async function performConfirmed() {
    if (!confirming) return;
    const { kind, target } = confirming;
    setConfirming(null);
    const isSelf = target.id === user?.id;

    if (kind === "disable") {
      const { data } = await api.POST("/api/auth/users/{id}/disable", {
        params: { path: { id: target.id } },
      });
      if (data) {
        setUsers(
          (prev) => prev?.map((u) => (u.id === data.id ? data : u)) ?? null,
        );
      }
    } else if (kind === "enable") {
      const { data } = await api.POST("/api/auth/users/{id}/enable", {
        params: { path: { id: target.id } },
      });
      if (data) {
        setUsers(
          (prev) => prev?.map((u) => (u.id === data.id ? data : u)) ?? null,
        );
      }
    } else {
      const { response } = await api.DELETE("/api/auth/users/{id}", {
        params: { path: { id: target.id } },
      });
      if (response.ok) {
        setUsers((prev) => prev?.filter((u) => u.id !== target.id) ?? null);
      }
    }

    if (isSelf && (kind === "disable" || kind === "delete")) {
      await logout();
      navigate({ to: "/", replace: true });
    }
  }

  return (
    <div className="flex flex-col gap-8">
      <section className="flex flex-col gap-3">
        <h2 className="text-sm font-semibold text-zinc-500 dark:text-zinc-400">
          {t("settings.users.usersHeading")}
        </h2>

        <form onSubmit={onInvite} className="flex items-end gap-2">
          <label className="flex flex-1 flex-col gap-1.5 text-sm font-medium">
            {t("settings.users.inviteEmailLabel")}
            <input
              type="email"
              required
              value={inviteEmail}
              onChange={(event) => setInviteEmail(event.target.value)}
              className={inputClass}
            />
          </label>
          <button
            type="submit"
            disabled={inviting}
            className="rounded-md bg-zinc-900 px-3 py-2 text-sm font-medium text-white transition-colors hover:bg-zinc-700 disabled:opacity-60 dark:bg-white dark:text-zinc-900 dark:hover:bg-zinc-200"
          >
            {inviting
              ? t("settings.users.inviting")
              : t("settings.users.invite")}
          </button>
        </form>
        {inviteError && (
          <p className="text-sm text-red-600 dark:text-red-400">
            {inviteError}
          </p>
        )}

        <div className="overflow-x-auto rounded-lg border border-black/10 dark:border-white/10">
          <table className="w-full text-left text-sm">
            <thead className="border-b border-black/10 text-xs uppercase text-zinc-500 dark:border-white/10 dark:text-zinc-400">
              <tr>
                <th className="px-3 py-2 font-medium">
                  {t("settings.users.columnUser")}
                </th>
                <th className="px-3 py-2 font-medium">
                  {t("settings.users.columnRole")}
                </th>
                <th className="px-3 py-2 font-medium">
                  {t("settings.users.columnStatus")}
                </th>
                <th className="px-3 py-2 font-medium">
                  {t("settings.users.columnCreated")}
                </th>
                <th className="px-3 py-2" />
              </tr>
            </thead>
            <tbody>
              {users?.map((u) => (
                <tr
                  key={u.id}
                  className="border-b border-black/5 last:border-0 dark:border-white/5"
                >
                  <td className="flex items-center gap-2 px-3 py-2">
                    <Avatar
                      id={u.id}
                      email={u.email}
                      displayName={u.display_name}
                      size={24}
                    />
                    <span className="flex flex-col leading-tight">
                      {u.display_name && (
                        <span className="text-zinc-900 dark:text-zinc-100">
                          {u.display_name}
                        </span>
                      )}
                      <span className="text-xs text-zinc-500 dark:text-zinc-400">
                        {u.email}
                      </span>
                    </span>
                  </td>
                  <td className="px-3 py-2">
                    {u.is_admin
                      ? t("settings.users.roleAdmin")
                      : t("settings.users.roleMember")}
                  </td>
                  <td className="px-3 py-2">
                    {u.disabled
                      ? t("settings.users.statusDisabled")
                      : t("settings.users.statusActive")}
                  </td>
                  <td className="px-3 py-2 text-zinc-500 dark:text-zinc-400">
                    {formatDate(u.created_at, i18n.resolvedLanguage ?? "en")}
                  </td>
                  <td className="px-3 py-2 text-right">
                    <div className="flex justify-end gap-2">
                      {u.disabled ? (
                        <button
                          type="button"
                          onClick={() =>
                            setConfirming({ kind: "enable", target: u })
                          }
                          className="text-xs font-medium text-zinc-600 underline underline-offset-2 hover:text-zinc-900 dark:text-zinc-400 dark:hover:text-zinc-100"
                        >
                          {t("settings.users.enable")}
                        </button>
                      ) : (
                        <button
                          type="button"
                          onClick={() =>
                            setConfirming({ kind: "disable", target: u })
                          }
                          className="text-xs font-medium text-zinc-600 underline underline-offset-2 hover:text-zinc-900 dark:text-zinc-400 dark:hover:text-zinc-100"
                        >
                          {t("settings.users.disable")}
                        </button>
                      )}
                      <button
                        type="button"
                        onClick={() =>
                          setConfirming({ kind: "delete", target: u })
                        }
                        className="text-xs font-medium text-red-600 underline underline-offset-2 hover:text-red-800 dark:text-red-400 dark:hover:text-red-300"
                      >
                        {t("settings.users.delete")}
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </section>

      <section className="flex flex-col gap-3">
        <h2 className="text-sm font-semibold text-zinc-500 dark:text-zinc-400">
          {t("settings.users.invitesHeading")}
        </h2>
        <ul className="flex flex-col gap-2">
          {invites?.map((invite) => (
            <li
              key={invite.id}
              className="flex flex-col gap-0.5 rounded-md border border-black/10 px-3 py-2 text-sm dark:border-white/10"
            >
              <span className="font-medium">{invite.email}</span>
              <span className="text-xs text-zinc-500 dark:text-zinc-400">
                {t("settings.users.invitedBy", {
                  name:
                    invite.invited_by.display_name || invite.invited_by.email,
                })}
                {" · "}
                {invite.accepted_at
                  ? t("settings.users.accepted", {
                      date: formatDate(
                        invite.accepted_at,
                        i18n.resolvedLanguage ?? "en",
                      ),
                    })
                  : `${t("settings.users.pending")} · ${t(
                      "settings.users.expires",
                      {
                        date: formatDate(
                          invite.expires_at,
                          i18n.resolvedLanguage ?? "en",
                        ),
                      },
                    )}`}
              </span>
            </li>
          ))}
        </ul>
      </section>

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
                  {t(`settings.users.confirm.${confirming.kind}Title`, {
                    email: confirming.target.email,
                  })}
                </DialogTitle>
                <Description className="text-sm text-zinc-600 dark:text-zinc-400">
                  {t(`settings.users.confirm.${confirming.kind}Body`)}
                  {confirming.target.id === user.id &&
                    confirming.kind !== "enable" && (
                      <span className="mt-2 block font-medium text-amber-700 dark:text-amber-400">
                        {t("settings.users.confirm.selfWarning")}
                      </span>
                    )}
                </Description>
                <div className="flex justify-end gap-2">
                  <button
                    type="button"
                    onClick={() => setConfirming(null)}
                    className="rounded-md px-3 py-2 text-sm font-medium text-zinc-600 hover:bg-black/[.04] dark:text-zinc-400 dark:hover:bg-white/[.06]"
                  >
                    {t("settings.users.confirm.cancel")}
                  </button>
                  <button
                    type="button"
                    onClick={performConfirmed}
                    className="rounded-md bg-zinc-900 px-3 py-2 text-sm font-medium text-white hover:bg-zinc-700 dark:bg-white dark:text-zinc-900 dark:hover:bg-zinc-200"
                  >
                    {t("settings.users.confirm.confirmAction")}
                  </button>
                </div>
              </>
            )}
          </DialogPanel>
        </div>
      </Dialog>
    </div>
  );
}
