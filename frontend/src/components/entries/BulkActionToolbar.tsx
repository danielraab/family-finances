import { useState } from "react";
import { useTranslation } from "react-i18next";
import type { components } from "../../api/schema";
import {
  type BulkTask,
  buildDeleteTasks,
  buildRecurringTasks,
  buildSetCategoryTasks,
  buildTagTasks,
} from "../../lib/bulkEntryActions";
import { flattenCategoryTree } from "../../lib/categoryTree";
import { resolveTagIds } from "../../lib/resolveTags";
import { BulkActionRunModal } from "./BulkActionRunModal";
import { BulkDeleteConfirmDialog } from "./BulkDeleteConfirmDialog";
import { BulkLinkRecurringDialog } from "./BulkLinkRecurringDialog";
import { BulkSetCategoryDialog } from "./BulkSetCategoryDialog";
import { BulkTagsDialog } from "./BulkTagsDialog";

type Entry = components["schemas"]["Entry"];
type Category = components["schemas"]["Category"];
type Tag = components["schemas"]["Tag"];

type ActiveDialog =
  | null
  | "category"
  | "addTags"
  | "removeTags"
  | "setTags"
  | "recurring"
  | "delete";

const actionButtonClass =
  "rounded-md border border-black/15 px-3 py-1.5 text-sm font-medium transition-colors hover:bg-black/[.04] dark:border-white/15 dark:hover:bg-white/[.06] disabled:cursor-not-allowed disabled:opacity-40 disabled:hover:bg-transparent";

/**
 * The entry ledger's bulk-action toolbar: rendered above the table
 * whenever at least one row is selected. Owns which action's picker
 * dialog is open and, once confirmed, hands the resulting task list to
 * BulkActionRunModal — see design.md's "one shared run modal across all
 * six actions" decision.
 */
export function BulkActionToolbar({
  items,
  selectedIds,
  onClear,
  categories,
  tags,
  onReload,
}: {
  items: Entry[];
  selectedIds: Set<string>;
  onClear: () => void;
  categories: Category[];
  tags: Tag[];
  onReload: () => void;
}) {
  const { t } = useTranslation();
  const [activeDialog, setActiveDialog] = useState<ActiveDialog>(null);
  const [runTasks, setRunTasks] = useState<BulkTask[] | null>(null);

  const selectedEntries = items.filter((e) => selectedIds.has(e.id));
  const selectedAccountIds = new Set(selectedEntries.map((e) => e.account_id));
  const singleAccountId =
    selectedAccountIds.size === 1
      ? (selectedEntries[0]?.account_id ?? null)
      : null;

  // A shared category's parent_id is stripped before flattening so it
  // always renders top-level — mirrors entries.$entryId.edit.tsx.
  const categoryOptions = flattenCategoryTree(
    categories
      .filter((c) => !c.disabled && c.permission !== "view")
      .map((c) => {
        if (!c.shared) return c;
        const { parent_id, ...rest } = c;
        return rest;
      }),
  );
  const creatableTags = tags.filter(
    (tg) => !tg.disabled && tg.permission !== "view",
  );
  const usedTagIds = new Set(selectedEntries.flatMap((e) => e.tag_ids));
  const usedTags = tags.filter((tg) => usedTagIds.has(tg.id));

  function startRun(tasks: BulkTask[]) {
    setActiveDialog(null);
    setRunTasks(tasks);
  }

  async function applyAddOrSetTags(names: string[], mode: "add" | "set") {
    const tagIds = await resolveTagIds(names, tags);
    startRun(buildTagTasks(selectedEntries, tagIds, mode));
  }

  function applyRemoveTags(names: string[]) {
    // Never creates a tag: an unmatched typed name simply resolves to
    // nothing and is dropped — see design.md / BulkTagsDialog.
    const byName = new Map(usedTags.map((tg) => [tg.name, tg.id]));
    const tagIds = names
      .map((name) => byName.get(name))
      .filter((id): id is string => id !== undefined);
    startRun(buildTagTasks(selectedEntries, tagIds, "remove"));
  }

  return (
    <div className="flex flex-wrap items-center gap-3 rounded-lg border border-black/10 bg-black/[.03] px-3 py-2 text-sm dark:border-white/10 dark:bg-white/[.04]">
      <span className="font-medium">
        {t("entries.bulk.selectedCount", { count: selectedIds.size })}
      </span>
      <button
        type="button"
        onClick={onClear}
        className="text-zinc-500 underline-offset-2 hover:underline dark:text-zinc-400"
      >
        {t("entries.bulk.clear")}
      </button>
      <div className="ml-auto flex flex-wrap items-center gap-2">
        <button
          type="button"
          onClick={() => setActiveDialog("category")}
          className={actionButtonClass}
        >
          {t("entries.bulk.actions.setCategory")}
        </button>
        <button
          type="button"
          onClick={() => setActiveDialog("addTags")}
          className={actionButtonClass}
        >
          {t("entries.bulk.actions.addTags")}
        </button>
        <button
          type="button"
          onClick={() => setActiveDialog("removeTags")}
          className={actionButtonClass}
        >
          {t("entries.bulk.actions.removeTags")}
        </button>
        <button
          type="button"
          onClick={() => setActiveDialog("setTags")}
          className={actionButtonClass}
        >
          {t("entries.bulk.actions.setTags")}
        </button>
        <button
          type="button"
          onClick={() => setActiveDialog("recurring")}
          disabled={singleAccountId === null}
          title={
            singleAccountId === null
              ? t("entries.bulk.linkRecurring.disabledHint")
              : undefined
          }
          className={actionButtonClass}
        >
          {t("entries.bulk.actions.linkRecurring")}
        </button>
        <button
          type="button"
          onClick={() => setActiveDialog("delete")}
          className="rounded-md border border-red-200 px-3 py-1.5 text-sm font-medium text-red-600 transition-colors hover:bg-red-50 dark:border-red-900/50 dark:text-red-400 dark:hover:bg-red-950/30"
        >
          {t("entries.bulk.actions.delete")}
        </button>
      </div>

      <BulkSetCategoryDialog
        open={activeDialog === "category"}
        categoryOptions={categoryOptions}
        onClose={() => setActiveDialog(null)}
        onApply={(categoryId) =>
          startRun(buildSetCategoryTasks(selectedEntries, categoryId))
        }
      />
      <BulkTagsDialog
        open={activeDialog === "addTags"}
        mode="add"
        existingTags={creatableTags}
        onClose={() => setActiveDialog(null)}
        onApply={(names) => applyAddOrSetTags(names, "add")}
      />
      <BulkTagsDialog
        open={activeDialog === "removeTags"}
        mode="remove"
        existingTags={usedTags}
        onClose={() => setActiveDialog(null)}
        onApply={applyRemoveTags}
      />
      <BulkTagsDialog
        open={activeDialog === "setTags"}
        mode="set"
        existingTags={creatableTags}
        onClose={() => setActiveDialog(null)}
        onApply={(names) => applyAddOrSetTags(names, "set")}
      />
      <BulkLinkRecurringDialog
        open={activeDialog === "recurring"}
        accountId={singleAccountId}
        onClose={() => setActiveDialog(null)}
        onApply={(recurringTransactionId) =>
          startRun(buildRecurringTasks(selectedEntries, recurringTransactionId))
        }
      />
      <BulkDeleteConfirmDialog
        open={activeDialog === "delete"}
        count={selectedEntries.length}
        onClose={() => setActiveDialog(null)}
        onConfirm={() => startRun(buildDeleteTasks(selectedEntries))}
      />

      {runTasks && (
        <BulkActionRunModal
          tasks={runTasks}
          onClose={() => setRunTasks(null)}
          onReload={() => {
            setRunTasks(null);
            onReload();
          }}
        />
      )}
    </div>
  );
}
