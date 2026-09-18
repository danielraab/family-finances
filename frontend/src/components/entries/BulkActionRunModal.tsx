import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import type { BulkTask } from "../../lib/bulkEntryActions";
import { Modal } from "../Modal";

type FailedItem = { id: string; title: string; reason: string };
type Result = { succeeded: number; failed: FailedItem[] };

/**
 * Runs a bulk action's tasks sequentially (mirrors ImportRunStep.tsx's
 * shape for "many entries, one call each"), showing live progress, then a
 * succeeded/failed breakdown with Reload list / Close — see design.md.
 * Mounted only while a run is in flight or showing its result; the parent
 * unmounts it by clearing its task list.
 */
export function BulkActionRunModal({
  tasks,
  onClose,
  onReload,
}: {
  tasks: BulkTask[];
  onClose: () => void;
  onReload: () => void;
}) {
  const { t } = useTranslation();
  const [done, setDone] = useState(0);
  const [result, setResult] = useState<Result | null>(null);

  // biome-ignore lint/correctness/useExhaustiveDependencies: intentionally runs once on mount for this task batch — a new batch means a freshly mounted modal.
  useEffect(() => {
    let unmounted = false;

    async function run() {
      let succeeded = 0;
      const failed: FailedItem[] = [];
      for (const task of tasks) {
        if (unmounted) return;
        const outcome = await task.run();
        if (outcome.ok) {
          succeeded++;
        } else {
          failed.push({
            id: task.id,
            title: task.title,
            reason: outcome.reason,
          });
        }
        if (!unmounted) setDone((d) => d + 1);
      }
      if (!unmounted) setResult({ succeeded, failed });
    }

    run();
    return () => {
      unmounted = true;
    };
  }, []);

  const total = tasks.length;
  const percent = total === 0 ? 100 : Math.round((done / total) * 100);
  const finished = result !== null;

  return (
    <Modal
      open
      onClose={onClose}
      dismissable={finished}
      title={t("entries.bulk.run.title")}
    >
      {!finished ? (
        <>
          <p className="text-sm text-zinc-600 dark:text-zinc-400">
            {t("entries.bulk.run.progress", { done, total })}
          </p>
          <div className="h-2 w-full overflow-hidden rounded-full bg-black/10 dark:bg-white/10">
            <div
              className="h-full bg-zinc-900 transition-all dark:bg-white"
              style={{ width: `${percent}%` }}
            />
          </div>
        </>
      ) : (
        <>
          <p className="text-sm">
            {t("entries.bulk.run.result", {
              succeeded: result.succeeded,
              failed: result.failed.length,
            })}
          </p>
          {result.failed.length > 0 && (
            <ul className="flex max-h-48 flex-col gap-1 overflow-y-auto text-sm">
              {result.failed.map((f) => (
                <li key={f.id} className="text-red-600 dark:text-red-400">
                  {f.title} — {t(`entries.bulk.run.reasons.${f.reason}`)}
                </li>
              ))}
            </ul>
          )}
          <div className="flex justify-end gap-2">
            <button
              type="button"
              onClick={onClose}
              className="rounded-md px-3 py-2 text-sm font-medium text-zinc-600 hover:bg-black/[.04] dark:text-zinc-400 dark:hover:bg-white/[.06]"
            >
              {t("entries.bulk.run.close")}
            </button>
            <button
              type="button"
              onClick={onReload}
              className="rounded-md bg-zinc-900 px-3 py-2 text-sm font-medium text-white hover:bg-zinc-700 dark:bg-white dark:text-zinc-900 dark:hover:bg-zinc-200"
            >
              {t("entries.bulk.run.reload")}
            </button>
          </div>
        </>
      )}
    </Modal>
  );
}
