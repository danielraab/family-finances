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
import { compact } from "../lib/compact";

export const Route = createFileRoute("/settings/account-types")({
  component: AccountTypesSettingsTab,
});

type AccountType = components["schemas"]["AccountType"];
type ConfirmKind = "disable" | "enable" | "delete";

const inputClass =
  "rounded-md border border-black/15 bg-transparent px-3 py-2 text-sm font-normal outline-none transition-colors focus:border-black/40 dark:border-white/15 dark:focus:border-white/40";

/**
 * The admin-only Account Types tab: lists account types and can
 * create/edit/disable/enable/(hard) delete one. Guards itself against a
 * direct link from a non-admin — the tab link itself is already hidden by
 * settings.tsx's tab list, but a bookmarked/typed URL still needs this.
 */
function AccountTypesSettingsTab() {
  const { user } = useAuth();
  const navigate = useNavigate();
  const { t } = useTranslation();

  const [types, setTypes] = useState<AccountType[] | null>(null);

  const [createTitle, setCreateTitle] = useState("");
  const [createDescription, setCreateDescription] = useState("");
  const [creating, setCreating] = useState(false);
  const [createError, setCreateError] = useState<string | null>(null);

  const [editing, setEditing] = useState<AccountType | null>(null);
  const [editTitle, setEditTitle] = useState("");
  const [editDescription, setEditDescription] = useState("");
  const [saving, setSaving] = useState(false);
  const [editError, setEditError] = useState<string | null>(null);

  const [confirming, setConfirming] = useState<{
    kind: ConfirmKind;
    target: AccountType;
  } | null>(null);
  const [deleteError, setDeleteError] = useState<string | null>(null);

  useEffect(() => {
    if (user && !user.is_admin) {
      navigate({ to: "/settings", replace: true });
    }
  }, [user, navigate]);

  useEffect(() => {
    if (!user?.is_admin) return;
    let cancelled = false;
    api.GET("/api/account-types").then(({ data }) => {
      if (cancelled || !data) return;
      setTypes(data);
    });
    return () => {
      cancelled = true;
    };
  }, [user?.is_admin]);

  if (!user?.is_admin) {
    return null;
  }

  async function onCreate(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setCreating(true);
    setCreateError(null);
    const { data, response } = await api.POST("/api/account-types", {
      body: {
        title: createTitle.trim(),
        ...compact({ description: createDescription.trim() || undefined }),
      },
    });
    setCreating(false);
    if (!response.ok || !data) {
      setCreateError(t("settings.accountTypes.createError"));
      return;
    }
    setTypes((prev) => [...(prev ?? []), data]);
    setCreateTitle("");
    setCreateDescription("");
  }

  function openEdit(type: AccountType) {
    setEditing(type);
    setEditTitle(type.title);
    setEditDescription(type.description ?? "");
    setEditError(null);
  }

  async function onSaveEdit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!editing) return;
    setSaving(true);
    setEditError(null);
    const { data, response } = await api.PATCH("/api/account-types/{id}", {
      params: { path: { id: editing.id } },
      body: {
        title: editTitle.trim(),
        ...compact({ description: editDescription.trim() || undefined }),
      },
    });
    setSaving(false);
    if (!response.ok || !data) {
      setEditError(t("settings.accountTypes.editError"));
      return;
    }
    setTypes(
      (prev) => prev?.map((ty) => (ty.id === data.id ? data : ty)) ?? null,
    );
    setEditing(null);
  }

  async function performConfirmed() {
    if (!confirming) return;
    const { kind, target } = confirming;
    setConfirming(null);
    setDeleteError(null);

    if (kind === "disable") {
      const { data } = await api.POST("/api/account-types/{id}/disable", {
        params: { path: { id: target.id } },
      });
      if (data) {
        setTypes(
          (prev) => prev?.map((ty) => (ty.id === data.id ? data : ty)) ?? null,
        );
      }
      return;
    }

    if (kind === "enable") {
      const { data } = await api.POST("/api/account-types/{id}/enable", {
        params: { path: { id: target.id } },
      });
      if (data) {
        setTypes(
          (prev) => prev?.map((ty) => (ty.id === data.id ? data : ty)) ?? null,
        );
      }
      return;
    }

    const { response } = await api.DELETE("/api/account-types/{id}", {
      params: { path: { id: target.id } },
    });
    if (response.ok) {
      setTypes((prev) => prev?.filter((ty) => ty.id !== target.id) ?? null);
    } else {
      setDeleteError(
        t("settings.accountTypes.deleteError", { title: target.title }),
      );
    }
  }

  return (
    <div className="flex flex-col gap-8">
      <section className="flex flex-col gap-3">
        <h2 className="text-sm font-semibold text-zinc-500 dark:text-zinc-400">
          {t("settings.accountTypes.heading")}
        </h2>

        <form
          onSubmit={onCreate}
          className="flex flex-col gap-2 sm:flex-row sm:items-end"
        >
          <label className="flex flex-1 flex-col gap-1.5 text-sm font-medium">
            {t("settings.accountTypes.titleLabel")}
            <input
              required
              value={createTitle}
              onChange={(event) => setCreateTitle(event.target.value)}
              className={inputClass}
            />
          </label>
          <label className="flex flex-1 flex-col gap-1.5 text-sm font-medium">
            {t("settings.accountTypes.descriptionLabel")}
            <input
              value={createDescription}
              onChange={(event) => setCreateDescription(event.target.value)}
              className={inputClass}
            />
          </label>
          <button
            type="submit"
            disabled={creating}
            className="rounded-md bg-zinc-900 px-3 py-2 text-sm font-medium text-white transition-colors hover:bg-zinc-700 disabled:opacity-60 dark:bg-white dark:text-zinc-900 dark:hover:bg-zinc-200"
          >
            {creating
              ? t("settings.accountTypes.creating")
              : t("settings.accountTypes.create")}
          </button>
        </form>
        {createError && (
          <p className="text-sm text-red-600 dark:text-red-400">
            {createError}
          </p>
        )}
        {deleteError && (
          <p className="text-sm text-red-600 dark:text-red-400">
            {deleteError}
          </p>
        )}

        {types !== null && types.length === 0 ? (
          <p className="text-sm text-zinc-500 dark:text-zinc-400">
            {t("settings.accountTypes.empty")}
          </p>
        ) : (
          <div className="overflow-x-auto rounded-lg border border-black/10 dark:border-white/10">
            <table className="w-full text-left text-sm">
              <thead className="border-b border-black/10 text-xs uppercase text-zinc-500 dark:border-white/10 dark:text-zinc-400">
                <tr>
                  <th className="px-3 py-2 font-medium">
                    {t("settings.accountTypes.columnTitle")}
                  </th>
                  <th className="px-3 py-2 font-medium">
                    {t("settings.accountTypes.columnDescription")}
                  </th>
                  <th className="px-3 py-2 font-medium">
                    {t("settings.accountTypes.columnStatus")}
                  </th>
                  <th className="px-3 py-2" />
                </tr>
              </thead>
              <tbody>
                {types?.map((ty) => (
                  <tr
                    key={ty.id}
                    className="border-b border-black/5 last:border-0 dark:border-white/5"
                  >
                    <td className="px-3 py-2 text-zinc-900 dark:text-zinc-100">
                      {ty.title}
                    </td>
                    <td className="px-3 py-2 text-zinc-500 dark:text-zinc-400">
                      {ty.description}
                    </td>
                    <td className="px-3 py-2">
                      {ty.disabled
                        ? t("settings.accountTypes.statusDisabled")
                        : t("settings.accountTypes.statusActive")}
                    </td>
                    <td className="px-3 py-2 text-right">
                      <div className="flex justify-end gap-2">
                        <button
                          type="button"
                          onClick={() => openEdit(ty)}
                          className="text-xs font-medium text-zinc-600 underline underline-offset-2 hover:text-zinc-900 dark:text-zinc-400 dark:hover:text-zinc-100"
                        >
                          {t("settings.accountTypes.edit")}
                        </button>
                        {ty.disabled ? (
                          <button
                            type="button"
                            onClick={() =>
                              setConfirming({ kind: "enable", target: ty })
                            }
                            className="text-xs font-medium text-zinc-600 underline underline-offset-2 hover:text-zinc-900 dark:text-zinc-400 dark:hover:text-zinc-100"
                          >
                            {t("settings.accountTypes.enable")}
                          </button>
                        ) : (
                          <button
                            type="button"
                            onClick={() =>
                              setConfirming({ kind: "disable", target: ty })
                            }
                            className="text-xs font-medium text-zinc-600 underline underline-offset-2 hover:text-zinc-900 dark:text-zinc-400 dark:hover:text-zinc-100"
                          >
                            {t("settings.accountTypes.disable")}
                          </button>
                        )}
                        <button
                          type="button"
                          onClick={() =>
                            setConfirming({ kind: "delete", target: ty })
                          }
                          className="text-xs font-medium text-red-600 underline underline-offset-2 hover:text-red-800 dark:text-red-400 dark:hover:text-red-300"
                        >
                          {t("settings.accountTypes.delete")}
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>

      <Dialog
        open={editing !== null}
        onClose={() => setEditing(null)}
        className="relative z-50"
      >
        <div className="fixed inset-0 bg-black/40" aria-hidden="true" />
        <div className="fixed inset-0 flex items-center justify-center p-4">
          <DialogPanel className="flex w-full max-w-sm flex-col gap-4 rounded-lg bg-white p-6 dark:bg-neutral-900">
            {editing && (
              <form onSubmit={onSaveEdit} className="flex flex-col gap-4">
                <DialogTitle className="text-base font-semibold">
                  {t("settings.accountTypes.editTitle")}
                </DialogTitle>
                <label className="flex flex-col gap-1.5 text-sm font-medium">
                  {t("settings.accountTypes.titleLabel")}
                  <input
                    required
                    value={editTitle}
                    onChange={(event) => setEditTitle(event.target.value)}
                    className={inputClass}
                  />
                </label>
                <label className="flex flex-col gap-1.5 text-sm font-medium">
                  {t("settings.accountTypes.descriptionLabel")}
                  <input
                    value={editDescription}
                    onChange={(event) => setEditDescription(event.target.value)}
                    className={inputClass}
                  />
                </label>
                {editError && (
                  <p className="text-sm text-red-600 dark:text-red-400">
                    {editError}
                  </p>
                )}
                <div className="flex justify-end gap-2">
                  <button
                    type="button"
                    onClick={() => setEditing(null)}
                    className="rounded-md px-3 py-2 text-sm font-medium text-zinc-600 hover:bg-black/[.04] dark:text-zinc-400 dark:hover:bg-white/[.06]"
                  >
                    {t("settings.accountTypes.cancel")}
                  </button>
                  <button
                    type="submit"
                    disabled={saving}
                    className="rounded-md bg-zinc-900 px-3 py-2 text-sm font-medium text-white hover:bg-zinc-700 disabled:opacity-60 dark:bg-white dark:text-zinc-900 dark:hover:bg-zinc-200"
                  >
                    {saving
                      ? t("settings.accountTypes.saving")
                      : t("settings.accountTypes.save")}
                  </button>
                </div>
              </form>
            )}
          </DialogPanel>
        </div>
      </Dialog>

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
                  {t(`settings.accountTypes.confirm.${confirming.kind}Title`, {
                    title: confirming.target.title,
                  })}
                </DialogTitle>
                <Description className="text-sm text-zinc-600 dark:text-zinc-400">
                  {t(`settings.accountTypes.confirm.${confirming.kind}Body`)}
                </Description>
                <div className="flex justify-end gap-2">
                  <button
                    type="button"
                    onClick={() => setConfirming(null)}
                    className="rounded-md px-3 py-2 text-sm font-medium text-zinc-600 hover:bg-black/[.04] dark:text-zinc-400 dark:hover:bg-white/[.06]"
                  >
                    {t("settings.accountTypes.cancel")}
                  </button>
                  <button
                    type="button"
                    onClick={performConfirmed}
                    className="rounded-md bg-zinc-900 px-3 py-2 text-sm font-medium text-white hover:bg-zinc-700 dark:bg-white dark:text-zinc-900 dark:hover:bg-zinc-200"
                  >
                    {t("settings.accountTypes.confirm.confirmAction")}
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
