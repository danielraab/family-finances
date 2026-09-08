import {
  Description,
  Dialog,
  DialogPanel,
  DialogTitle,
} from "@headlessui/react";
import { createFileRoute } from "@tanstack/react-router";
import { type FormEvent, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { api } from "../api/client";
import type { components } from "../api/schema";
import { useAuth } from "../components/AuthProvider";

export const Route = createFileRoute("/settings/tags")({
  component: TagsSettingsTab,
});

type Tag = components["schemas"]["Tag"];

const inputClass =
  "rounded-md border border-black/15 bg-transparent px-3 py-2 text-sm font-normal outline-none transition-colors focus:border-black/40 dark:border-white/15 dark:focus:border-white/40";

/**
 * The Tags tab: lists the caller's own tags with how many of their entries
 * carry each one, and can create/rename/disable/enable/delete one. Open to
 * every authenticated user — the backend scopes everything to the caller.
 * Unlike the Account Types tab, disable/enable apply immediately (cheaply
 * reversible, no effect on existing entries); only delete is confirmed,
 * since it detaches the tag from every entry that carries it.
 */
function TagsSettingsTab() {
  const { user } = useAuth();
  const { t } = useTranslation();

  const [tags, setTags] = useState<Tag[] | null>(null);

  const [createName, setCreateName] = useState("");
  const [creating, setCreating] = useState(false);
  const [createError, setCreateError] = useState<string | null>(null);

  const [editing, setEditing] = useState<Tag | null>(null);
  const [editName, setEditName] = useState("");
  const [saving, setSaving] = useState(false);
  const [editError, setEditError] = useState<string | null>(null);

  const [confirmingDelete, setConfirmingDelete] = useState<Tag | null>(null);

  useEffect(() => {
    if (!user) return;
    let cancelled = false;
    api.GET("/api/tags").then(({ data }) => {
      if (cancelled || !data) return;
      setTags(data);
    });
    return () => {
      cancelled = true;
    };
  }, [user]);

  if (!user) {
    return null;
  }

  async function onCreate(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setCreating(true);
    setCreateError(null);
    const { data, response } = await api.POST("/api/tags", {
      body: { name: createName.trim() },
    });
    setCreating(false);
    if (!response.ok || !data) {
      setCreateError(t("settings.tags.createError"));
      return;
    }
    setTags((prev) => [...(prev ?? []), data]);
    setCreateName("");
  }

  function openEdit(tg: Tag) {
    setEditing(tg);
    setEditName(tg.name);
    setEditError(null);
  }

  async function onSaveEdit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!editing) return;
    setSaving(true);
    setEditError(null);
    const { data, response } = await api.PATCH("/api/tags/{id}", {
      params: { path: { id: editing.id } },
      body: { name: editName.trim() },
    });
    setSaving(false);
    if (!response.ok || !data) {
      setEditError(t("settings.tags.editError"));
      return;
    }
    setTags(
      (prev) => prev?.map((tg) => (tg.id === data.id ? data : tg)) ?? null,
    );
    setEditing(null);
  }

  async function onToggleDisabled(tg: Tag) {
    const path = tg.disabled
      ? "/api/tags/{id}/enable"
      : "/api/tags/{id}/disable";
    const { data } = await api.POST(path, { params: { path: { id: tg.id } } });
    if (data) {
      setTags(
        (prev) => prev?.map((t2) => (t2.id === data.id ? data : t2)) ?? null,
      );
    }
  }

  async function onConfirmDelete() {
    if (!confirmingDelete) return;
    const target = confirmingDelete;
    setConfirmingDelete(null);
    const { response } = await api.DELETE("/api/tags/{id}", {
      params: { path: { id: target.id } },
    });
    if (response.ok) {
      setTags((prev) => prev?.filter((tg) => tg.id !== target.id) ?? null);
    }
  }

  return (
    <div className="flex flex-col gap-8">
      <section className="flex flex-col gap-3">
        <h2 className="text-sm font-semibold text-zinc-500 dark:text-zinc-400">
          {t("settings.tags.heading")}
        </h2>

        <form
          onSubmit={onCreate}
          className="flex flex-col gap-2 sm:flex-row sm:items-end"
        >
          <label className="flex flex-1 flex-col gap-1.5 text-sm font-medium">
            {t("settings.tags.nameLabel")}
            <input
              required
              value={createName}
              onChange={(event) => setCreateName(event.target.value)}
              className={inputClass}
            />
          </label>
          <button
            type="submit"
            disabled={creating}
            className="rounded-md bg-zinc-900 px-3 py-2 text-sm font-medium text-white transition-colors hover:bg-zinc-700 disabled:opacity-60 dark:bg-white dark:text-zinc-900 dark:hover:bg-zinc-200"
          >
            {creating ? t("settings.tags.creating") : t("settings.tags.create")}
          </button>
        </form>
        {createError && (
          <p className="text-sm text-red-600 dark:text-red-400">
            {createError}
          </p>
        )}

        {tags !== null && tags.length === 0 ? (
          <p className="text-sm text-zinc-500 dark:text-zinc-400">
            {t("settings.tags.empty")}
          </p>
        ) : (
          <div className="overflow-x-auto rounded-lg border border-black/10 dark:border-white/10">
            <table className="w-full text-left text-sm">
              <thead className="border-b border-black/10 text-xs uppercase text-zinc-500 dark:border-white/10 dark:text-zinc-400">
                <tr>
                  <th className="px-3 py-2 font-medium">
                    {t("settings.tags.columnName")}
                  </th>
                  <th className="px-3 py-2 font-medium">
                    {t("settings.tags.columnEntries")}
                  </th>
                  <th className="px-3 py-2 font-medium">
                    {t("settings.tags.columnStatus")}
                  </th>
                  <th className="px-3 py-2" />
                </tr>
              </thead>
              <tbody>
                {tags?.map((tg) => (
                  <tr
                    key={tg.id}
                    className="border-b border-black/5 last:border-0 dark:border-white/5"
                  >
                    <td className="px-3 py-2 text-zinc-900 dark:text-zinc-100">
                      {tg.name}
                    </td>
                    <td className="px-3 py-2 text-zinc-500 dark:text-zinc-400">
                      {tg.entry_count}
                    </td>
                    <td className="px-3 py-2">
                      {tg.disabled
                        ? t("settings.tags.statusDisabled")
                        : t("settings.tags.statusActive")}
                    </td>
                    <td className="px-3 py-2 text-right">
                      <div className="flex justify-end gap-2">
                        <button
                          type="button"
                          onClick={() => openEdit(tg)}
                          className="text-xs font-medium text-zinc-600 underline underline-offset-2 hover:text-zinc-900 dark:text-zinc-400 dark:hover:text-zinc-100"
                        >
                          {t("settings.tags.edit")}
                        </button>
                        <button
                          type="button"
                          onClick={() => onToggleDisabled(tg)}
                          className="text-xs font-medium text-zinc-600 underline underline-offset-2 hover:text-zinc-900 dark:text-zinc-400 dark:hover:text-zinc-100"
                        >
                          {tg.disabled
                            ? t("settings.tags.enable")
                            : t("settings.tags.disable")}
                        </button>
                        <button
                          type="button"
                          onClick={() => setConfirmingDelete(tg)}
                          className="text-xs font-medium text-red-600 underline underline-offset-2 hover:text-red-800 dark:text-red-400 dark:hover:text-red-300"
                        >
                          {t("settings.tags.delete")}
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
                  {t("settings.tags.editTitle")}
                </DialogTitle>
                <label className="flex flex-col gap-1.5 text-sm font-medium">
                  {t("settings.tags.nameLabel")}
                  <input
                    required
                    value={editName}
                    onChange={(event) => setEditName(event.target.value)}
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
                    {t("settings.tags.cancel")}
                  </button>
                  <button
                    type="submit"
                    disabled={saving}
                    className="rounded-md bg-zinc-900 px-3 py-2 text-sm font-medium text-white hover:bg-zinc-700 disabled:opacity-60 dark:bg-white dark:text-zinc-900 dark:hover:bg-zinc-200"
                  >
                    {saving
                      ? t("settings.tags.saving")
                      : t("settings.tags.save")}
                  </button>
                </div>
              </form>
            )}
          </DialogPanel>
        </div>
      </Dialog>

      <Dialog
        open={confirmingDelete !== null}
        onClose={() => setConfirmingDelete(null)}
        className="relative z-50"
      >
        <div className="fixed inset-0 bg-black/40" aria-hidden="true" />
        <div className="fixed inset-0 flex items-center justify-center p-4">
          <DialogPanel className="flex w-full max-w-sm flex-col gap-4 rounded-lg bg-white p-6 dark:bg-neutral-900">
            {confirmingDelete && (
              <>
                <DialogTitle className="text-base font-semibold">
                  {t("settings.tags.confirm.deleteTitle", {
                    name: confirmingDelete.name,
                  })}
                </DialogTitle>
                <Description className="text-sm text-zinc-600 dark:text-zinc-400">
                  {t("settings.tags.confirm.deleteBody")}
                </Description>
                <div className="flex justify-end gap-2">
                  <button
                    type="button"
                    onClick={() => setConfirmingDelete(null)}
                    className="rounded-md px-3 py-2 text-sm font-medium text-zinc-600 hover:bg-black/[.04] dark:text-zinc-400 dark:hover:bg-white/[.06]"
                  >
                    {t("settings.tags.cancel")}
                  </button>
                  <button
                    type="button"
                    onClick={onConfirmDelete}
                    className="rounded-md bg-zinc-900 px-3 py-2 text-sm font-medium text-white hover:bg-zinc-700 dark:bg-white dark:text-zinc-900 dark:hover:bg-zinc-200"
                  >
                    {t("settings.tags.confirm.confirmAction")}
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
