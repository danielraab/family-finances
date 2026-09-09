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
type ConfirmKind = "disable" | "enable" | "delete";

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
 * The /categories management page: the caller's own tree, editable
 * entirely through buttons (edit name/icon/colour, disable/enable, ▲/▼
 * reorder, a "Move to…" reparent picker, delete) — no drag gestures, so it
 * works the same
 * on a phone as on a desktop. See web-client-categories's spec and
 * design.md for why reordering/reparenting are dedicated actions rather
 * than drag-and-drop.
 */
function CategoriesPage() {
  const { status } = useAuth();
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
  const [editIcon, setEditIcon] = useState("");
  const [editColor, setEditColor] = useState("");
  const [saving, setSaving] = useState(false);
  const [editError, setEditError] = useState<string | null>(null);

  const [moving, setMoving] = useState<Category | null>(null);
  const [moveParentId, setMoveParentId] = useState("");
  const [movingSubmitting, setMovingSubmitting] = useState(false);
  const [moveError, setMoveError] = useState<string | null>(null);

  const [confirming, setConfirming] = useState<{
    kind: ConfirmKind;
    target: Category;
  } | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);

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

  const tree = buildCategoryTree(categories ?? []);
  const parentOptions = flattenCategoryTree(categories ?? []);

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
    setEditIcon(cat.icon ?? "");
    setEditColor(cat.color ?? "");
    setEditError(null);
  }

  async function onSaveEdit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!editing) return;
    setSaving(true);
    setEditError(null);
    const { data, response } = await api.PATCH("/api/categories/{id}", {
      params: { path: { id: editing.id } },
      body: { name: editName.trim(), icon: editIcon, color: editColor },
    });
    setSaving(false);
    if (!response.ok || !data) {
      setEditError(t("categories.edit.error"));
      return;
    }
    setCategories(
      (prev) => prev?.map((c) => (c.id === data.id ? data : c)) ?? null,
    );
    setEditing(null);
  }

  function openMove(cat: Category) {
    setMoving(cat);
    setMoveParentId(cat.parent_id ?? "");
    setMoveError(null);
  }

  async function onConfirmMove(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!moving) return;
    setMovingSubmitting(true);
    setMoveError(null);
    const { response } = await api.PATCH("/api/categories/{id}", {
      params: { path: { id: moving.id } },
      body: { parent_id: moveParentId || null },
    });
    setMovingSubmitting(false);
    if (!response.ok) {
      setMoveError(t("categories.move.error"));
      return;
    }
    setMoving(null);
    refresh();
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

  async function performConfirmed() {
    if (!confirming) return;
    const { kind, target } = confirming;
    setConfirming(null);
    setActionError(null);

    if (kind === "disable" || kind === "enable") {
      const { data } = await api.POST(
        kind === "disable"
          ? "/api/categories/{id}/disable"
          : "/api/categories/{id}/enable",
        { params: { path: { id: target.id } } },
      );
      if (data) {
        setCategories(
          (prev) => prev?.map((c) => (c.id === data.id ? data : c)) ?? null,
        );
      }
      return;
    }

    const { response } = await api.DELETE("/api/categories/{id}", {
      params: { path: { id: target.id } },
    });
    if (response.ok) {
      setCategories((prev) => prev?.filter((c) => c.id !== target.id) ?? null);
    } else {
      setActionError(t("categories.deleteError", { name: target.name }));
    }
  }

  const moveOptions = moving
    ? parentOptions.filter(
        (o) => !subtreeIds(categories ?? [], moving.id).has(o.id),
      )
    : [];

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
              onClick={() => openMove(node)}
              className="rounded-md px-2 py-1 text-xs font-medium text-zinc-600 underline underline-offset-2 hover:text-zinc-900 dark:text-zinc-400 dark:hover:text-zinc-100"
            >
              {t("categories.actions.moveTo")}
            </button>
            <button
              type="button"
              onClick={() => openEdit(node)}
              className="rounded-md px-2 py-1 text-xs font-medium text-zinc-600 underline underline-offset-2 hover:text-zinc-900 dark:text-zinc-400 dark:hover:text-zinc-100"
            >
              {t("categories.actions.edit")}
            </button>
            {node.disabled ? (
              <button
                type="button"
                onClick={() => setConfirming({ kind: "enable", target: node })}
                className="rounded-md px-2 py-1 text-xs font-medium text-zinc-600 underline underline-offset-2 hover:text-zinc-900 dark:text-zinc-400 dark:hover:text-zinc-100"
              >
                {t("categories.actions.enable")}
              </button>
            ) : (
              <button
                type="button"
                onClick={() => setConfirming({ kind: "disable", target: node })}
                className="rounded-md px-2 py-1 text-xs font-medium text-zinc-600 underline underline-offset-2 hover:text-zinc-900 dark:text-zinc-400 dark:hover:text-zinc-100"
              >
                {t("categories.actions.disable")}
              </button>
            )}
            <button
              type="button"
              disabled={node.children.length > 0}
              title={
                node.children.length > 0
                  ? t("categories.actions.deleteDisabledHint")
                  : undefined
              }
              onClick={() => setConfirming({ kind: "delete", target: node })}
              className="rounded-md px-2 py-1 text-xs font-medium text-red-600 underline underline-offset-2 hover:text-red-800 disabled:opacity-30 disabled:no-underline dark:text-red-400 dark:hover:text-red-300"
            >
              {t("categories.actions.delete")}
            </button>
          </div>
        </div>
        {node.children.map((child, i) =>
          renderNode(child, depth + 1, i === 0, i === node.children.length - 1),
        )}
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
      {actionError && (
        <p className="text-sm text-red-600 dark:text-red-400">{actionError}</p>
      )}
      {loadError && (
        <p className="text-sm text-red-600 dark:text-red-400">
          {t("categories.loadError")}
        </p>
      )}

      {categories !== null && categories.length === 0 ? (
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
        open={moving !== null}
        onClose={() => setMoving(null)}
        className="relative z-50"
      >
        <div className="fixed inset-0 bg-black/40" aria-hidden="true" />
        <div className="fixed inset-0 flex items-center justify-center p-4">
          <DialogPanel className="flex w-full max-w-sm flex-col gap-4 rounded-lg bg-white p-6 dark:bg-neutral-900">
            {moving && (
              <form onSubmit={onConfirmMove} className="flex flex-col gap-4">
                <DialogTitle className="text-base font-semibold">
                  {t("categories.move.heading", { name: moving.name })}
                </DialogTitle>
                <label className="flex flex-col gap-1.5 text-sm font-medium">
                  {t("categories.move.parentLabel")}
                  <select
                    value={moveParentId}
                    onChange={(e) => setMoveParentId(e.target.value)}
                    className={inputClass}
                  >
                    <option value="">{t("categories.move.root")}</option>
                    {moveOptions.map((o) => (
                      <option key={o.id} value={o.id}>
                        {o.label}
                      </option>
                    ))}
                  </select>
                </label>
                {moveError && (
                  <p className="text-sm text-red-600 dark:text-red-400">
                    {moveError}
                  </p>
                )}
                <div className="flex justify-end gap-2">
                  <button
                    type="button"
                    onClick={() => setMoving(null)}
                    className="rounded-md px-3 py-2 text-sm font-medium text-zinc-600 hover:bg-black/[.04] dark:text-zinc-400 dark:hover:bg-white/[.06]"
                  >
                    {t("categories.edit.cancel")}
                  </button>
                  <button
                    type="submit"
                    disabled={movingSubmitting}
                    className="rounded-md bg-zinc-900 px-3 py-2 text-sm font-medium text-white hover:bg-zinc-700 disabled:opacity-60 dark:bg-white dark:text-zinc-900 dark:hover:bg-zinc-200"
                  >
                    {movingSubmitting
                      ? t("categories.move.moving")
                      : t("categories.move.confirm")}
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
                  {t(`categories.confirm.${confirming.kind}Title`, {
                    name: confirming.target.name,
                  })}
                </DialogTitle>
                <Description className="text-sm text-zinc-600 dark:text-zinc-400">
                  {t(`categories.confirm.${confirming.kind}Body`)}
                </Description>
                <div className="flex justify-end gap-2">
                  <button
                    type="button"
                    onClick={() => setConfirming(null)}
                    className="rounded-md px-3 py-2 text-sm font-medium text-zinc-600 hover:bg-black/[.04] dark:text-zinc-400 dark:hover:bg-white/[.06]"
                  >
                    {t("categories.confirm.cancel")}
                  </button>
                  <button
                    type="button"
                    onClick={performConfirmed}
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
