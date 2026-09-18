import { useState } from "react";
import { useTranslation } from "react-i18next";
import type { components } from "../../api/schema";
import { Modal } from "../Modal";
import { TagInput } from "../TagInput";

type Tag = components["schemas"]["Tag"];

const TITLE_KEYS = {
  add: "entries.bulk.addTags.title",
  remove: "entries.bulk.removeTags.title",
  set: "entries.bulk.setTags.title",
} as const;

/**
 * Shared by Add tags / Remove tags / Set tags — all three are "pick some
 * tags," so all three reuse the entry form's TagInput. Remove tags is
 * passed only the tags actually present on the selection (existingTags),
 * restricting what it *offers*; the caller (BulkActionToolbar) is
 * responsible for only resolving chosen names that match a known tag for
 * that mode, never creating one — see design.md.
 */
export function BulkTagsDialog({
  open,
  mode,
  existingTags,
  onClose,
  onApply,
}: {
  open: boolean;
  mode: "add" | "remove" | "set";
  existingTags: Tag[];
  onClose: () => void;
  onApply: (names: string[]) => void;
}) {
  const { t } = useTranslation();
  const [names, setNames] = useState<string[]>([]);

  function handleClose() {
    setNames([]);
    onClose();
  }

  return (
    <Modal open={open} onClose={handleClose} title={t(TITLE_KEYS[mode])}>
      <TagInput value={names} onChange={setNames} existingTags={existingTags} />
      {/* onMouseDown preventDefault, mirroring TagInput's own suggestion
          buttons: without it, mousedown here blurs the tag input first,
          collapsing its suggestion list and shifting these buttons up
          before mouseup completes the click — occasionally missing it
          entirely. */}
      <div className="flex justify-end gap-2">
        <button
          type="button"
          onMouseDown={(event) => event.preventDefault()}
          onClick={handleClose}
          className="rounded-md px-3 py-2 text-sm font-medium text-zinc-600 hover:bg-black/[.04] dark:text-zinc-400 dark:hover:bg-white/[.06]"
        >
          {t("entries.bulk.cancel")}
        </button>
        <button
          type="button"
          disabled={names.length === 0}
          onMouseDown={(event) => event.preventDefault()}
          onClick={() => {
            onApply(names);
            setNames([]);
          }}
          className="rounded-md bg-zinc-900 px-3 py-2 text-sm font-medium text-white transition-colors hover:bg-zinc-700 disabled:opacity-50 dark:bg-white dark:text-zinc-900 dark:hover:bg-zinc-200"
        >
          {t("entries.bulk.apply")}
        </button>
      </div>
    </Modal>
  );
}
