import { type Theme, useTheme } from "../lib/theme";

const ORDER: Theme[] = ["system", "light", "dark"];

const LABEL: Record<Theme, string> = {
  system: "System",
  light: "Light",
  dark: "Dark",
};

function SunGlyph() {
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
      <circle cx="12" cy="12" r="4" />
      <path d="M12 2v2M12 20v2M4.93 4.93l1.41 1.41M17.66 17.66l1.41 1.41M2 12h2M20 12h2M4.93 19.07l1.41-1.41M17.66 6.34l1.41-1.41" />
    </svg>
  );
}

function MoonGlyph() {
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
      <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z" />
    </svg>
  );
}

function MonitorGlyph() {
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
      <rect x="2" y="4" width="20" height="14" rx="2" />
      <path d="M8 22h8M12 18v4" />
    </svg>
  );
}

function ThemeGlyph({ choice }: { choice: Theme }) {
  if (choice === "light") return <SunGlyph />;
  if (choice === "dark") return <MoonGlyph />;
  return <MonitorGlyph />;
}

export function ThemeToggle({ iconOnly = false }: { iconOnly?: boolean }) {
  const { theme, setTheme } = useTheme();
  const next: Theme =
    ORDER[(ORDER.indexOf(theme) + 1) % ORDER.length] ?? "system";

  return (
    <button
      type="button"
      onClick={() => setTheme(next)}
      aria-label={`Theme: ${LABEL[theme]}. Switch to ${LABEL[next]}.`}
      title={iconOnly ? `Theme: ${LABEL[theme]}` : undefined}
      className={`flex items-center rounded-md text-sm font-medium text-zinc-600 transition-colors hover:bg-black/[.04] dark:text-zinc-400 dark:hover:bg-white/[.06] ${
        iconOnly ? "p-2" : "w-full gap-3 px-3 py-2"
      }`}
    >
      <span className="shrink-0">
        <ThemeGlyph choice={theme} />
      </span>
      {!iconOnly && <span className="truncate">{LABEL[theme]}</span>}
    </button>
  );
}
