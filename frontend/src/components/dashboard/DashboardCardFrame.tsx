import { useTranslation } from "react-i18next";

const buttonClass =
  "rounded-md px-2 py-1 text-xs font-medium text-zinc-600 hover:bg-black/[.04] disabled:opacity-30 dark:text-zinc-400 dark:hover:bg-white/[.06]";

/**
 * Edit-mode chrome wrapped around a card's own content: ▲/▼ move buttons
 * (disabled at either end of the whole list), an edit action opening the
 * card's config form pre-filled, and a remove action — no drag-and-drop,
 * matching /categories's reorder convention so the dashboard is editable
 * the same way on a phone as on a desktop. Renders children unwrapped
 * when editMode is false.
 *
 * The toolbar sits flush against the card with no gap, sharing this
 * wrapper's own border/rounding (`overflow-hidden` clips the card's own
 * rounded-corner box to match), so it reads as one attached header/body
 * unit rather than two separately-floating boxes — a plain dashed box
 * hovering just above the card, with a visible gap, reads as unrelated to
 * it rather than acting on it.
 */
export function DashboardCardFrame({
  editMode,
  isFirst,
  isLast,
  onMoveUp,
  onMoveDown,
  onEdit,
  onRemove,
  children,
}: {
  editMode: boolean;
  isFirst: boolean;
  isLast: boolean;
  onMoveUp: () => void;
  onMoveDown: () => void;
  onEdit: () => void;
  onRemove: () => void;
  children: React.ReactNode;
}) {
  const { t } = useTranslation();
  if (!editMode) {
    return <>{children}</>;
  }
  return (
    <div className="flex flex-col overflow-hidden rounded-lg border border-black/10 dark:border-white/10">
      <div className="flex items-center justify-between gap-1 border-b border-black/10 bg-black/[.03] px-1.5 py-1 dark:border-white/10 dark:bg-white/[.05]">
        <div className="flex items-center gap-1">
          <button
            type="button"
            disabled={isFirst}
            onClick={onMoveUp}
            aria-label={t("dashboard.edit.moveUp")}
            className={buttonClass}
          >
            ▲
          </button>
          <button
            type="button"
            disabled={isLast}
            onClick={onMoveDown}
            aria-label={t("dashboard.edit.moveDown")}
            className={buttonClass}
          >
            ▼
          </button>
        </div>
        <div className="flex items-center gap-1">
          <button
            type="button"
            onClick={onEdit}
            aria-label={t("dashboard.edit.edit")}
            className={buttonClass}
          >
            ✎
          </button>
          <button
            type="button"
            onClick={onRemove}
            aria-label={t("dashboard.edit.remove")}
            className={`${buttonClass} text-red-600 dark:text-red-400`}
          >
            ✕
          </button>
        </div>
      </div>
      {children}
    </div>
  );
}
