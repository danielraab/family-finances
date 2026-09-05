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
import { TagInput } from "../components/TagInput";
import { amountToInput, inputToAmount } from "../lib/amount";
import { flattenCategoryTree } from "../lib/categoryTree";
import { compact } from "../lib/compact";

type Account = components["schemas"]["Account"];
type Category = components["schemas"]["Category"];
type Tag = components["schemas"]["Tag"];
type Entry = components["schemas"]["Entry"];

export const Route = createFileRoute("/entries/$entryId/edit")({
  component: EditEntry,
});

function toLocalInput(iso: string): string {
  const d = new Date(iso);
  const tzOffsetMs = d.getTimezoneOffset() * 60_000;
  return new Date(d.getTime() - tzOffsetMs).toISOString().slice(0, 16);
}

const inputClass =
  "rounded-md border border-black/15 bg-transparent px-3 py-2 text-sm font-normal outline-none transition-colors focus:border-black/40 dark:border-white/15 dark:focus:border-white/40";

function EditEntry() {
  const { entryId } = Route.useParams();
  const { t } = useTranslation();
  const navigate = useNavigate();

  const [entry, setEntry] = useState<Entry | null | undefined>(undefined);
  const [account, setAccount] = useState<Account | null>(null);
  const [categories, setCategories] = useState<Category[]>([]);
  const [tags, setTags] = useState<Tag[]>([]);

  const [amount, setAmount] = useState("");
  const [bookingTimestamp, setBookingTimestamp] = useState("");
  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [categoryId, setCategoryId] = useState("");
  const [tagNames, setTagNames] = useState<string[]>([]);

  const [invalidField, setInvalidField] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [confirmingDelete, setConfirmingDelete] = useState(false);

  useEffect(() => {
    let cancelled = false;
    Promise.all([
      api.GET("/api/entries/{id}", { params: { path: { id: entryId } } }),
      api.GET("/api/categories"),
      api.GET("/api/tags"),
    ]).then(([entryRes, categoriesRes, tagsRes]) => {
      if (cancelled) return;
      setCategories(categoriesRes.data ?? []);
      const allTags = tagsRes.data ?? [];
      setTags(allTags);
      const e = entryRes.data ?? null;
      setEntry(e);
      if (e) {
        setAmount(amountToInput(e.amount));
        setBookingTimestamp(toLocalInput(e.booking_timestamp));
        setTitle(e.title);
        setDescription(e.description ?? "");
        setCategoryId(e.category_id ?? "");
        setTagNames(
          e.tag_ids
            .map((id) => allTags.find((tag) => tag.id === id)?.name)
            .filter((name): name is string => Boolean(name)),
        );
        api
          .GET("/api/accounts/{id}", { params: { path: { id: e.account_id } } })
          .then(({ data }) => {
            if (!cancelled) setAccount(data ?? null);
          });
      }
    });
    return () => {
      cancelled = true;
    };
  }, [entryId]);

  async function resolveTagIds(): Promise<string[]> {
    const ids: string[] = [];
    for (const name of tagNames) {
      const existing = tags.find((tag) => tag.name === name);
      if (existing) {
        ids.push(existing.id);
        continue;
      }
      const { data } = await api.POST("/api/tags", { body: { name } });
      if (data) ids.push(data.id);
    }
    return ids;
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!entry) return;
    if (title.trim() === "") {
      setInvalidField("title");
      return;
    }
    const parsedAmount = inputToAmount(amount);
    if (parsedAmount === null) {
      setInvalidField("amount");
      return;
    }
    if (entry.kind === "transaction" && !categoryId) {
      setInvalidField("category_id");
      return;
    }
    setInvalidField(null);
    setSubmitting(true);
    setError(null);

    const tagIds = await resolveTagIds();
    const { data, response } = await api.PATCH("/api/entries/{id}", {
      params: { path: { id: entryId } },
      body: {
        amount: parsedAmount,
        booking_timestamp: new Date(bookingTimestamp).toISOString(),
        title: title.trim(),
        category_id: categoryId || null,
        tag_ids: tagIds,
        ...compact({ description: description.trim() || undefined }),
      },
    });
    setSubmitting(false);
    if (!response.ok || !data) {
      setError(t("entries.form.saveError"));
      return;
    }
    navigate({
      to: "/entries",
      search: { account_id: entry.account_id },
    });
  }

  async function handleDelete() {
    setConfirmingDelete(false);
    const { response } = await api.DELETE("/api/entries/{id}", {
      params: { path: { id: entryId } },
    });
    if (response.ok && entry) {
      navigate({ to: "/entries", search: { account_id: entry.account_id } });
    }
  }

  if (entry === undefined) {
    return null;
  }
  if (entry === null) {
    return (
      <section className="mx-auto flex w-full max-w-xl flex-col gap-4 px-6 py-12 sm:px-10">
        <p className="text-sm text-zinc-500 dark:text-zinc-400">
          {t("entries.notFound")}
        </p>
      </section>
    );
  }

  const categoryOptions = flattenCategoryTree(categories);

  return (
    <section className="mx-auto flex w-full max-w-xl flex-col gap-8 px-6 py-12 sm:px-10">
      <h1 className="text-2xl font-semibold tracking-tight">
        {t("entries.edit.title")}
      </h1>

      <form onSubmit={handleSubmit} className="flex flex-col gap-4">
        <div className="flex flex-col gap-1.5 text-sm font-medium">
          {t("entries.form.account")}
          <input
            value={account?.title ?? entry.account_id}
            disabled
            className={`${inputClass} opacity-60`}
          />
        </div>

        <div className="flex flex-col gap-1.5 text-sm font-medium">
          {t("entries.form.kind")}
          <input
            value={
              entry.kind === "transaction"
                ? t("entries.kind.transaction")
                : t("entries.kind.balanceAdjustment")
            }
            disabled
            className={`${inputClass} opacity-60`}
          />
        </div>

        <label className="flex flex-col gap-1.5 text-sm font-medium">
          {t("entries.form.amount", { currency: account?.currency ?? "" })}
          <input
            value={amount}
            onChange={(e) => setAmount(e.target.value)}
            inputMode="decimal"
            className={inputClass}
            required
          />
          {invalidField === "amount" && (
            <span className="text-xs font-normal text-red-600 dark:text-red-400">
              {t("entries.form.amountInvalid")}
            </span>
          )}
        </label>

        <label className="flex flex-col gap-1.5 text-sm font-medium">
          {t("entries.form.bookingTimestamp")}
          <input
            type="datetime-local"
            value={bookingTimestamp}
            onChange={(e) => setBookingTimestamp(e.target.value)}
            className={inputClass}
            required
          />
        </label>

        <label className="flex flex-col gap-1.5 text-sm font-medium">
          {t("entries.form.title")}
          <input
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            className={inputClass}
            required
          />
          {invalidField === "title" && (
            <span className="text-xs font-normal text-red-600 dark:text-red-400">
              {t("entries.form.titleRequired")}
            </span>
          )}
        </label>

        <label className="flex flex-col gap-1.5 text-sm font-medium">
          {t("entries.form.description")}
          <textarea
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            className={`${inputClass} min-h-16`}
          />
        </label>

        <label className="flex flex-col gap-1.5 text-sm font-medium">
          {t("entries.form.category")}
          <select
            value={categoryId}
            onChange={(e) => setCategoryId(e.target.value)}
            className={inputClass}
          >
            <option value="">
              {entry.kind === "balance_adjustment"
                ? t("entries.form.categoryNone")
                : t("entries.form.categoryPlaceholder")}
            </option>
            {categoryOptions.map((c) => (
              <option key={c.id} value={c.id}>
                {c.label}
              </option>
            ))}
          </select>
          {invalidField === "category_id" && (
            <span className="text-xs font-normal text-red-600 dark:text-red-400">
              {t("entries.form.categoryRequired")}
            </span>
          )}
        </label>

        <div className="flex flex-col gap-1.5 text-sm font-medium">
          {t("entries.form.tags")}
          <TagInput
            value={tagNames}
            onChange={setTagNames}
            existingTags={tags}
          />
        </div>

        {error && (
          <p className="text-sm text-red-600 dark:text-red-400">{error}</p>
        )}

        <div className="flex items-center justify-between">
          <button
            type="submit"
            disabled={submitting}
            className="rounded-md bg-zinc-900 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-zinc-700 disabled:opacity-60 dark:bg-white dark:text-zinc-900 dark:hover:bg-zinc-200"
          >
            {t("entries.form.save")}
          </button>
          <button
            type="button"
            onClick={() => setConfirmingDelete(true)}
            className="rounded-md border border-red-200 px-3 py-2 text-sm font-medium text-red-600 transition-colors hover:bg-red-50 dark:border-red-900/50 dark:text-red-400 dark:hover:bg-red-950/30"
          >
            {t("entries.edit.delete")}
          </button>
        </div>
      </form>

      <Dialog
        open={confirmingDelete}
        onClose={() => setConfirmingDelete(false)}
        className="relative z-50"
      >
        <div className="fixed inset-0 bg-black/40" aria-hidden="true" />
        <div className="fixed inset-0 flex items-center justify-center p-4">
          <DialogPanel className="flex w-full max-w-sm flex-col gap-4 rounded-lg bg-white p-6 dark:bg-neutral-900">
            <DialogTitle className="text-base font-semibold">
              {t("entries.edit.confirmDeleteTitle")}
            </DialogTitle>
            <Description className="text-sm text-zinc-600 dark:text-zinc-400">
              {t("entries.edit.confirmDeleteBody")}
            </Description>
            <div className="flex justify-end gap-2">
              <button
                type="button"
                onClick={() => setConfirmingDelete(false)}
                className="rounded-md px-3 py-2 text-sm font-medium text-zinc-600 hover:bg-black/[.04] dark:text-zinc-400 dark:hover:bg-white/[.06]"
              >
                {t("accounts.edit.confirm.cancel")}
              </button>
              <button
                type="button"
                onClick={handleDelete}
                className="rounded-md bg-red-600 px-3 py-2 text-sm font-medium text-white hover:bg-red-700"
              >
                {t("entries.edit.delete")}
              </button>
            </div>
          </DialogPanel>
        </div>
      </Dialog>
    </section>
  );
}
