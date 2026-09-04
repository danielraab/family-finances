import { Menu, MenuButton, MenuItem, MenuItems } from "@headlessui/react";
import { Link } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";
import { useAuth } from "./AuthProvider";
import { Avatar } from "./Avatar";

const ROW =
  "flex w-full items-center gap-3 rounded-md px-3 py-2 text-sm font-medium text-zinc-600 transition-colors hover:bg-black/[.04] dark:text-zinc-400 dark:hover:bg-white/[.06]";

function PersonGlyph() {
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
      <circle cx="12" cy="8" r="4" />
      <path d="M4 21c0-4 3.5-7 8-7s8 3 8 7" />
    </svg>
  );
}

function LogInGlyph() {
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
      <path d="M15 3h4a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2h-4" />
      <path d="M10 17l5-5-5-5" />
      <path d="M15 12H3" />
    </svg>
  );
}

function GearGlyph() {
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
      <circle cx="12" cy="12" r="3" />
      <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 1 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 1 1-2.83-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 1 1 2.83-2.83l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 1 1 2.83 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z" />
    </svg>
  );
}

function PowerGlyph() {
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
      <path d="M18.36 6.64a9 9 0 1 1-12.73 0" />
      <path d="M12 2v10" />
    </svg>
  );
}

function SelectorGlyph() {
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
      <path d="M7 9l5-5 5 5" />
      <path d="M7 15l5 5 5-5" />
    </svg>
  );
}

/**
 * The sidebar footer user control. Renders per `useAuth()` state:
 * - loading  → a neutral person glyph, no label (so a signed-in visitor never
 *   sees "Log in" flash before their identity resolves);
 * - anonymous → a "Log in" link to /login;
 * - authenticated → the initials avatar + name/email, opening a menu with
 *   "Settings" (links to /settings) and "Log out".
 * Takes the sidebar's `collapsed` state to switch to a glyph/avatar-only row.
 */
export function SidebarUser({ collapsed }: { collapsed: boolean }) {
  const { status, user, logout } = useAuth();
  const { t } = useTranslation();

  if (status === "loading" || (status === "authenticated" && !user)) {
    return (
      <div
        aria-hidden="true"
        className={`flex items-center gap-3 px-3 py-2 text-zinc-400 dark:text-zinc-600 ${
          collapsed ? "justify-center" : ""
        }`}
      >
        <span className="shrink-0">
          <PersonGlyph />
        </span>
        {!collapsed && (
          <span className="h-3.5 w-24 rounded bg-black/[.06] dark:bg-white/[.08]" />
        )}
      </div>
    );
  }

  if (status === "anonymous" || !user) {
    return (
      <Link
        to="/login"
        title={collapsed ? t("user.logIn") : undefined}
        className={`${ROW} ${collapsed ? "justify-center" : ""}`}
      >
        <span className="shrink-0">
          <LogInGlyph />
        </span>
        {!collapsed && <span className="truncate">{t("user.logIn")}</span>}
      </Link>
    );
  }

  const name = user.display_name?.trim();

  return (
    <Menu>
      <MenuButton
        title={collapsed ? name || user.email : undefined}
        className={`${ROW} ${collapsed ? "justify-center" : ""}`}
      >
        <Avatar
          id={user.id}
          email={user.email}
          displayName={user.display_name}
          size={collapsed ? 24 : 28}
        />
        {!collapsed && (
          <span className="flex min-w-0 flex-1 flex-col text-left leading-tight">
            {name && (
              <span className="truncate text-zinc-900 dark:text-zinc-100">
                {name}
              </span>
            )}
            <span className="truncate text-xs text-zinc-500 dark:text-zinc-400">
              {user.email}
            </span>
          </span>
        )}
        {!collapsed && (
          <span className="shrink-0 text-zinc-400 dark:text-zinc-500">
            <SelectorGlyph />
          </span>
        )}
      </MenuButton>

      <MenuItems
        anchor={{ to: collapsed ? "right end" : "top start", gap: 8 }}
        className="z-50 w-52 rounded-md border border-black/10 bg-white p-1 shadow-lg focus:outline-none dark:border-white/10 dark:bg-neutral-900"
      >
        <MenuItem>
          {({ focus }) => (
            <Link
              to="/settings"
              className={`flex w-full items-center gap-3 rounded-md px-3 py-2 text-left text-sm font-medium text-zinc-700 dark:text-zinc-300 ${
                focus ? "bg-black/[.05] dark:bg-white/[.08]" : ""
              }`}
            >
              <span className="shrink-0">
                <GearGlyph />
              </span>
              {t("user.settings")}
            </Link>
          )}
        </MenuItem>
        <MenuItem>
          {({ focus }) => (
            <button
              type="button"
              onClick={() => logout()}
              className={`flex w-full items-center gap-3 rounded-md px-3 py-2 text-left text-sm font-medium text-zinc-700 dark:text-zinc-300 ${
                focus ? "bg-black/[.05] dark:bg-white/[.08]" : ""
              }`}
            >
              <span className="shrink-0">
                <PowerGlyph />
              </span>
              {t("user.logOut")}
            </button>
          )}
        </MenuItem>
      </MenuItems>
    </Menu>
  );
}
