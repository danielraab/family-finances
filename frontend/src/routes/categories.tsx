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
import { CategoryLabel } from "../components/CategoryLabel";
import { IconColorPicker } from "../components/IconColorPicker";
import {
  buildCategoryTree,
  type CategoryNode,
  flattenCategoryTree,
} from "../lib/categoryTree";
import { compact } from "../lib/compact";

export const Route = createFileRoute("/categories")({
  component: CategoriesPage,
});

type Category = components["schemas"]["Category"];

const inputClass =
  "rounded-md border border-black/15 bg-transparent px-3 py-2 text-sm font-normal outline-none transition-colors focus:border-black/40 dark:border-white/15 dark:focus:border-white/40";

/** id and every descendant id of id within categories, root-inclusive. */
function subtreeIds(categories: Category[], id: string): Set<string> {
  const children = new Map<string, string[]>();
  for (const c of categories) {
    if (c.parent_id) {
      children.set(c.parent_id, [...(children.get(c.parent_id) ?? []), c.id]);
    }
  }
  const out = new Set<string>([id]);
  const queue = [id];
  while (queue.length > 0) {
    const cur = queue.shift();
    if (cur === undefined) break;
    for (const child of children.get(cur) ?? []) {
      out.add(child);
      queue.push(child);
    }
  }
  return out;
}

/**
 * The /categories management page: the caller's own tree. The inline node
 * row carries only ▲/▼ sibling reorder and Edit; everything else scoped to
 * a single category — rename, icon/colour, reparent (a parent picker
 * applied on Save), disable/enable (immediate), and delete (confirmed) —
 * lives in the edit dialog. No drag gestures, so it works the same on a
 * phone as on a desktop. See web-client-categories's spec and design.md.
 */
function CategoriesPage() {
  const { status, user } = useAuth();
  const navigate = useNavigate();
  const { t } = useTranslation();

  useEffect(() => {
    if (status === "anonymous") {
      navigate({ to: "/login", replace: true });
    }
  }, [status, navigate]);

  const [categories, setCategories] = useState<Category[] | null>(null);
  const [loadError, setLoadError] = useState(false);

  const [createName, setCreateName] = useState("");
  const [createParentId, setCreateParentId] = useState("");
  const [createIcon, setCreateIcon] = useState("");
  const [createColor, setCreateColor] = useState("");
  const [creating, setCreating] = useState(false);
  const [createError, setCreateError] = useState<string | null>(null);

  const [editing, setEditing] = useState<Category | null>(null);
  const [editName, setEditName] = useState("");
  const [editParentId, setEditParentId] = useState("");
  const [editIcon, setEditIcon] = useState("");
  const [editColor, setEditColor] = useState("");
  const [saving, setSaving] = useState(false);
  const [editError, setEditError] = useState<string | null>(null);

  const [confirmingDelete, setConfirmingDelete] = useState<Category | null>(
    null,
  );
  const [confirmingLeave, setConfirmingLeave] = useState<Category | null>(null);

  async function refresh() {
    const { data, response } = await api.GET("/api/categories");
    if (response.ok && data) {
      setCategories(data);
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

  // The caller's own tree (nested, full controls) is built only from
  // categories they really own — a category shared with them, even at
  // append, never nests into or offers as a parent within it. It renders
  // separately, flat, in its own "shared with me" section below.
  const ownCategories = (categories ?? []).filter(
    (c) => c.permission === "owner",
  );
  const sharedCategories = (categories ?? []).filter((c) => c.shared);
  const tree = buildCategoryTree(ownCategories);
  const parentOptions = flattenCategoryTree(ownCategories);
  const editParentOptions = editing
    ? parentOptions.filter(
        (o) => !subtreeIds(ownCategories, editing.id).has(o.id),
      )
    : [];
  const editingHasChildren =
    editing !== null && ownCategories.some((c) => c.parent_id === editing.id);

  async function onCreate(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setCreating(true);
    setCreateError(null);
    const { data, response } = await api.POST("/api/categories", {
      body: {
        name: createName.trim(),
        icon: createIcon,
        color: createColor,
        ...compact({ parent_id: createParentId || undefined }),
      },
    });
    setCreating(false);
    if (!response.ok || !data) {
      setCreateError(t("categories.create.error"));
      return;
    }
    setCategories((prev) => [...(prev ?? []), data]);
    setCreateName("");
    setCreateParentId("");
    setCreateIcon("");
    setCreateColor("");
  }

  function openEdit(cat: Category) {
    setEditing(cat);
    setEditName(cat.name);
    setEditParentId(cat.parent_id ?? "");
    setEditIcon(cat.icon ?? "");
    setEditColor(cat.color ?? "");
    setEditError(null);
  }

  async function onSaveEdit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!editing) return;
    setSaving(true);
    setEditError(null);
    const { response } = await api.PATCH("/api/categories/{id}", {
      params: { path: { id: editing.id } },
      body: {
        name: editName.trim(),
        icon: editIcon,
        color: editColor,
        parent_id: editParentId || null,
      },
    });
    setSaving(false);
    if (!response.ok) {
      setEditError(t("categories.edit.error"));
      return;
    }
    // A reparent shifts sibling order under the new parent — re-fetch the
    // whole tree rather than patch it locally.
    await refresh();
    setEditing(null);
  }

  async function toggleDisabled(cat: Category) {
    const { data } = await api.POST(
      cat.disabled
        ? "/api/categories/{id}/enable"
        : "/api/categories/{id}/disable",
      { params: { path: { id: cat.id } } },
    );
    if (data) {
      setCategories(
        (prev) => prev?.map((c) => (c.id === data.id ? data : c)) ?? null,
      );
      setEditing((cur) => (cur && cur.id === data.id ? data : cur));
    }
  }

  async function moveUp(cat: Category) {
    await api.POST("/api/categories/{id}/move-up", {
      params: { path: { id: cat.id } },
    });
    refresh();
  }

  async function moveDown(cat: Category) {
    await api.POST("/api/categories/{id}/move-down", {
      params: { path: { id: cat.id } },
    });
    refresh();
  }

  async function performDelete() {
    if (!confirmingDelete) return;
    const target = confirmingDelete;
    setConfirmingDelete(null);
    setEditError(null);
    const { response } = await api.DELETE("/api/categories/{id}", {
      params: { path: { id: target.id } },
    });
    if (response.ok) {
      setCategories((prev) => prev?.filter((c) => c.id !== target.id) ?? null);
      setEditing(null);
    } else {
      setEditError(t("categories.deleteError", { name: target.name }));
    }
  }

  async function performLeave() {
    if (!confirmingLeave || !user) return;
    const target = confirmingLeave;
    setConfirmingLeave(null);
    const { response } = await api.DELETE(
      "/api/categories/{id}/shares/{userId}",
      { params: { path: { id: target.id, userId: user.id } } },
    );
    if (response.ok) {
      setCategories((prev) => prev?.filter((c) => c.id !== target.id) ?? null);
    }
  }

  function renderNode(
    node: CategoryNode,
    depth: number,
    isFirst: boolean,
    isLast: boolean,
  ) {
    return (
      <div key={node.id}>
        <div
          className="flex flex-col gap-2 rounded-lg border border-black/10 bg-white p-3 sm:flex-row sm:items-center sm:justify-between dark:border-white/10 dark:bg-black"
          style={{ marginLeft: Math.min(depth, 6) * 20 }}
        >
          <div className="flex items-center gap-2 text-sm">
            <CategoryLabel
              category={node}
              className="font-medium text-zinc-900 dark:text-zinc-100"
            />
            <span
              className={`rounded-full px-2 py-0.5 text-xs font-medium ${
                node.disabled
                  ? "bg-zinc-100 text-zinc-500 dark:bg-white/10 dark:text-zinc-400"
                  : "bg-emerald-100 text-emerald-700 dark:bg-emerald-500/10 dark:text-emerald-400"
              }`}
            >
              {node.disabled
                ? t("categories.status.disabled")
                : t("categories.status.active")}
            </span>
            <span className="rounded-full bg-zinc-100 px-2 py-0.5 text-xs font-medium text-zinc-500 dark:bg-white/10 dark:text-zinc-400">
              {t("categories.entryCount", { count: node.entry_count })}
            </span>
          </div>
          <div className="flex flex-wrap items-center gap-1">
            <button
              type="button"
              disabled={isFirst}
              onClick={() => moveUp(node)}
              aria-label={t("categories.actions.moveUp")}
              className="rounded-md px-2 py-1 text-xs font-medium text-zinc-600 hover:bg-black/[.04] disabled:opacity-30 dark:text-zinc-400 dark:hover:bg-white/[.06]"
            >
              ▲
            </button>
            <button
              type="button"
              disabled={isLast}
              onClick={() => moveDown(node)}
              aria-label={t("categories.actions.moveDown")}
              className="rounded-md px-2 py-1 text-xs font-medium text-zinc-600 hover:bg-black/[.04] disabled:opacity-30 dark:text-zinc-400 dark:hover:bg-white/[.06]"
            >
              ▼
            </button>
            <button
              type="button"
              onClick={() => openEdit(node)}
              className="rounded-md px-2 py-1 text-xs font-medium text-zinc-600 underline underline-offset-2 hover:text-zinc-900 dark:text-zinc-400 dark:hover:text-zinc-100"
            >
              {t("categories.actions.edit")}
            </button>
            <Link
              to="/categories/$categoryId/sharing"
              params={{ categoryId: node.id }}
              className="rounded-md px-2 py-1 text-xs font-medium text-zinc-600 underline underline-offset-2 hover:text-zinc-900 dark:text-zinc-400 dark:hover:text-zinc-100"
            >
              {t("categories.actions.share")}
            </Link>
          </div>
        </div>
        {node.children.map((child, i) =>
          renderNode(child, depth + 1, i === 0, i === node.children.length - 1),
        )}
      </div>
    );
  }

  function renderSharedNode(node: Category) {
    return (
      <div
        key={node.id}
        className="flex flex-col gap-2 rounded-lg border border-black/10 bg-white p-3 sm:flex-row sm:items-center sm:justify-between dark:border-white/10 dark:bg-black"
      >
        <div className="flex items-center gap-2 text-sm">
          <CategoryLabel
            category={node}
            className="font-medium text-zinc-900 dark:text-zinc-100"
          />
          <span className="rounded-full bg-zinc-100 px-2 py-0.5 text-xs font-medium text-zinc-500 dark:bg-white/10 dark:text-zinc-400">
            {t(`categories.sharing.permission.${node.permission}`)}
          </span>
        </div>
        <div className="flex flex-wrap items-center gap-1">
          <Link
            to="/categories/$categoryId/sharing"
            params={{ categoryId: node.id }}
            className="rounded-md px-2 py-1 text-xs font-medium text-zinc-600 underline underline-offset-2 hover:text-zinc-900 dark:text-zinc-400 dark:hover:text-zinc-100"
          >
            {t("categories.actions.viewSharing")}
          </Link>
          <button
            type="button"
            onClick={() => setConfirmingLeave(node)}
            className="rounded-md border border-red-200 px-3 py-1.5 text-xs font-medium text-red-600 transition-colors hover:bg-red-50 dark:border-red-900/50 dark:text-red-400 dark:hover:bg-red-950/30"
          >
            {t("categories.actions.leave")}
          </button>
        </div>
      </div>
    );
  }

  return (
    <section className="mx-auto flex w-full max-w-3xl flex-col gap-6 px-6 py-12 sm:px-10">
      <h1 className="text-2xl font-semibold tracking-tight">
        {t("categories.title")}
      </h1>

      <form
        onSubmit={onCreate}
        className="flex flex-col gap-3 rounded-lg border border-black/10 p-4 dark:border-white/10"
      >
        <div className="flex flex-col gap-2 sm:flex-row sm:items-end">
          <label className="flex flex-1 flex-col gap-1.5 text-sm font-medium">
            {t("categories.create.nameLabel")}
            <input
              required
              value={createName}
              onChange={(e) => setCreateName(e.target.value)}
              className={inputClass}
            />
          </label>
          <label className="flex flex-1 flex-col gap-1.5 text-sm font-medium">
            {t("categories.create.parentLabel")}
            <select
              value={createParentId}
              onChange={(e) => setCreateParentId(e.target.value)}
              className={inputClass}
            >
              <option value="">{t("categories.create.parentRoot")}</option>
              {parentOptions.map((o) => (
                <option key={o.id} value={o.id}>
                  {o.label}
                </option>
              ))}
            </select>
          </label>
          <button
            type="submit"
            disabled={creating}
            className="rounded-md bg-zinc-900 px-3 py-2 text-sm font-medium text-white transition-colors hover:bg-zinc-700 disabled:opacity-60 dark:bg-white dark:text-zinc-900 dark:hover:bg-zinc-200"
          >
            {creating
              ? t("categories.create.submitting")
              : t("categories.create.submit")}
          </button>
        </div>
        <IconColorPicker
          value={{ icon: createIcon, color: createColor }}
          onChange={(next) => {
            setCreateIcon(next.icon);
            setCreateColor(next.color);
          }}
        />
      </form>
      {createError && (
        <p className="text-sm text-red-600 dark:text-red-400">{createError}</p>
      )}
      {loadError && (
        <p className="text-sm text-red-600 dark:text-red-400">
          {t("categories.loadError")}
        </p>
      )}

      {categories !== null && ownCategories.length === 0 ? (
        <p className="text-sm text-zinc-500 dark:text-zinc-400">
          {t("categories.empty")}
        </p>
      ) : (
        <div className="flex flex-col gap-2">
          {tree.map((node, i) =>
            renderNode(node, 0, i === 0, i === tree.length - 1),
          )}
        </div>
      )}

      {sharedCategories.length > 0 && (
        <div className="flex flex-col gap-3 border-t border-black/10 pt-6 dark:border-white/10">
          <h2 className="text-sm font-semibold text-zinc-500 dark:text-zinc-400">
            {t("categories.sharedWithMe.heading")}
          </h2>
          <div className="flex flex-col gap-2">
            {sharedCategories.map((c) => renderSharedNode(c))}
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
                  {t("categories.sharedWithMe.confirmLeaveTitle", {
                    name: confirmingLeave.name,
                  })}
                </DialogTitle>
                <Description className="text-sm text-zinc-600 dark:text-zinc-400">
                  {t("categories.sharedWithMe.confirmLeaveBody")}
                </Description>
                <div className="flex justify-end gap-2">
                  <button
                    type="button"
                    onClick={() => setConfirmingLeave(null)}
                    className="rounded-md px-3 py-2 text-sm font-medium text-zinc-600 hover:bg-black/[.04] dark:text-zinc-400 dark:hover:bg-white/[.06]"
                  >
                    {t("categories.confirm.cancel")}
                  </button>
                  <button
                    type="button"
                    onClick={performLeave}
                    className="rounded-md bg-red-600 px-3 py-2 text-sm font-medium text-white hover:bg-red-700"
                  >
                    {t("categories.confirm.confirmAction")}
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
                  {t("categories.edit.heading")}
                </DialogTitle>
                <label className="flex flex-col gap-1.5 text-sm font-medium">
                  {t("categories.edit.nameLabel")}
                  <input
                    required
                    value={editName}
                    onChange={(e) => setEditName(e.target.value)}
                    className={inputClass}
                  />
                </label>
                <label className="flex flex-col gap-1.5 text-sm font-medium">
                  {t("categories.edit.parentLabel")}
                  <select
                    value={editParentId}
                    onChange={(e) => setEditParentId(e.target.value)}
                    className={inputClass}
                  >
                    <option value="">{t("categories.edit.parentRoot")}</option>
                    {editParentOptions.map((o) => (
                      <option key={o.id} value={o.id}>
                        {o.label}
                      </option>
                    ))}
                  </select>
                </label>
                <IconColorPicker
                  value={{ icon: editIcon, color: editColor }}
                  onChange={(next) => {
                    setEditIcon(next.icon);
                    setEditColor(next.color);
                  }}
                />
                {editError && (
                  <p className="text-sm text-red-600 dark:text-red-400">
                    {editError}
                  </p>
                )}
                <div className="flex flex-wrap items-center gap-2 border-t border-black/10 pt-3 dark:border-white/10">
                  <button
                    type="button"
                    onClick={() => toggleDisabled(editing)}
                    className="rounded-md px-2 py-1 text-xs font-medium text-zinc-600 underline underline-offset-2 hover:text-zinc-900 dark:text-zinc-400 dark:hover:text-zinc-100"
                  >
                    {editing.disabled
                      ? t("categories.actions.enable")
                      : t("categories.actions.disable")}
                  </button>
                  <button
                    type="button"
                    disabled={editingHasChildren}
                    title={
                      editingHasChildren
                        ? t("categories.actions.deleteDisabledHint")
                        : undefined
                    }
                    onClick={() => setConfirmingDelete(editing)}
                    className="rounded-md px-2 py-1 text-xs font-medium text-red-600 underline underline-offset-2 hover:text-red-800 disabled:opacity-30 disabled:no-underline dark:text-red-400 dark:hover:text-red-300"
                  >
                    {t("categories.actions.delete")}
                  </button>
                </div>
                <div className="flex justify-end gap-2">
                  <button
                    type="button"
                    onClick={() => setEditing(null)}
                    className="rounded-md px-3 py-2 text-sm font-medium text-zinc-600 hover:bg-black/[.04] dark:text-zinc-400 dark:hover:bg-white/[.06]"
                  >
                    {t("categories.edit.cancel")}
                  </button>
                  <button
                    type="submit"
                    disabled={saving}
                    className="rounded-md bg-zinc-900 px-3 py-2 text-sm font-medium text-white hover:bg-zinc-700 disabled:opacity-60 dark:bg-white dark:text-zinc-900 dark:hover:bg-zinc-200"
                  >
                    {saving
                      ? t("categories.edit.saving")
                      : t("categories.edit.save")}
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
                  {t("categories.confirm.deleteTitle", {
                    name: confirmingDelete.name,
                  })}
                </DialogTitle>
                <Description className="text-sm text-zinc-600 dark:text-zinc-400">
                  {t("categories.confirm.deleteBody")}
                </Description>
                <div className="flex justify-end gap-2">
                  <button
                    type="button"
                    onClick={() => setConfirmingDelete(null)}
                    className="rounded-md px-3 py-2 text-sm font-medium text-zinc-600 hover:bg-black/[.04] dark:text-zinc-400 dark:hover:bg-white/[.06]"
                  >
                    {t("categories.confirm.cancel")}
                  </button>
                  <button
                    type="button"
                    onClick={performDelete}
                    className="rounded-md bg-zinc-900 px-3 py-2 text-sm font-medium text-white hover:bg-zinc-700 dark:bg-white dark:text-zinc-900 dark:hover:bg-zinc-200"
                  >
                    {t("categories.confirm.confirmAction")}
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
