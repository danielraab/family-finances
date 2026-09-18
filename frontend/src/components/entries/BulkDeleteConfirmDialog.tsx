import { Description } from "@headlessui/react";
import { useTranslation } from "react-i18next";
import { Modal } from "../Modal";

export function BulkDeleteConfirmDialog({
  open,
  count,
  onClose,
  onConfirm,
}: {
  open: boolean;
  count: number;
  onClose: () => void;
  onConfirm: () => void;
}) {
  const { t } = useTranslation();

  return (
    <Modal
      open={open}
      onClose={onClose}
      title={t("entries.bulk.delete.title", { count })}
    >
      <Description className="text-sm text-zinc-600 dark:text-zinc-400">
        {t("entries.bulk.delete.body")}
      </Description>
      <div className="flex justify-end gap-2">
        <button
          type="button"
          onClick={onClose}
          className="rounded-md px-3 py-2 text-sm font-medium text-zinc-600 hover:bg-black/[.04] dark:text-zinc-400 dark:hover:bg-white/[.06]"
        >
          {t("entries.bulk.cancel")}
        </button>
        <button
          type="button"
          onClick={onConfirm}
          className="rounded-md bg-red-600 px-3 py-2 text-sm font-medium text-white hover:bg-red-700"
        >
          {t("entries.bulk.delete.confirm")}
        </button>
      </div>
    </Modal>
  );
}
