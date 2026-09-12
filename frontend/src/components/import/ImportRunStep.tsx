import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { api } from "../../api/client";
import type { components } from "../../api/schema";
import type { ClassifiedRow } from "../../lib/import/mapRow";
import { resolveTagIds } from "../../lib/resolveTags";
import type { RunFailure } from "./types";

type Tag = components["schemas"]["Tag"];

/**
 * Step 4: resolves the batch's tag names to ids once, then submits one
 * `POST /api/entries` per non-failed row in file order with live progress,
 * skipping a row the backend rejects rather than stopping. Cancel stops
 * further submissions between rows — entries already created stay created
 * (see design.md's "resolve tags once, then submit sequentially"
 * decision).
 */
export function ImportRunStep({
  accountId,
  rows,
  tagNames,
  tags,
  onFinished,
}: {
  accountId: string;
  /** Only non-failed rows (ok + suspicious) — every `entry` is non-null. */
  rows: ClassifiedRow[];
  tagNames: string[];
  tags: Tag[];
  onFinished: (result: {
    total: number;
    created: number;
    failed: RunFailure[];
    canceled: boolean;
  }) => void;
}) {
  const { t } = useTranslation();
  const [created, setCreated] = useState(0);
  const [done, setDone] = useState(false);
  const canceledRef = useRef(false);

  const total = rows.length;

  // biome-ignore lint/correctness/useExhaustiveDependencies: intentionally runs once on mount — re-running on a prop identity change would resubmit already-created rows.
  useEffect(() => {
    let unmounted = false;

    async function run() {
      const tagIds = await resolveTagIds(tagNames, tags);
      let createdCount = 0;
      const failures: RunFailure[] = [];

      for (const row of rows) {
        if (canceledRef.current) break;
        if (!row.entry) continue;
        const { data } = await api.POST("/api/entries", {
          body: { ...row.entry, account_id: accountId, tag_ids: tagIds },
        });
        if (data) {
          createdCount++;
          if (!unmounted) setCreated(createdCount);
        } else {
          failures.push({ index: row.index, reason: "submitRejected" });
        }
      }

      if (!unmounted) {
        setDone(true);
        onFinished({
          total,
          created: createdCount,
          failed: failures,
          canceled: canceledRef.current,
        });
      }
    }

    run();

    return () => {
      unmounted = true;
    };
  }, []);

  const percent = total === 0 ? 100 : Math.round((created / total) * 100);

  return (
    <div className="flex flex-col gap-4">
      <p className="text-sm">
        {t("entries.import.steps.run.progress", { created, total })}
      </p>
      <div className="h-2 w-full overflow-hidden rounded-full bg-black/10 dark:bg-white/10">
        <div
          className="h-full bg-zinc-900 transition-all dark:bg-white"
          style={{ width: `${percent}%` }}
        />
      </div>
      {!done && (
        <div>
          <button
            type="button"
            onClick={() => {
              canceledRef.current = true;
            }}
            className="rounded-md border border-black/15 px-4 py-2 text-sm font-medium transition-colors hover:bg-black/[.04] dark:border-white/15 dark:hover:bg-white/[.06]"
          >
            {t("entries.import.steps.run.cancel")}
          </button>
        </div>
      )}
    </div>
  );
}
