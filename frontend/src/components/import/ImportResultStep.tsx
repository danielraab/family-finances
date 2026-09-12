import { useTranslation } from "react-i18next";
import type { ClassifiedRow } from "../../lib/import/mapRow";
import type { RunFailure } from "./types";

type FailedRow = { index: number; reasons: string[] };

/**
 * Step 5: the final report — created count plus one unified list of every
 * failed row (dry-run failures that were never submitted, and any row the
 * backend rejected during the run), reasons only, no per-success listing.
 */
export function ImportResultStep({
  created,
  dryRunFailedRows,
  runFailures,
  canceled,
  onStartOver,
}: {
  created: number;
  dryRunFailedRows: ClassifiedRow[];
  runFailures: RunFailure[];
  canceled: boolean;
  onStartOver: () => void;
}) {
  const { t } = useTranslation();

  const failedRows: FailedRow[] = [
    ...dryRunFailedRows.map((r) => ({
      index: r.index,
      reasons: r.issues.map((issue) =>
        t(`entries.import.reasons.${issue.reason}`),
      ),
    })),
    ...runFailures.map((f) => ({
      index: f.index,
      reasons: [t(`entries.import.reasons.${f.reason}`)],
    })),
  ].sort((a, b) => a.index - b.index);

  return (
    <div className="flex flex-col gap-4">
      <p className="text-sm font-medium">
        {t("entries.import.steps.result.created", { count: created })}
      </p>

      {canceled && (
        <p className="text-sm text-amber-600 dark:text-amber-400">
          {t("entries.import.steps.result.canceled")}
        </p>
      )}

      {failedRows.length > 0 && (
        <div className="flex flex-col gap-2">
          <p className="text-sm text-red-600 dark:text-red-400">
            {t("entries.import.steps.result.failedCount", {
              count: failedRows.length,
            })}
          </p>
          <div className="overflow-x-auto">
            <table className="w-full text-left text-sm">
              <thead>
                <tr className="text-xs font-medium text-zinc-500 dark:text-zinc-400">
                  <th className="pr-4 py-1">
                    {t("entries.import.steps.mapping.rowColumn")}
                  </th>
                  <th className="py-1">
                    {t("entries.import.steps.mapping.reasonColumn")}
                  </th>
                </tr>
              </thead>
              <tbody>
                {failedRows.map((r) => (
                  <tr
                    key={r.index}
                    className="border-t border-black/5 dark:border-white/5"
                  >
                    <td className="pr-4 py-1">{r.index + 1}</td>
                    <td className="py-1">{r.reasons.join(", ")}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      <div>
        <button
          type="button"
          onClick={onStartOver}
          className="rounded-md bg-zinc-900 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-zinc-700 dark:bg-white dark:text-zinc-900 dark:hover:bg-zinc-200"
        >
          {t("entries.import.steps.result.startOver")}
        </button>
      </div>
    </div>
  );
}
