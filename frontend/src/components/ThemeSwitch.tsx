import { useTranslation } from "react-i18next";
import { type Theme, useTheme } from "../lib/theme";

const ORDER: Theme[] = ["system", "light", "dark"];

const LABEL_KEY: Record<Theme, string> = {
  system: "theme.system",
  light: "theme.light",
  dark: "theme.dark",
};

function SunGlyph() {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 24 24"
      width={16}
      height={16}
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
      width={16}
      height={16}
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
      width={16}
      height={16}
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

/**
 * The theme control, visible to every visitor regardless of sign-in state
 * (it's a device preference, not an account setting) — so it lives in
 * `Sidebar`, not the account menu. Expanded, it's a single-row segmented
 * pill (System/Light/Dark side by side); collapsed to icon-only width, that
 * doesn't fit, so it shrinks to one button that cycles through the three.
 */
export function ThemeSwitch({ collapsed }: { collapsed: boolean }) {
  const { theme, setTheme } = useTheme();
  const { t } = useTranslation();

  if (collapsed) {
    const next = ORDER[(ORDER.indexOf(theme) + 1) % ORDER.length] ?? "system";
    return (
      <button
        type="button"
        onClick={() => setTheme(next)}
        aria-label={t("theme.switchTo", {
          current: t(LABEL_KEY[theme]),
          next: t(LABEL_KEY[next]),
        })}
        title={t("theme.current", { label: t(LABEL_KEY[theme]) })}
        className="flex items-center justify-center rounded-md p-2 text-zinc-600 transition-colors hover:bg-black/[.04] dark:text-zinc-400 dark:hover:bg-white/[.06]"
      >
        <ThemeGlyph choice={theme} />
      </button>
    );
  }

  return (
    <div className="flex items-center gap-0.5 rounded-md bg-black/[.04] p-0.5 dark:bg-white/[.06]">
      {ORDER.map((choice) => (
        <button
          key={choice}
          type="button"
          aria-pressed={theme === choice}
          aria-label={t("theme.current", { label: t(LABEL_KEY[choice]) })}
          title={t(LABEL_KEY[choice])}
          onClick={() => setTheme(choice)}
          className={`flex flex-1 items-center justify-center rounded p-1.5 transition-colors ${
            theme === choice
              ? "bg-white text-black shadow-sm dark:bg-neutral-700 dark:text-white"
              : "text-zinc-500 hover:text-zinc-700 dark:text-zinc-400 dark:hover:text-zinc-200"
          }`}
        >
          <ThemeGlyph choice={choice} />
        </button>
      ))}
    </div>
  );
}
