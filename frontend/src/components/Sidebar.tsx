import { Link, useLocation } from "@tanstack/react-router";
import { useEffect } from "react";
import { useTranslation } from "react-i18next";
import { useMediaQuery } from "../lib/useMediaQuery";
import { Icon } from "./Icon";
import { SidebarUser } from "./SidebarUser";
import { SidebarVersion } from "./SidebarVersion";
import { ThemeSwitch } from "./ThemeSwitch";

const NAV = [
  { to: "/home", labelKey: "nav.home", glyph: "home" },
  { to: "/accounts", labelKey: "nav.accounts", glyph: "accounts" },
  { to: "/entries", labelKey: "nav.entries", glyph: "entries" },
  { to: "/recurring", labelKey: "nav.recurring", glyph: "recurring" },
  { to: "/categories", labelKey: "nav.categories", glyph: "categories" },
  { to: "/tags", labelKey: "nav.tags", glyph: "tags" },
  { to: "/reports", labelKey: "nav.reports", glyph: "reports" },
] as const;

function HomeGlyph() {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 24 24"
      width={20}
      height={20}
      fill="none"
      stroke="currentColor"
      strokeWidth={2}
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      <path d="M3 11.5 12 4l9 7.5" />
      <path d="M5 10v10h14V10" />
    </svg>
  );
}

function AccountsGlyph() {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 24 24"
      width={20}
      height={20}
      fill="none"
      stroke="currentColor"
      strokeWidth={2}
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      <rect x="3" y="6" width="18" height="13" rx="2" />
      <path d="M3 10h18" />
      <path d="M7 14h4" />
    </svg>
  );
}

function EntriesGlyph() {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 24 24"
      width={20}
      height={20}
      fill="none"
      stroke="currentColor"
      strokeWidth={2}
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      <path d="M8 6h13" />
      <path d="M8 12h13" />
      <path d="M8 18h13" />
      <path d="M3 6h.01" />
      <path d="M3 12h.01" />
      <path d="M3 18h.01" />
    </svg>
  );
}

function RecurringGlyph() {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 24 24"
      width={20}
      height={20}
      fill="none"
      stroke="currentColor"
      strokeWidth={2}
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      <path d="M17 2.1 21 6l-4 3.9" />
      <path d="M3 11V9a4 4 0 0 1 4-4h14" />
      <path d="M7 21.9 3 18l4-3.9" />
      <path d="M21 13v2a4 4 0 0 1-4 4H3" />
    </svg>
  );
}

function CategoriesGlyph() {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 24 24"
      width={20}
      height={20}
      fill="none"
      stroke="currentColor"
      strokeWidth={2}
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      <path d="M4 5h6v6H4z" />
      <path d="M14 13h6v6h-6z" />
      <path d="M7 11v3a2 2 0 0 0 2 2h1" />
      <path d="M17 13v-2a2 2 0 0 0-2-2h-1" />
    </svg>
  );
}

function TagsGlyph() {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 24 24"
      width={20}
      height={20}
      fill="none"
      stroke="currentColor"
      strokeWidth={2}
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      <path d="M12.59 3.41 4 12v7a1 1 0 0 0 1 1h7l8.59-8.59a2 2 0 0 0 0-2.83l-5.17-5.17a2 2 0 0 0-2.83 0Z" />
      <path d="M9 9h.01" />
    </svg>
  );
}

function ReportsGlyph() {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 24 24"
      width={20}
      height={20}
      fill="none"
      stroke="currentColor"
      strokeWidth={2}
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      <path d="M4 19V5" />
      <path d="M8 19v-7" />
      <path d="M12 19v-4" />
      <path d="M16 19V9" />
      <path d="M20 19V6" />
    </svg>
  );
}

const GLYPHS = {
  home: HomeGlyph,
  accounts: AccountsGlyph,
  entries: EntriesGlyph,
  recurring: RecurringGlyph,
  categories: CategoriesGlyph,
  tags: TagsGlyph,
  reports: ReportsGlyph,
} as const;

const GITHUB_REPO_URL = "https://github.com/danielraab/family-finances";

function GitHubGlyph() {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 24 24"
      width={16}
      height={16}
      fill="currentColor"
      aria-hidden="true"
    >
      <path d="M12 2C6.48 2 2 6.58 2 12.2c0 4.5 2.87 8.31 6.84 9.66.5.1.68-.22.68-.5 0-.24-.01-1.04-.01-1.9-2.78.62-3.37-1.22-3.37-1.22-.45-1.18-1.11-1.49-1.11-1.49-.91-.63.07-.62.07-.62 1 .07 1.53 1.05 1.53 1.05.9 1.55 2.36 1.11 2.94.85.09-.65.35-1.11.63-1.36-2.22-.26-4.56-1.14-4.56-5.06 0-1.12.39-2.03 1.03-2.75-.1-.26-.45-1.3.1-2.71 0 0 .84-.27 2.75 1.05a9.34 9.34 0 0 1 5 0c1.91-1.32 2.75-1.05 2.75-1.05.55 1.41.2 2.45.1 2.71.64.72 1.03 1.63 1.03 2.75 0 3.93-2.35 4.79-4.58 5.05.36.32.68.94.68 1.9 0 1.37-.01 2.47-.01 2.81 0 .28.18.61.69.5A10.02 10.02 0 0 0 22 12.2C22 6.58 17.52 2 12 2Z" />
    </svg>
  );
}

/**
 * Mobile-first: below `md` the sidebar is an off-canvas drawer, closed by
 * default, that slides over the page (with a backdrop) when `mobileOpen`.
 * At `md` and up it reverts to the persistent, collapsible column — driven
 * by `collapsed` — that was always in flow. `onCloseMobile` dismisses the
 * drawer (backdrop tap, Escape, or a nav link) and is a no-op on desktop.
 */
export function Sidebar({
  collapsed,
  mounted,
  mobileOpen,
  onCloseMobile,
}: {
  collapsed: boolean;
  mounted: boolean;
  mobileOpen: boolean;
  onCloseMobile: () => void;
}) {
  const pathname = useLocation({ select: (l) => l.pathname });
  const isDesktop = useMediaQuery("(min-width: 768px)");
  const effectiveCollapsed = isDesktop && collapsed;
  const { t } = useTranslation();

  useEffect(() => {
    if (!mobileOpen) {
      return;
    }
    function onKeyDown(e: KeyboardEvent) {
      if (e.key === "Escape") {
        onCloseMobile();
      }
    }
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [mobileOpen, onCloseMobile]);

  return (
    <>
      {mobileOpen && (
        <button
          type="button"
          aria-label={t("sidebar.closeSidebar")}
          onClick={onCloseMobile}
          className="fixed inset-0 z-40 bg-black/40 md:hidden"
        />
      )}

      <aside
        data-collapsed={effectiveCollapsed}
        className={`fixed inset-y-0 left-0 z-50 flex w-72 flex-col border-r border-black/10 bg-white transition-transform duration-200 ease-out dark:border-white/10 dark:bg-black md:sticky md:top-0 md:z-auto md:h-screen md:translate-x-0 md:self-start ${
          mobileOpen ? "translate-x-0" : "-translate-x-full"
        } ${effectiveCollapsed ? "md:w-16" : "md:w-60"} ${
          mounted ? "md:transition-[width,transform]" : ""
        }`}
      >
        <div
          className={`flex h-14 items-center gap-2 ${
            effectiveCollapsed ? "justify-center px-0" : "px-3"
          }`}
        >
          <Icon size={32} className="shrink-0" />
          {!effectiveCollapsed && (
            <span className="truncate font-semibold tracking-tight">
              Family Finances
            </span>
          )}
        </div>

        <nav className="flex flex-1 flex-col gap-1 overflow-y-auto px-2 py-2">
          {NAV.map((item) => {
            const active =
              pathname === item.to || pathname.startsWith(`${item.to}/`);
            const label = t(item.labelKey);
            const Glyph = GLYPHS[item.glyph];
            return (
              <Link
                key={item.to}
                to={item.to}
                onClick={onCloseMobile}
                aria-current={active ? "page" : undefined}
                title={effectiveCollapsed ? label : undefined}
                className={`flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors ${
                  active
                    ? "bg-black/[.06] text-black dark:bg-white/10 dark:text-white"
                    : "text-zinc-600 hover:bg-black/[.04] dark:text-zinc-400 dark:hover:bg-white/[.06]"
                } ${effectiveCollapsed ? "justify-center" : ""}`}
              >
                <span className="shrink-0">
                  <Glyph />
                </span>
                {!effectiveCollapsed && (
                  <span className="truncate">{label}</span>
                )}
              </Link>
            );
          })}
        </nav>

        <div className="flex flex-col gap-1 border-t border-black/10 px-2 py-2 dark:border-white/10">
          <div
            className={`flex items-center gap-1 ${
              effectiveCollapsed ? "flex-col" : "justify-between"
            }`}
          >
            <ThemeSwitch collapsed={effectiveCollapsed} />
            <a
              href={GITHUB_REPO_URL}
              target="_blank"
              rel="noopener noreferrer"
              title={t("sidebar.githubRepo")}
              aria-label={t("sidebar.githubRepo")}
              className="inline-flex shrink-0 rounded-md p-1.5 text-zinc-400 transition-colors hover:bg-black/[.04] hover:text-zinc-600 dark:text-zinc-500 dark:hover:bg-white/[.06] dark:hover:text-zinc-300"
            >
              <GitHubGlyph />
            </a>
          </div>
          <SidebarUser collapsed={effectiveCollapsed} />
          <SidebarVersion collapsed={effectiveCollapsed} />
        </div>
      </aside>
    </>
  );
}
