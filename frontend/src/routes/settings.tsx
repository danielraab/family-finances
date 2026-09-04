import {
  createFileRoute,
  Link,
  Outlet,
  useLocation,
  useNavigate,
} from "@tanstack/react-router";
import { useEffect } from "react";
import { useTranslation } from "react-i18next";
import { useAuth } from "../components/AuthProvider";

export const Route = createFileRoute("/settings")({
  component: SettingsLayout,
});

/**
 * The settings page's auth gate and tab chrome. The first route in the app
 * that requires authentication: an anonymous visitor is redirected to
 * /login, mirroring /login's own redirect-when-authenticated in reverse.
 * Renders nothing while useAuth is loading or the redirect is pending, so
 * there's no flash of the form before the decision is made. The Users tab
 * is only ever listed for an admin — see settings.users.tsx for the
 * matching direct-link redirect.
 */
function SettingsLayout() {
  const { status, user } = useAuth();
  const navigate = useNavigate();
  const pathname = useLocation({ select: (l) => l.pathname });
  const { t } = useTranslation();

  useEffect(() => {
    if (status === "anonymous") {
      navigate({ to: "/login", replace: true });
    }
  }, [status, navigate]);

  if (status !== "authenticated" || !user) {
    return null;
  }

  const tabs = [
    { to: "/settings" as const, label: t("settings.tabs.common") },
    ...(user.is_admin
      ? [{ to: "/settings/users" as const, label: t("settings.tabs.users") }]
      : []),
  ];

  return (
    <section className="mx-auto flex w-full max-w-3xl flex-col gap-6 px-6 py-12 sm:px-10">
      <header className="flex flex-col gap-2">
        <h1 className="text-2xl font-semibold tracking-tight">
          {t("settings.title")}
        </h1>
      </header>

      <nav className="flex gap-1 border-b border-black/10 dark:border-white/10">
        {tabs.map((tab) => {
          const active = pathname === tab.to;
          return (
            <Link
              key={tab.to}
              to={tab.to}
              aria-current={active ? "page" : undefined}
              className={`-mb-px border-b-2 px-3 py-2 text-sm font-medium transition-colors ${
                active
                  ? "border-black text-black dark:border-white dark:text-white"
                  : "border-transparent text-zinc-500 hover:text-zinc-800 dark:text-zinc-400 dark:hover:text-zinc-200"
              }`}
            >
              {tab.label}
            </Link>
          );
        })}
      </nav>

      <Outlet />
    </section>
  );
}
