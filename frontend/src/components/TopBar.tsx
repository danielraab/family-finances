import { useMediaQuery } from "../lib/useMediaQuery";
import { ThemeToggle } from "./ThemeToggle";

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

function MenuGlyph() {
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
      <path d="M4 7h16" />
      <path d="M4 12h16" />
      <path d="M4 17h16" />
    </svg>
  );
}

function XGlyph() {
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
      <path d="M6 6l12 12" />
      <path d="M18 6L6 18" />
    </svg>
  );
}

/**
 * Below `md` the toggle button opens/closes the off-canvas sidebar drawer
 * (hamburger / X). At `md` and up it collapses/expands the persistent
 * sidebar column (chevron) instead — both drive the same `onToggle`, which
 * branches on viewport at click time.
 */
export function TopBar({
  collapsed,
  mobileOpen,
  onToggle,
}: {
  collapsed: boolean;
  mobileOpen: boolean;
  onToggle: () => void;
}) {
  const isDesktop = useMediaQuery("(min-width: 768px)");
  const label = isDesktop
    ? collapsed
      ? "Expand sidebar"
      : "Collapse sidebar"
    : mobileOpen
      ? "Close sidebar"
      : "Open sidebar";

  return (
    <header className="flex h-14 shrink-0 items-center justify-between border-b border-black/10 px-2 dark:border-white/10">
      <button
        type="button"
        onClick={onToggle}
        aria-label={label}
        aria-pressed={isDesktop ? collapsed : mobileOpen}
        title={label}
        className="flex items-center rounded-md p-2 text-zinc-600 transition-colors hover:bg-black/[.04] dark:text-zinc-400 dark:hover:bg-white/[.06]"
      >
        <span className="md:hidden">
          {mobileOpen ? <XGlyph /> : <MenuGlyph />}
        </span>
        <span className="hidden md:inline-flex">
          <ChevronGlyph pointing={collapsed ? "right" : "left"} />
        </span>
      </button>

      <ThemeToggle iconOnly />
    </header>
  );
}
