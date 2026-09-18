import type { ReactNode } from "react";

/**
 * One labelled row in a summary modal. Renders nothing when its value is
 * absent, so a summary can list every field a model *could* carry and let
 * the data decide which rows appear — see `EntrySummaryModal`.
 */
export function SummaryField({
  label,
  children,
}: {
  label: string;
  children: ReactNode;
}) {
  if (
    children === null ||
    children === undefined ||
    children === false ||
    children === ""
  ) {
    return null;
  }
  return (
    <div className="flex flex-col gap-0.5">
      <dt className="text-xs font-medium text-zinc-500 dark:text-zinc-400">
        {label}
      </dt>
      <dd className="text-sm">{children}</dd>
    </div>
  );
}

/** The `<dl>` a summary's fields live in. */
export function SummaryFields({ children }: { children: ReactNode }) {
  return <dl className="flex flex-col gap-3">{children}</dl>;
}

/** The action row every summary ends with. */
export function SummaryActions({ children }: { children: ReactNode }) {
  return (
    <div className="flex flex-wrap items-center justify-end gap-2 border-t border-black/10 pt-3 dark:border-white/10">
      {children}
    </div>
  );
}

export const summaryLinkClass =
  "rounded-md bg-zinc-900 px-3 py-2 text-sm font-medium text-white hover:bg-zinc-700 dark:bg-white dark:text-zinc-900 dark:hover:bg-zinc-200";

export const summaryQuietButtonClass =
  "rounded-md px-3 py-2 text-sm font-medium text-zinc-600 hover:bg-black/[.04] dark:text-zinc-400 dark:hover:bg-white/[.06]";
