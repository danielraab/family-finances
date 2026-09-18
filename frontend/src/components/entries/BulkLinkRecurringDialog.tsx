import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { api } from "../../api/client";
import type { components } from "../../api/schema";
import { Modal } from "../Modal";

type RecurringTransaction = components["schemas"]["RecurringTransaction"];

const selectClass =
  "rounded-md border border-black/15 bg-transparent px-3 py-2 text-sm font-normal outline-none transition-colors focus:border-black/40 dark:border-white/15 dark:focus:border-white/40";

/**
 * Only ever opened for a single-account selection (the toolbar disables
 * this action otherwise — see design.md), so it fetches recurring
 * transactions scoped to that one account, mirroring
 * entries.$entryId.edit.tsx's own scoped fetch.
 */
export function BulkLinkRecurringDialog({
  open,
  accountId,
  onClose,
  onApply,
}: {
  open: boolean;
  accountId: string | null;
  onClose: () => void;
  onApply: (recurringTransactionId: string | null) => void;
}) {
  const { t } = useTranslation();
  const [value, setValue] = useState("");
  const [options, setOptions] = useState<RecurringTransaction[]>([]);

  useEffect(() => {
    if (!open || !accountId) return;
    let cancelled = false;
    api
      .GET("/api/recurring-transactions", {
        params: { query: { account_id: [accountId] } },
      })
      .then(({ data }) => {
        if (!cancelled) setOptions(data ?? []);
      });
    return () => {
      cancelled = true;
    };
  }, [open, accountId]);

  function handleClose() {
    setValue("");
    onClose();
  }

  return (
    <Modal
      open={open}
      onClose={handleClose}
      title={t("entries.bulk.linkRecurring.title")}
    >
      <select
        value={value}
        onChange={(e) => setValue(e.target.value)}
        className={selectClass}
      >
        <option value="">{t("entries.bulk.linkRecurring.none")}</option>
        {options.map((rt) => (
          <option key={rt.id} value={rt.id}>
            {rt.title}
          </option>
        ))}
      </select>
      <div className="flex justify-end gap-2">
        <button
          type="button"
          onClick={handleClose}
          className="rounded-md px-3 py-2 text-sm font-medium text-zinc-600 hover:bg-black/[.04] dark:text-zinc-400 dark:hover:bg-white/[.06]"
        >
          {t("entries.bulk.cancel")}
        </button>
        <button
          type="button"
          onClick={() => {
            onApply(value || null);
            setValue("");
          }}
          className="rounded-md bg-zinc-900 px-3 py-2 text-sm font-medium text-white hover:bg-zinc-700 dark:bg-white dark:text-zinc-900 dark:hover:bg-zinc-200"
        >
          {t("entries.bulk.apply")}
        </button>
      </div>
    </Modal>
  );
}
