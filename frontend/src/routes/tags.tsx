import {
  Description,
  Dialog,
  DialogPanel,
  DialogTitle,
} from "@headlessui/react";
import { createFileRoute, Link, useNavigate } from "@tanstack/react-router";
import { type FormEvent, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { api } from "../api/client";
import type { components } from "../api/schema";
import { useAuth } from "../components/AuthProvider";
import { TagLabel } from "../components/TagLabel";

export const Route = createFileRoute("/tags")({
  component: TagsPage,
});

type Tag = components["schemas"]["Tag"];

const inputClass =
  "rounded-md border border-black/15 bg-transparent px-3 py-2 text-sm font-normal outline-none transition-colors focus:border-black/40 dark:border-white/15 dark:focus:border-white/40";

/**
 * The /tags page: the caller's own tags (list/create/rename/disable/
 * enable/delete/share) plus a "Shared with me" section below, mirroring
 * /categories's own+shared split. See web-client-tags's spec and
 * design.md — tags stay a flat list (no tree, no icon/colour), and delete
 * is blocked (409) only while the tag currently has an active share, not
 * by being in use on the owner's own entries.
 */
function TagsPage() {
  const { status, user } = useAuth();
  const navigate = useNavigate();
  const { t } = useTranslation();

  useEffect(() => {
    if (status === "anonymous") {
      navigate({ to: "/login", replace: true });
    }
  }, [status, navigate]);

  const [tags, setTags] = useState<Tag[] | null>(null);
  const [loadError, setLoadError] = useState(false);

  const [createName, setCreateName] = useState("");
  const [creating, setCreating] = useState(false);
  const [createError, setCreateError] = useState<string | null>(null);

  const [editing, setEditing] = useState<Tag | null>(null);
  const [editName, setEditName] = useState("");
  const [saving, setSaving] = useState(false);
  const [editError, setEditError] = useState<string | null>(null);

  const [confirmingDelete, setConfirmingDelete] = useState<Tag | null>(null);
  const [confirmingLeave, setConfirmingLeave] = useState<Tag | null>(null);

  async function refresh() {
    const { data, response } = await api.GET("/api/tags");
    if (response.ok && data) {
      setTags(data);
      setLoadError(false);
    } else {
      setLoadError(true);
    }
  }

  // biome-ignore lint/correctness/useExhaustiveDependencies: refresh is a plain function, not memoized — only status should retrigger the initial fetch.
  useEffect(() => {
    if (status === "authenticated") {
      refresh();
    }
  }, [status]);

  if (status !== "authenticated") {
    return null;
  }

  const ownTags = (tags ?? []).filter((tg) => tg.permission === "owner");
  const sharedTags = (tags ?? []).filter((tg) => tg.shared);

  async function onCreate(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setCreating(true);
    setCreateError(null);
    const { data, response } = await api.POST("/api/tags", {
      body: { name: createName.trim() },
    });
    setCreating(false);
    if (!response.ok || !data) {
      setCreateError(t("tags.create.error"));
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
      setEditError(t("tags.edit.error"));
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
      setEditing((cur) => (cur && cur.id === data.id ? data : cur));
    }
  }

  async function onConfirmDelete() {
    if (!confirmingDelete) return;
    const target = confirmingDelete;
    setConfirmingDelete(null);
    setEditError(null);
    const { response } = await api.DELETE("/api/tags/{id}", {
      params: { path: { id: target.id } },
    });
    if (response.ok) {
      setTags((prev) => prev?.filter((tg) => tg.id !== target.id) ?? null);
      setEditing(null);
    } else if (response.status === 409) {
      setEditError(t("tags.deleteErrorShared", { name: target.name }));
    } else {
      setEditError(t("tags.deleteError", { name: target.name }));
    }
  }

  async function performLeave() {
    if (!confirmingLeave || !user) return;
    const target = confirmingLeave;
    setConfirmingLeave(null);
    const { response } = await api.DELETE("/api/tags/{id}/shares/{userId}", {
      params: { path: { id: target.id, userId: user.id } },
    });
    if (response.ok) {
      setTags((prev) => prev?.filter((tg) => tg.id !== target.id) ?? null);
    }
  }

  return (
    <section className="mx-auto flex w-full max-w-3xl flex-col gap-6 px-6 py-12 sm:px-10">
      <h1 className="text-2xl font-semibold tracking-tight">
        {t("tags.title")}
      </h1>

      <form
        onSubmit={onCreate}
        className="flex flex-col gap-2 sm:flex-row sm:items-end"
      >
        <label className="flex flex-1 flex-col gap-1.5 text-sm font-medium">
          {t("tags.create.nameLabel")}
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
          {creating ? t("tags.create.submitting") : t("tags.create.submit")}
        </button>
      </form>
      {createError && (
        <p className="text-sm text-red-600 dark:text-red-400">{createError}</p>
      )}
      {loadError && (
        <p className="text-sm text-red-600 dark:text-red-400">
          {t("tags.loadError")}
        </p>
      )}

      {tags !== null && ownTags.length === 0 ? (
        <p className="text-sm text-zinc-500 dark:text-zinc-400">
          {t("tags.empty")}
        </p>
      ) : (
        <div className="overflow-x-auto rounded-lg border border-black/10 dark:border-white/10">
          <table className="w-full text-left text-sm">
            <thead className="border-b border-black/10 text-xs uppercase text-zinc-500 dark:border-white/10 dark:text-zinc-400">
              <tr>
                <th className="px-3 py-2 font-medium">
                  {t("tags.columnName")}
                </th>
                <th className="px-3 py-2 font-medium">
                  {t("tags.columnEntries")}
                </th>
                <th className="px-3 py-2 font-medium">
                  {t("tags.columnStatus")}
                </th>
                <th className="px-3 py-2" />
              </tr>
            </thead>
            <tbody>
              {ownTags.map((tg) => (
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
                      ? t("tags.status.disabled")
                      : t("tags.status.active")}
                  </td>
                  <td className="px-3 py-2 text-right">
                    <div className="flex flex-wrap justify-end gap-2">
                      <button
                        type="button"
                        onClick={() => openEdit(tg)}
                        className="text-xs font-medium text-zinc-600 underline underline-offset-2 hover:text-zinc-900 dark:text-zinc-400 dark:hover:text-zinc-100"
                      >
                        {t("tags.actions.edit")}
                      </button>
                      <Link
                        to="/tags/$tagId/sharing"
                        params={{ tagId: tg.id }}
                        className="text-xs font-medium text-zinc-600 underline underline-offset-2 hover:text-zinc-900 dark:text-zinc-400 dark:hover:text-zinc-100"
                      >
                        {t("tags.actions.share")}
                      </Link>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {sharedTags.length > 0 && (
        <div className="flex flex-col gap-3 border-t border-black/10 pt-6 dark:border-white/10">
          <h2 className="text-sm font-semibold text-zinc-500 dark:text-zinc-400">
            {t("tags.sharedWithMe.heading")}
          </h2>
          <div className="flex flex-col gap-2">
            {sharedTags.map((tg) => (
              <div
                key={tg.id}
                className="flex flex-col gap-2 rounded-lg border border-black/10 bg-white p-3 sm:flex-row sm:items-center sm:justify-between dark:border-white/10 dark:bg-black"
              >
                <div className="flex items-center gap-2 text-sm">
                  <TagLabel
                    tag={tg}
                    className="font-medium text-zinc-900 dark:text-zinc-100"
                  />
                  <span className="rounded-full bg-zinc-100 px-2 py-0.5 text-xs font-medium text-zinc-500 dark:bg-white/10 dark:text-zinc-400">
                    {t(`tags.sharing.permission.${tg.permission}`)}
                  </span>
                </div>
                <div className="flex flex-wrap items-center gap-1">
                  <Link
                    to="/tags/$tagId/sharing"
                    params={{ tagId: tg.id }}
                    className="rounded-md px-2 py-1 text-xs font-medium text-zinc-600 underline underline-offset-2 hover:text-zinc-900 dark:text-zinc-400 dark:hover:text-zinc-100"
                  >
                    {t("tags.actions.viewSharing")}
                  </Link>
                  <button
                    type="button"
                    onClick={() => setConfirmingLeave(tg)}
                    className="rounded-md border border-red-200 px-3 py-1.5 text-xs font-medium text-red-600 transition-colors hover:bg-red-50 dark:border-red-900/50 dark:text-red-400 dark:hover:bg-red-950/30"
                  >
                    {t("tags.actions.leave")}
                  </button>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      <Dialog
        open={confirmingLeave !== null}
        onClose={() => setConfirmingLeave(null)}
        className="relative z-50"
      >
        <div className="fixed inset-0 bg-black/40" aria-hidden="true" />
        <div className="fixed inset-0 flex items-center justify-center p-4">
          <DialogPanel className="flex w-full max-w-sm flex-col gap-4 rounded-lg bg-white p-6 dark:bg-neutral-900">
            {confirmingLeave && (
              <>
                <DialogTitle className="text-base font-semibold">
                  {t("tags.sharedWithMe.confirmLeaveTitle", {
                    name: confirmingLeave.name,
                  })}
                </DialogTitle>
                <Description className="text-sm text-zinc-600 dark:text-zinc-400">
                  {t("tags.sharedWithMe.confirmLeaveBody")}
                </Description>
                <div className="flex justify-end gap-2">
                  <button
                    type="button"
                    onClick={() => setConfirmingLeave(null)}
                    className="rounded-md px-3 py-2 text-sm font-medium text-zinc-600 hover:bg-black/[.04] dark:text-zinc-400 dark:hover:bg-white/[.06]"
                  >
                    {t("tags.confirm.cancel")}
                  </button>
                  <button
                    type="button"
                    onClick={performLeave}
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
                  {t("tags.edit.heading")}
                </DialogTitle>
                <label className="flex flex-col gap-1.5 text-sm font-medium">
                  {t("tags.edit.nameLabel")}
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
                <div className="flex flex-wrap items-center gap-2 border-t border-black/10 pt-3 dark:border-white/10">
                  <button
                    type="button"
                    onClick={() => onToggleDisabled(editing)}
                    className="rounded-md px-2 py-1 text-xs font-medium text-zinc-600 underline underline-offset-2 hover:text-zinc-900 dark:text-zinc-400 dark:hover:text-zinc-100"
                  >
                    {editing.disabled
                      ? t("tags.actions.enable")
                      : t("tags.actions.disable")}
                  </button>
                  <button
                    type="button"
                    onClick={() => setConfirmingDelete(editing)}
                    className="rounded-md px-2 py-1 text-xs font-medium text-red-600 underline underline-offset-2 hover:text-red-800 dark:text-red-400 dark:hover:text-red-300"
                  >
                    {t("tags.actions.delete")}
                  </button>
                </div>
                <div className="flex justify-end gap-2">
                  <button
                    type="button"
                    onClick={() => setEditing(null)}
                    className="rounded-md px-3 py-2 text-sm font-medium text-zinc-600 hover:bg-black/[.04] dark:text-zinc-400 dark:hover:bg-white/[.06]"
                  >
                    {t("tags.edit.cancel")}
                  </button>
                  <button
                    type="submit"
                    disabled={saving}
                    className="rounded-md bg-zinc-900 px-3 py-2 text-sm font-medium text-white hover:bg-zinc-700 disabled:opacity-60 dark:bg-white dark:text-zinc-900 dark:hover:bg-zinc-200"
                  >
                    {saving ? t("tags.edit.saving") : t("tags.edit.save")}
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
                  {t("tags.confirm.deleteTitle", {
                    name: confirmingDelete.name,
                  })}
                </DialogTitle>
                <Description className="text-sm text-zinc-600 dark:text-zinc-400">
                  {t("tags.confirm.deleteBody")}
                </Description>
                <div className="flex justify-end gap-2">
                  <button
                    type="button"
                    onClick={() => setConfirmingDelete(null)}
                    className="rounded-md px-3 py-2 text-sm font-medium text-zinc-600 hover:bg-black/[.04] dark:text-zinc-400 dark:hover:bg-white/[.06]"
                  >
                    {t("tags.confirm.cancel")}
                  </button>
                  <button
                    type="button"
                    onClick={onConfirmDelete}
                    className="rounded-md bg-zinc-900 px-3 py-2 text-sm font-medium text-white hover:bg-zinc-700 dark:bg-white dark:text-zinc-900 dark:hover:bg-zinc-200"
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
