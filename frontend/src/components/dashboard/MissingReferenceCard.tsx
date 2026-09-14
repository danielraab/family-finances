import { useTranslation } from "react-i18next";

/**
 * Rendered in place of any card whose config references an account/
 * category/tag that no longer resolves in the caller's own accounts/
 * categories/tags (a revoked share, a soft delete) — same footprint as a
 * live card, no data fetch attempted, per web-client-home's "stale
 * reference" requirement. Still removable via edit mode's ordinary remove
 * action, wired by the caller.
 */
export function MissingReferenceCard() {
  const { t } = useTranslation();
  return (
    <div className="flex min-h-24 flex-col items-center justify-center gap-1 rounded-lg border border-dashed border-black/15 p-4 text-center dark:border-white/15">
      <p className="text-sm font-medium text-zinc-500 dark:text-zinc-400">
        {t("dashboard.cardMissingReference")}
      </p>
    </div>
  );
}
