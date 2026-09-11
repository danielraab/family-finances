import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { type FormEvent, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { api } from "../api/client";
import type { components } from "../api/schema";
import { SignedAmountInput } from "../components/SignedAmountInput";
import { TagInput } from "../components/TagInput";
import { inputToAmount } from "../lib/amount";
import { flattenCategoryTree } from "../lib/categoryTree";
import { compact } from "../lib/compact";

type Account = components["schemas"]["Account"];
type Category = components["schemas"]["Category"];
type Tag = components["schemas"]["Tag"];
type EntryKind = components["schemas"]["EntryKind"];

type NewEntrySearch = { account_id?: string | undefined };

export const Route = createFileRoute("/entries/new")({
  validateSearch: (search: Record<string, unknown>): NewEntrySearch => ({
    account_id:
      typeof search["account_id"] === "string"
        ? search["account_id"]
        : undefined,
  }),
  component: NewEntry,
});

function nowLocalInput(): string {
  const d = new Date();
  d.setSeconds(0, 0);
  const tzOffsetMs = d.getTimezoneOffset() * 60_000;
  return new Date(d.getTime() - tzOffsetMs).toISOString().slice(0, 16);
}

const inputClass =
  "rounded-md border border-black/15 bg-transparent px-3 py-2 text-sm font-normal outline-none transition-colors focus:border-black/40 dark:border-white/15 dark:focus:border-white/40";

function NewEntry() {
  const { account_id: presetAccountId } = Route.useSearch();
  const { t } = useTranslation();
  const navigate = useNavigate();

  const [accounts, setAccounts] = useState<Account[]>([]);
  const [categories, setCategories] = useState<Category[]>([]);
  const [tags, setTags] = useState<Tag[]>([]);

  const [accountId, setAccountId] = useState(presetAccountId ?? "");
  const [kind, setKind] = useState<EntryKind>("transaction");
  const [amount, setAmount] = useState("");
  const [transactionAmount, setTransactionAmount] = useState("");
  const [transactionNegative, setTransactionNegative] = useState(true);
  const [bookingTimestamp, setBookingTimestamp] = useState(nowLocalInput);
  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [categoryId, setCategoryId] = useState("");
  const [tagNames, setTagNames] = useState<string[]>([]);

  const [invalidField, setInvalidField] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    Promise.all([
      api.GET("/api/accounts"),
      api.GET("/api/categories"),
      api.GET("/api/tags"),
    ]).then(([a, c, tg]) => {
      setAccounts(a.data ?? []);
      setCategories(c.data ?? []);
      setTags(tg.data ?? []);
    });
  }, []);

  const account = accounts.find((a) => a.id === accountId);
  // A new entry never starts with a category, so a disabled one is simply
  // never offered — unlike editing, there's no existing value to preserve.
  // A category shared at view tier only is excluded too — it can be
  // filtered/seen by, but never newly selected on an entry; an
  // append-shared one (or an owned one, permission "owner") is offered
  // like any other, always flat — its parent_id is cleared so it can never
  // nest even in the rare case its real parent happens to also be shared
  // with this caller (ids are otherwise unrelated to this caller's own
  // tree, so there's no other reason it would ever nest).
  const categoryOptions = flattenCategoryTree(
    categories
      .filter((c) => !c.disabled && c.permission !== "view")
      .map((c) => {
        if (!c.shared) return c;
        const { parent_id, ...rest } = c;
        return rest;
      }),
  );
  // view-only accounts can't have entries created against them — never
  // offered when picking freely; a preset account_id (the ?account_id=
  // flow, field locked either way) is left as-is rather than filtered.
  const selectableAccounts = accounts.filter((a) => a.permission !== "view");

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
    if (!accountId) {
      setInvalidField("account_id");
      return;
    }
    if (title.trim() === "") {
      setInvalidField("title");
      return;
    }
    let parsedAmount: number | null;
    if (kind === "transaction") {
      const magnitude = inputToAmount(transactionAmount);
      if (magnitude === null || magnitude === 0) {
        setInvalidField("amount");
        return;
      }
      parsedAmount = transactionNegative ? -magnitude : magnitude;
    } else {
      parsedAmount = inputToAmount(amount);
      if (parsedAmount === null) {
        setInvalidField("amount");
        return;
      }
    }
    if (kind === "transaction" && !categoryId) {
      setInvalidField("category_id");
      return;
    }
    setInvalidField(null);
    setSubmitting(true);
    setError(null);

    const tagIds = await resolveTagIds();
    const { data, response } = await api.POST("/api/entries", {
      body: {
        account_id: accountId,
        kind,
        booking_timestamp: new Date(bookingTimestamp).toISOString(),
        title: title.trim(),
        tag_ids: tagIds,
        ...(kind === "transaction"
          ? { amount: parsedAmount }
          : { balance: parsedAmount }),
        ...compact({
          description: description.trim() || undefined,
          category_id: categoryId || undefined,
        }),
      },
    });
    setSubmitting(false);
    if (!response.ok || !data) {
      setError(t("entries.form.saveError"));
      return;
    }
    navigate({ to: "/entries", search: { account_id: accountId } });
  }

  return (
    <section className="mx-auto flex w-full max-w-xl flex-col gap-6 px-6 py-12 sm:px-10">
      <h1 className="text-2xl font-semibold tracking-tight">
        {t("entries.new.title")}
      </h1>

      <form onSubmit={handleSubmit} className="flex flex-col gap-4">
        <label className="flex flex-col gap-1.5 text-sm font-medium">
          {t("entries.form.account")}
          <select
            value={accountId}
            onChange={(e) => setAccountId(e.target.value)}
            className={inputClass}
            disabled={presetAccountId !== undefined}
            required
          >
            <option value="" disabled>
              {t("entries.form.accountPlaceholder")}
            </option>
            {selectableAccounts.map((a) => (
              <option key={a.id} value={a.id}>
                {a.title} ({a.currency})
              </option>
            ))}
          </select>
          {invalidField === "account_id" && (
            <span className="text-xs font-normal text-red-600 dark:text-red-400">
              {t("entries.form.accountRequired")}
            </span>
          )}
        </label>

        <fieldset className="flex flex-col gap-1.5 text-sm font-medium">
          {t("entries.form.kind")}
          <div className="flex gap-4 text-sm font-normal">
            <label className="flex items-center gap-1.5">
              <input
                type="radio"
                checked={kind === "transaction"}
                onChange={() => setKind("transaction")}
              />
              {t("entries.kind.transaction")}
            </label>
            <label className="flex items-center gap-1.5">
              <input
                type="radio"
                checked={kind === "balance_adjustment"}
                onChange={() => setKind("balance_adjustment")}
              />
              {t("entries.kind.balanceAdjustment")}
            </label>
          </div>
        </fieldset>

        {kind === "transaction" ? (
          <div className="flex flex-col gap-1.5 text-sm font-medium">
            {t("entries.form.amount", { currency: account?.currency ?? "" })}
            <SignedAmountInput
              magnitude={transactionAmount}
              onMagnitudeChange={setTransactionAmount}
              negative={transactionNegative}
              onNegativeChange={setTransactionNegative}
              currency={account?.currency ?? ""}
              invalid={invalidField === "amount"}
            />
            {invalidField === "amount" && (
              <span className="text-xs font-normal text-red-600 dark:text-red-400">
                {inputToAmount(transactionAmount) === 0
                  ? t("entries.form.amountZero")
                  : t("entries.form.amountInvalid")}
              </span>
            )}
          </div>
        ) : (
          <label className="flex flex-col gap-1.5 text-sm font-medium">
            {t("entries.form.amount", { currency: account?.currency ?? "" })}
            <input
              value={amount}
              onChange={(e) => setAmount(e.target.value)}
              inputMode="decimal"
              placeholder="0.00"
              className={inputClass}
              required
            />
            {invalidField === "amount" && (
              <span className="text-xs font-normal text-red-600 dark:text-red-400">
                {t("entries.form.amountInvalid")}
              </span>
            )}
          </label>
        )}

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
              {kind === "balance_adjustment"
                ? t("entries.form.categoryNone")
                : t("entries.form.categoryPlaceholder")}
            </option>
            {categoryOptions.map((c) => (
              <option key={c.id} value={c.id}>
                {c.label}
                {c.shared &&
                  ` — ${t("categories.shared.badgeTitle", { owner: c.ownerName ?? "" })}`}
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
            existingTags={tags.filter((tg) => !tg.disabled)}
          />
        </div>

        {error && (
          <p className="text-sm text-red-600 dark:text-red-400">{error}</p>
        )}

        <div>
          <button
            type="submit"
            disabled={submitting}
            className="rounded-md bg-zinc-900 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-zinc-700 disabled:opacity-60 dark:bg-white dark:text-zinc-900 dark:hover:bg-zinc-200"
          >
            {t("entries.form.create")}
          </button>
        </div>
      </form>
    </section>
  );
}
