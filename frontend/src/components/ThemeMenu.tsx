import { MenuItem } from "@headlessui/react";
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
      width={18}
      height={18}
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
      width={18}
      height={18}
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
      width={18}
      height={18}
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

function CheckGlyph() {
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
      <path d="M20 6L9 17l-5-5" />
    </svg>
  );
}

/**
 * The three explicit theme choices (system / light / dark), meant as
 * `<MenuItem>`s inside an existing Headless UI `<Menu>` — e.g. the sidebar's
 * account dropdown — rather than a standalone always-visible control.
 */
export function ThemeMenuItems() {
  const { theme, setTheme } = useTheme();

  return (
    <>
      {ORDER.map((choice) => (
        <MenuItem key={choice}>
          {({ focus }) => (
            <button
              type="button"
              onClick={() => setTheme(choice)}
              className={`flex w-full items-center gap-3 rounded-md px-3 py-2 text-left text-sm font-medium text-zinc-700 dark:text-zinc-300 ${
                focus ? "bg-black/[.05] dark:bg-white/[.08]" : ""
              }`}
            >
              <span className="shrink-0">
                <ThemeGlyph choice={choice} />
              </span>
              <span className="flex-1 truncate">{LABEL[choice]}</span>
              {theme === choice && (
                <span className="shrink-0 text-zinc-500 dark:text-zinc-400">
                  <CheckGlyph />
                </span>
              )}
            </button>
          )}
        </MenuItem>
      ))}
    </>
  );
}
