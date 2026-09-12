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
import { LocationField } from "../components/LocationField";
import { SignedAmountInput } from "../components/SignedAmountInput";
import { TagInput } from "../components/TagInput";
import { amountToInput, inputToAmount } from "../lib/amount";
import { flattenCategoryTree } from "../lib/categoryTree";
import { compact } from "../lib/compact";
import { resolveTagIds } from "../lib/resolveTags";

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
  const { user } = useAuth();
  const navigate = useNavigate();

  const [entry, setEntry] = useState<Entry | null | undefined>(undefined);
  const [account, setAccount] = useState<Account | null>(null);
  const [accounts, setAccounts] = useState<Account[]>([]);
  const [categories, setCategories] = useState<Category[]>([]);
  const [tags, setTags] = useState<Tag[]>([]);

  const [amount, setAmount] = useState("");
  const [transactionAmount, setTransactionAmount] = useState("");
  const [transactionNegative, setTransactionNegative] = useState(true);
  const [bookingTimestamp, setBookingTimestamp] = useState("");
  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [categoryId, setCategoryId] = useState("");
  const [counterparty, setCounterparty] = useState("");
  const [counterparties, setCounterparties] = useState<string[]>([]);
  const [location, setLocation] = useState("");
  const [tagNames, setTagNames] = useState<string[]>([]);
  const [accountUnlocked, setAccountUnlocked] = useState(false);
  const [selectedAccountId, setSelectedAccountId] = useState("");
  const [pendingAmount, setPendingAmount] = useState<number | null>(null);

  const [invalidField, setInvalidField] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [confirmingDelete, setConfirmingDelete] = useState(false);
  const [confirmingAccountChange, setConfirmingAccountChange] = useState(false);

  useEffect(() => {
    let cancelled = false;
    Promise.all([
      api.GET("/api/entries/{id}", { params: { path: { id: entryId } } }),
      api.GET("/api/categories"),
      api.GET("/api/tags"),
      api.GET("/api/accounts"),
      api.GET("/api/entries/counterparties"),
    ]).then(([entryRes, categoriesRes, tagsRes, accountsRes, cpRes]) => {
      if (cancelled) return;
      setCategories(categoriesRes.data ?? []);
      const allTags = tagsRes.data ?? [];
      setTags(allTags);
      setAccounts(accountsRes.data ?? []);
      setCounterparties(cpRes.data ?? []);
      const e = entryRes.data ?? null;
      setEntry(e);
      if (e) {
        if (e.kind === "transaction") {
          setTransactionNegative(e.amount < 0);
          setTransactionAmount(amountToInput(Math.abs(e.amount)));
        } else {
          setAmount(amountToInput(e.balance ?? 0));
        }
        setBookingTimestamp(toLocalInput(e.booking_timestamp));
        setTitle(e.title);
        setDescription(e.description ?? "");
        setCategoryId(e.category_id ?? "");
        setCounterparty(e.counterparty ?? "");
        setLocation(e.location ?? "");
        setSelectedAccountId(e.account_id);
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

  async function performSubmit(parsedAmount: number) {
    if (!entry) return;
    setConfirmingAccountChange(false);
    setSubmitting(true);
    setError(null);

    const tagIds = await resolveTagIds(tagNames, tags);
    const { data, response } = await api.PATCH("/api/entries/{id}", {
      params: { path: { id: entryId } },
      body: {
        account_id: selectedAccountId,
        booking_timestamp: new Date(bookingTimestamp).toISOString(),
        title: title.trim(),
        category_id: categoryId || null,
        tag_ids: tagIds,
        ...(entry.kind === "transaction"
          ? { amount: parsedAmount }
          : { balance: parsedAmount }),
        ...compact({
          description: description.trim() || undefined,
          ...(entry.kind === "transaction"
            ? {
                counterparty: counterparty.trim(),
                location: location.trim(),
              }
            : {}),
        }),
      },
    });
    setSubmitting(false);
    if (!response.ok || !data) {
      setError(t("entries.form.saveError"));
      return;
    }
    navigate({
      to: "/entries",
      search: { account_id: selectedAccountId },
    });
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!entry) return;
    if (title.trim() === "") {
      setInvalidField("title");
      return;
    }
    let parsedAmount: number | null;
    if (entry.kind === "transaction") {
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
    if (entry.kind === "transaction" && !categoryId) {
      setInvalidField("category_id");
      return;
    }
    setInvalidField(null);

    if (selectedAccountId !== entry.account_id) {
      setPendingAmount(parsedAmount);
      setConfirmingAccountChange(true);
      return;
    }
    await performSubmit(parsedAmount);
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

  // A category disabled since this entry was categorized still renders as
  // the current, selected value (so the form doesn't look like it lost
  // data) but isn't offered as a choice for switching to a different one —
  // mirroring AccountForm.tsx's disabled account-type handling. Leaving it
  // untouched and saving other fields is unaffected either way. A category
  // shared at view tier only (or since unshared/downgraded below append)
  // gets the same treatment — it can't be newly selected, but stays as the
  // current value if that's what the entry already carries.
  const currentCategory = categories.find((c) => c.id === categoryId);
  // A shared category's parent_id is cleared before flattening so it always
  // renders top-level, even in the rare case its real parent happens to
  // also be shared with this caller — mirrors entries.new.tsx.
  const categoryOptions = flattenCategoryTree(
    categories
      .filter(
        (c) => (!c.disabled && c.permission !== "view") || c.id === categoryId,
      )
      .map((c) => {
        if (!c.shared) return c;
        const { parent_id, ...rest } = c;
        return rest;
      }),
  );

  // The entry's current account still renders as a selectable option while
  // unlocked even if it has since been disabled, but isn't offered once a
  // different account has been chosen — mirroring the category picker's
  // disabled-current-value handling above.
  const accountOptions = accounts.filter(
    (a) => !a.disabled || a.id === selectedAccountId,
  );
  const selectedAccount = accounts.find((a) => a.id === selectedAccountId);
  const currencyMismatch =
    account !== null &&
    selectedAccount !== undefined &&
    selectedAccount.currency !== account.currency;

  const originalAccountId = entry.account_id;
  function cancelAccountChange() {
    setSelectedAccountId(originalAccountId);
    setAccountUnlocked(false);
  }

  // entry_admin/owner may edit any entry on the account; append may edit
  // only what they themselves created; view (or append on someone else's
  // entry) is read-only — see account-entries' design.md.
  const canEdit =
    account !== null &&
    (account.permission === "entry_admin" ||
      account.permission === "owner" ||
      (account.permission === "append" && entry.created_by === user?.id));

  return (
    <section className="mx-auto flex w-full max-w-xl flex-col gap-8 px-6 py-12 sm:px-10">
      <div className="flex flex-col gap-1">
        <h1 className="text-2xl font-semibold tracking-tight">
          {t("entries.edit.title")}
        </h1>
        {entry.created_by !== user?.id && entry.created_by_name && (
          <p className="text-sm text-zinc-500 dark:text-zinc-400">
            {t("entries.createdBy", { name: entry.created_by_name })}
          </p>
        )}
      </div>

      <form onSubmit={handleSubmit} className="flex flex-col gap-4">
        <fieldset disabled={!canEdit} className="contents">
          <div className="flex flex-col gap-1.5 text-sm font-medium">
            {t("entries.form.account")}
            {accountUnlocked ? (
              <div className="flex items-center gap-2">
                <select
                  value={selectedAccountId}
                  onChange={(e) => setSelectedAccountId(e.target.value)}
                  className={`${inputClass} flex-1`}
                >
                  {accountOptions.map((a) => (
                    <option
                      key={a.id}
                      value={a.id}
                      disabled={a.disabled && a.id === selectedAccountId}
                    >
                      {a.title}
                      {a.disabled && a.id === selectedAccountId
                        ? ` (${t("entries.form.accountDisabledOption")})`
                        : ""}
                    </option>
                  ))}
                </select>
                <button
                  type="button"
                  onClick={cancelAccountChange}
                  aria-label={t("entries.form.cancelAccountChange")}
                  className="rounded-md px-2 py-2 text-sm font-medium text-zinc-600 hover:bg-black/[.04] dark:text-zinc-400 dark:hover:bg-white/[.06]"
                >
                  ✕
                </button>
              </div>
            ) : (
              <div className="flex items-center gap-2">
                <input
                  value={account?.title ?? entry.account_id}
                  disabled
                  className={`${inputClass} flex-1 opacity-60`}
                />
                <button
                  type="button"
                  onClick={() => setAccountUnlocked(true)}
                  aria-label={t("entries.form.changeAccount")}
                  className="rounded-md px-2 py-2 text-sm font-medium text-zinc-600 hover:bg-black/[.04] dark:text-zinc-400 dark:hover:bg-white/[.06]"
                >
                  ✎
                </button>
              </div>
            )}
            {currencyMismatch && selectedAccount && account && (
              <span className="text-xs font-normal text-amber-600 dark:text-amber-400">
                {t("entries.form.accountCurrencyWarning", {
                  currency: selectedAccount.currency,
                  originalCurrency: account.currency,
                })}
              </span>
            )}
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

          {entry.kind === "transaction" ? (
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
                {entry.kind === "balance_adjustment"
                  ? t("entries.form.categoryNone")
                  : t("entries.form.categoryPlaceholder")}
              </option>
              {categoryOptions.map((c) => (
                <option
                  key={c.id}
                  value={c.id}
                  disabled={currentCategory?.disabled && c.id === categoryId}
                >
                  {c.label}
                  {c.shared &&
                    ` — ${t("categories.shared.badgeTitle", { owner: c.ownerName ?? "" })}`}
                  {currentCategory?.disabled && c.id === categoryId
                    ? ` (${t("entries.form.categoryDisabledOption")})`
                    : ""}
                </option>
              ))}
            </select>
            {invalidField === "category_id" && (
              <span className="text-xs font-normal text-red-600 dark:text-red-400">
                {t("entries.form.categoryRequired")}
              </span>
            )}
          </label>

          {entry.kind === "transaction" && (
            <>
              <label className="flex flex-col gap-1.5 text-sm font-medium">
                {t("entries.form.counterparty")}
                <input
                  list="counterparty-suggestions"
                  value={counterparty}
                  onChange={(e) => setCounterparty(e.target.value)}
                  placeholder={t("entries.form.counterpartyPlaceholder")}
                  className={inputClass}
                />
                <datalist id="counterparty-suggestions">
                  {counterparties.map((c) => (
                    <option key={c} value={c} />
                  ))}
                </datalist>
              </label>

              <div className="flex flex-col gap-1.5 text-sm font-medium">
                {t("entries.form.location")}
                <LocationField
                  value={location}
                  onChange={setLocation}
                  inputClassName={inputClass}
                />
              </div>
            </>
          )}

          <div className="flex flex-col gap-1.5 text-sm font-medium">
            {t("entries.form.tags")}
            {/* A view-only shared tag can be seen/filtered by but never
                newly attached, so it's excluded from suggestions here —
                an already-attached tag stays in `tagNames` regardless,
                so downgrading a tag below append never drops it off an
                entry that already carries it. */}
            <TagInput
              value={tagNames}
              onChange={setTagNames}
              existingTags={tags.filter(
                (tg) => !tg.disabled && tg.permission !== "view",
              )}
            />
          </div>

          {error && (
            <p className="text-sm text-red-600 dark:text-red-400">{error}</p>
          )}
        </fieldset>

        {canEdit && (
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
        )}
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

      <Dialog
        open={confirmingAccountChange}
        onClose={() => setConfirmingAccountChange(false)}
        className="relative z-50"
      >
        <div className="fixed inset-0 bg-black/40" aria-hidden="true" />
        <div className="fixed inset-0 flex items-center justify-center p-4">
          <DialogPanel className="flex w-full max-w-sm flex-col gap-4 rounded-lg bg-white p-6 dark:bg-neutral-900">
            <DialogTitle className="text-base font-semibold">
              {t("entries.edit.confirmAccountChangeTitle")}
            </DialogTitle>
            <Description className="text-sm text-zinc-600 dark:text-zinc-400">
              {t("entries.edit.confirmAccountChangeBody", {
                account: selectedAccount?.title ?? selectedAccountId,
              })}
            </Description>
            {currencyMismatch && selectedAccount && account && (
              <p className="text-sm text-amber-600 dark:text-amber-400">
                {t("entries.form.accountCurrencyWarning", {
                  currency: selectedAccount.currency,
                  originalCurrency: account.currency,
                })}
              </p>
            )}
            <div className="flex justify-end gap-2">
              <button
                type="button"
                onClick={() => setConfirmingAccountChange(false)}
                className="rounded-md px-3 py-2 text-sm font-medium text-zinc-600 hover:bg-black/[.04] dark:text-zinc-400 dark:hover:bg-white/[.06]"
              >
                {t("accounts.edit.confirm.cancel")}
              </button>
              <button
                type="button"
                onClick={() => {
                  if (pendingAmount !== null) void performSubmit(pendingAmount);
                }}
                className="rounded-md bg-zinc-900 px-3 py-2 text-sm font-medium text-white hover:bg-zinc-700 dark:bg-white dark:text-zinc-900 dark:hover:bg-zinc-200"
              >
                {t("entries.edit.confirmAccountChangeConfirm")}
              </button>
            </div>
          </DialogPanel>
        </div>
      </Dialog>
    </section>
  );
}
