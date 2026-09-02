import { Link, useLocation } from "@tanstack/react-router";
import { useEffect, useState } from "react";
import { Icon } from "./Icon";
import { SidebarUser } from "./SidebarUser";
import { ThemeToggle } from "./ThemeToggle";

const STORAGE_KEY = "ff:sidebar-collapsed";

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

function ChevronGlyph({ pointing }: { pointing: "left" | "right" }) {
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
      {pointing === "left" ? (
        <path d="M15 6l-6 6 6 6" />
      ) : (
        <path d="M9 6l6 6-6 6" />
      )}
    </svg>
  );
}

function readCollapsed(): boolean {
  try {
    return window.localStorage.getItem(STORAGE_KEY) === "true";
  } catch {
    return false;
  }
}

export function Sidebar() {
  const pathname = useLocation({ select: (l) => l.pathname });
  const [collapsed, setCollapsed] = useState(readCollapsed);
  const [mounted, setMounted] = useState(false);

  useEffect(() => {
    setMounted(true);
  }, []);

  function toggle() {
    setCollapsed((prev) => {
      const next = !prev;
      try {
        window.localStorage.setItem(STORAGE_KEY, String(next));
      } catch {
        // ignore persistence failure
      }
      return next;
    });
  }

  return (
    <aside
      data-collapsed={collapsed}
      className={`flex shrink-0 flex-col border-r border-black/10 bg-white dark:border-white/10 dark:bg-black ${
        collapsed ? "w-16" : "w-60"
      } ${mounted ? "transition-[width] duration-200 ease-out" : ""}`}
    >
      <div
        className={`flex h-14 items-center gap-2 ${
          collapsed ? "justify-center px-0" : "px-3"
        }`}
      >
        <Icon size={32} className="shrink-0" />
        {!collapsed && (
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
              aria-current={active ? "page" : undefined}
              title={collapsed ? item.label : undefined}
              className={`flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors ${
                active
                  ? "bg-black/[.06] text-black dark:bg-white/10 dark:text-white"
                  : "text-zinc-600 hover:bg-black/[.04] dark:text-zinc-400 dark:hover:bg-white/[.06]"
              } ${collapsed ? "justify-center" : ""}`}
            >
              <span className="shrink-0">
                <HomeGlyph />
              </span>
              {!collapsed && <span className="truncate">{item.label}</span>}
            </Link>
          );
        })}
      </nav>

      <div className="flex flex-col gap-1 border-t border-black/10 px-2 py-2 dark:border-white/10">
        <SidebarUser collapsed={collapsed} />
        <ThemeToggle collapsed={collapsed} />
        <button
          type="button"
          onClick={toggle}
          aria-label={collapsed ? "Expand sidebar" : "Collapse sidebar"}
          aria-pressed={collapsed}
          title={collapsed ? "Expand sidebar" : undefined}
          className={`flex w-full items-center gap-3 rounded-md px-3 py-2 text-sm font-medium text-zinc-600 transition-colors hover:bg-black/[.04] dark:text-zinc-400 dark:hover:bg-white/[.06] ${
            collapsed ? "justify-center" : ""
          }`}
        >
          <span className="shrink-0">
            <ChevronGlyph pointing={collapsed ? "right" : "left"} />
          </span>
          {!collapsed && <span className="truncate">Collapse</span>}
        </button>
      </div>
    </aside>
  );
}
