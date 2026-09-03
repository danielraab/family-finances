import { Link, useLocation } from "@tanstack/react-router";
import { useEffect } from "react";
import { useMediaQuery } from "../lib/useMediaQuery";
import { Icon } from "./Icon";
import { SidebarUser } from "./SidebarUser";

const NAV = [{ to: "/", label: "Home" }] as const;

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
          aria-label="Close sidebar"
          onClick={onCloseMobile}
          className="fixed inset-0 z-40 bg-black/40 md:hidden"
        />
      )}

      <aside
        data-collapsed={effectiveCollapsed}
        className={`fixed inset-y-0 left-0 z-50 flex w-72 flex-col border-r border-black/10 bg-white transition-transform duration-200 ease-out dark:border-white/10 dark:bg-black md:static md:inset-auto md:z-auto md:translate-x-0 ${
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

        <nav className="flex flex-1 flex-col gap-1 px-2 py-2">
          {NAV.map((item) => {
            const active = pathname === item.to;
            return (
              <Link
                key={item.to}
                to={item.to}
                onClick={onCloseMobile}
                aria-current={active ? "page" : undefined}
                title={effectiveCollapsed ? item.label : undefined}
                className={`flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors ${
                  active
                    ? "bg-black/[.06] text-black dark:bg-white/10 dark:text-white"
                    : "text-zinc-600 hover:bg-black/[.04] dark:text-zinc-400 dark:hover:bg-white/[.06]"
                } ${effectiveCollapsed ? "justify-center" : ""}`}
              >
                <span className="shrink-0">
                  <HomeGlyph />
                </span>
                {!effectiveCollapsed && (
                  <span className="truncate">{item.label}</span>
                )}
              </Link>
            );
          })}
        </nav>

        <div className="flex flex-col gap-1 border-t border-black/10 px-2 py-2 dark:border-white/10">
          <SidebarUser collapsed={effectiveCollapsed} />
        </div>
      </aside>
    </>
  );
}
