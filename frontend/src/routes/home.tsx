import { createFileRoute, Link, useNavigate } from "@tanstack/react-router";
import { useEffect } from "react";
import { useTranslation } from "react-i18next";
import { AccountCard } from "../components/AccountCard";
import { useAuth } from "../components/AuthProvider";
import { useAccountsWithBalances } from "../lib/useAccountsWithBalances";
import { useDisplayedDecimalPlaces } from "../lib/useDisplayedDecimalPlaces";

export const Route = createFileRoute("/home")({
  component: HomePage,
});

/**
 * The authenticated account dashboard. Single self-contained route (no
 * nested children, mirroring `categories.tsx`) doing its own auth gate: an
 * anonymous visitor is redirected to `/` — not `/login`, since this route is
 * reached from the sidebar's "Home" item, which anonymous visitors also see.
 */
function HomePage() {
  const { status } = useAuth();
  const navigate = useNavigate();

  useEffect(() => {
    if (status === "anonymous") {
      navigate({ to: "/", replace: true });
    }
  }, [status, navigate]);

  if (status !== "authenticated") {
    return null;
  }

  return <HomeDashboard />;
}

function HomeDashboard() {
  const { t, i18n } = useTranslation();
  const { accounts, balances } = useAccountsWithBalances();
  const displayedDecimalPlaces = useDisplayedDecimalPlaces();

  return (
    <section className="mx-auto flex w-full max-w-5xl flex-col gap-6 px-6 py-12 sm:px-10">
      <h1 className="text-2xl font-semibold tracking-tight">{t("nav.home")}</h1>

      {accounts === null ? null : accounts.length === 0 ? (
        <div className="flex flex-col items-center justify-center gap-3 rounded-lg border border-dashed border-black/15 px-6 py-16 text-center dark:border-white/15">
          <p className="font-medium">{t("dashboard.emptyTitle")}</p>
          <p className="text-sm text-zinc-500 dark:text-zinc-400">
            {t("dashboard.emptyBody")}
          </p>
          <Link
            to="/accounts/new"
            className="mt-2 rounded-md bg-zinc-900 px-3 py-2 text-sm font-medium text-white transition-colors hover:bg-zinc-700 dark:bg-white dark:text-zinc-900 dark:hover:bg-zinc-200"
          >
            {t("accounts.create")}
          </Link>
        </div>
      ) : (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {accounts.map((account) => (
            <AccountCard
              key={account.id}
              account={account}
              balance={balances[account.id]}
              displayedDecimalPlaces={displayedDecimalPlaces}
              locale={i18n.resolvedLanguage ?? "en"}
            />
          ))}
        </div>
      )}
    </section>
  );
}
