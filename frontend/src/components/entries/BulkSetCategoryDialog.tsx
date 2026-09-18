import { useState } from "react";
import { useTranslation } from "react-i18next";
import type { CategoryOption } from "../../lib/categoryTree";
import { Modal } from "../Modal";

const selectClass =
  "rounded-md border border-black/15 bg-transparent px-3 py-2 text-sm font-normal outline-none transition-colors focus:border-black/40 dark:border-white/15 dark:focus:border-white/40";

export function BulkSetCategoryDialog({
  open,
  categoryOptions,
  onClose,
  onApply,
}: {
  open: boolean;
  categoryOptions: CategoryOption[];
  onClose: () => void;
  onApply: (categoryId: string | null) => void;
}) {
  const { t } = useTranslation();
  const [value, setValue] = useState("");

  function handleClose() {
    setValue("");
    onClose();
  }

  return (
    <Modal
      open={open}
      onClose={handleClose}
      title={t("entries.bulk.setCategory.title")}
    >
      <select
        value={value}
        onChange={(e) => setValue(e.target.value)}
        className={selectClass}
      >
        <option value="">{t("entries.bulk.setCategory.none")}</option>
        {categoryOptions.map((c) => (
          <option key={c.id} value={c.id}>
            {c.label}
            {c.shared &&
              ` — ${t("categories.shared.badgeTitle", { owner: c.ownerName ?? "" })}`}
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
