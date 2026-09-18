import { type ReactNode, useCallback, useMemo, useState } from "react";
import type { components } from "../../api/schema";
import { EntrySummaryModal } from "./EntrySummaryModal";
import type { SummaryLookupSources } from "./lookups";
import { RecurringSummaryModal } from "./RecurringSummaryModal";

type Entry = components["schemas"]["Entry"];

type Target =
  | { kind: "entry"; entry: Entry }
  | { kind: "recurring"; id: string; back: Entry | null };

/**
 * The open summary as one piece of state, so following a cross-link
 * replaces the modal's content instead of stacking a second modal on top
 * (see the change's design.md). Every surface that lists entries wires
 * this in three lines: spread the lookups in, call `openEntry` /
 * `openRecurring` from a row, and render `summaryModals`.
 */
export function useSummaryModals(sources: SummaryLookupSources): {
  openEntry: (entry: Entry) => void;
  openRecurring: (recurringTransactionId: string) => void;
  summaryModals: ReactNode;
} {
  const [target, setTarget] = useState<Target | null>(null);
  const { accounts, categories, tags, userId, displayedDecimalPlaces } =
    sources;

  // Built here rather than by each call site so the maps are rebuilt only
  // when their source lists change, not on every render of a long ledger.
  const lookups = useMemo(
    () => ({
      accountById: new Map(accounts.map((a) => [a.id, a])),
      categoryById: new Map(categories.map((c) => [c.id, c])),
      tagById: new Map(tags.map((tag) => [tag.id, tag])),
      userId,
      displayedDecimalPlaces,
    }),
    [accounts, categories, tags, userId, displayedDecimalPlaces],
  );

  const openEntry = useCallback((entry: Entry) => {
    setTarget({ kind: "entry", entry });
  }, []);
  const openRecurring = useCallback((id: string) => {
    setTarget({ kind: "recurring", id, back: null });
  }, []);
  const close = useCallback(() => setTarget(null), []);

  const summaryModals = useMemo(() => {
    if (target === null) return null;
    if (target.kind === "entry") {
      return (
        <EntrySummaryModal
          entry={target.entry}
          lookups={lookups}
          onOpenRecurring={(id) =>
            setTarget({ kind: "recurring", id, back: target.entry })
          }
          onClose={close}
        />
      );
    }
    const back = target.back;
    return (
      <RecurringSummaryModal
        recurringTransactionId={target.id}
        lookups={lookups}
        onBack={
          back === null
            ? undefined
            : () => setTarget({ kind: "entry", entry: back })
        }
        onClose={close}
      />
    );
  }, [target, lookups, close]);

  return { openEntry, openRecurring, summaryModals };
}
