import { ChevronDown } from "lucide-react";
import { type ReactNode, useState } from "react";
import { useTranslation } from "react-i18next";

/**
 * The bordered filter block both `/entries` and `/recurring` put above
 * their list: a heading, a count of the filters currently applied, a
 * clear-all action, and the controls themselves in a responsive grid.
 *
 * Below the `sm` breakpoint the controls collapse behind the heading,
 * which doubles as the toggle — on a phone a page's worth of filter
 * chrome would otherwise stand between the visitor and the first row.
 * From `sm` up they are always shown and the toggle is not rendered.
 *
 * This is a presentational shell. It owns only its open/closed state; the
 * page owns what counts as an applied filter and what clearing them means,
 * since that differs between the two pages (`/entries` counts its date
 * range once across three URL parameters, `/recurring` folds its revealed
 * both-legs flag into the checkbox that reveals it).
 */
export function FilterPanel({
  id,
  activeCount,
  onClearAll,
  children,
}: {
  /** Ties the toggle's `aria-controls` to the controls it reveals. */
  id: string;
  activeCount: number;
  onClearAll: () => void;
  children: ReactNode;
}) {
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);

  return (
    <div className="flex flex-col gap-3 rounded-lg border border-black/10 p-3 dark:border-white/10">
      <div className="flex items-center justify-between gap-3">
        <button
          type="button"
          onClick={() => setOpen((wasOpen) => !wasOpen)}
          aria-expanded={open}
          aria-controls={id}
          className="flex items-center gap-2 text-sm font-medium sm:hidden"
        >
          {t("filters.heading")}
          {activeCount > 0 && (
            <>
              <span
                aria-hidden="true"
                className="rounded-full bg-zinc-900 px-1.5 py-0.5 text-xs leading-none text-white dark:bg-white dark:text-zinc-900"
              >
                {activeCount}
              </span>
              <span className="sr-only">
                {t("filters.activeCount", { count: activeCount })}
              </span>
            </>
          )}
          <ChevronDown
            size={14}
            aria-hidden="true"
            className={`text-zinc-400 transition-transform dark:text-zinc-500 ${open ? "rotate-180" : ""}`}
          />
        </button>
        <span className="hidden text-sm font-medium sm:inline">
          {t("filters.heading")}
        </span>
        {activeCount > 0 && (
          <button
            type="button"
            onClick={onClearAll}
            className="rounded-md border border-black/15 px-2.5 py-1 text-xs font-medium text-zinc-600 transition-colors hover:bg-black/[.04] dark:border-white/15 dark:text-zinc-300 dark:hover:bg-white/[.06]"
          >
            {t("filters.clearAll")}
          </button>
        )}
      </div>

      <div
        id={id}
        className={`${open ? "grid" : "hidden"} grid-cols-1 items-end gap-3 sm:grid sm:grid-cols-2 lg:grid-cols-3`}
      >
        {children}
      </div>
    </div>
  );
}

/**
 * The shared class list for a control inside a `FilterPanel` — exported so
 * both pages' selects and inputs sit on one line rather than two copies of
 * it.
 */
export const filterControlClass =
  "w-full rounded-md border border-black/15 bg-transparent px-2.5 py-1.5 text-sm font-normal outline-none transition-colors focus:border-black/40 dark:border-white/15 dark:focus:border-white/40";
