import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { api } from "../api/client";

const SHORT_COMMIT_LENGTH = 7;

/**
 * The sidebar footer's build-identity line: fetched once from
 * `GET /api/version` (unauthenticated, so it resolves the same for an
 * anonymous visitor). Renders nothing while pending, on error, or when the
 * backend reports neither field — an absent line is the honest answer, not
 * a placeholder.
 *
 * Shows the release tag when the backend reports one, otherwise the first
 * seven characters of the commit; the full commit is always available as
 * the `title` so it can be copied into an issue. Collapsed and expanded
 * render the same short text — there's no separate label to drop.
 */
export function SidebarVersion({ collapsed }: { collapsed: boolean }) {
  const { t } = useTranslation();
  const [build, setBuild] = useState<{
    version: string;
    commit: string;
  } | null>(null);

  useEffect(() => {
    let cancelled = false;
    api.GET("/api/version").then(({ data }) => {
      if (!cancelled && data) {
        setBuild(data);
      }
    });
    return () => {
      cancelled = true;
    };
  }, []);

  if (!build) {
    return null;
  }

  const text = build.version || build.commit.slice(0, SHORT_COMMIT_LENGTH);
  if (!text) {
    return null;
  }

  return (
    <div
      title={
        build.commit
          ? t("sidebar.buildVersion", { commit: build.commit })
          : undefined
      }
      className={`truncate px-3 py-1 text-xs text-zinc-400 dark:text-zinc-600 ${
        collapsed ? "text-center" : ""
      }`}
    >
      {text}
    </div>
  );
}
