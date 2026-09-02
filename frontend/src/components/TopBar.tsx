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

export function TopBar({
  collapsed,
  onToggle,
}: {
  collapsed: boolean;
  onToggle: () => void;
}) {
  return (
    <header className="flex h-14 shrink-0 items-center justify-between border-b border-black/10 px-2 dark:border-white/10">
      <button
        type="button"
        onClick={onToggle}
        aria-label={collapsed ? "Expand sidebar" : "Collapse sidebar"}
        aria-pressed={collapsed}
        title={collapsed ? "Expand sidebar" : "Collapse sidebar"}
        className="flex items-center rounded-md p-2 text-zinc-600 transition-colors hover:bg-black/[.04] dark:text-zinc-400 dark:hover:bg-white/[.06]"
      >
        <ChevronGlyph pointing={collapsed ? "right" : "left"} />
      </button>

      <ThemeToggle iconOnly />
    </header>
  );
}
