import { createFileRoute, Link, useNavigate } from "@tanstack/react-router";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { api } from "../api/client";
import type { components } from "../api/schema";
import { useAuth } from "../components/AuthProvider";
import { AccountStatCard } from "../components/dashboard/AccountStatCard";
import { BarChartCard } from "../components/dashboard/BarChartCard";
import { CardFormDialog } from "../components/dashboard/CardFormDialog";
import { DashboardCardFrame } from "../components/dashboard/DashboardCardFrame";
import { EntryListCard } from "../components/dashboard/EntryListCard";
import { LineChartCard } from "../components/dashboard/LineChartCard";
import { QueryStatCard } from "../components/dashboard/QueryStatCard";
import type { DashboardCard } from "../lib/dashboardFilter";
import { useAccountsWithBalances } from "../lib/useAccountsWithBalances";
import { useDisplayedDecimalPlaces } from "../lib/useDisplayedDecimalPlaces";
import { useWeekStart } from "../lib/useWeekStart";

type Category = components["schemas"]["Category"];
type Tag = components["schemas"]["Tag"];

export const Route = createFileRoute("/home")({
  component: HomePage,
});

/**
 * The authenticated account dashboard. Single self-contained route (no
 * nested children, mirroring `categories.tsx`) doing its own auth gate: an
 * anonymous visitor is redirected to `/` — not `/login` — since this route is
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

/** Segments an ordered card list into grid runs (3-4 per row) broken by
 * standalone, always-full-width bar_chart/line_chart cards — see
 * web-client-home's layout requirement. */
type Segment =
  | { kind: "grid"; cards: DashboardCard[] }
  | { kind: "chart"; card: DashboardCard };

/** entry_list is the only card type with a configurable grid span (2-4
 * columns of the responsive 1/2/3/4-column grid, default 2 — see
 * dashboard-cards' spec). account_stat/query_stat always occupy exactly
 * one column (no class needed); bar_chart/line_chart are handled
 * separately, always full width. Literal, statically-written class
 * strings — required for Tailwind's scanner to emit them, since it never
 * evaluates runtime-constructed class names. */
function entryListSpanClass(columns: number | undefined): string {
  switch (columns) {
    case 3:
      return "sm:col-span-2 lg:col-span-3 xl:col-span-3";
    case 4:
      return "sm:col-span-2 lg:col-span-3 xl:col-span-4";
    default:
      return "sm:col-span-2 lg:col-span-2 xl:col-span-2";
  }
}

function segmentCards(cards: DashboardCard[]): Segment[] {
  const segments: Segment[] = [];
  for (const card of cards) {
    if (card.type === "bar_chart" || card.type === "line_chart") {
      segments.push({ kind: "chart", card });
      continue;
    }
    const last = segments[segments.length - 1];
    if (last?.kind === "grid") {
      last.cards.push(card);
    } else {
      segments.push({ kind: "grid", cards: [card] });
    }
  }
  return segments;
}

function HomeDashboard() {
  const { t, i18n } = useTranslation();
  const { accounts, balances } = useAccountsWithBalances();
  const displayedDecimalPlaces = useDisplayedDecimalPlaces();
  const weekStart = useWeekStart();
  const locale = i18n.resolvedLanguage ?? "en";

  const [categories, setCategories] = useState<Category[]>([]);
  const [tags, setTags] = useState<Tag[]>([]);
  const [cards, setCards] = useState<DashboardCard[] | null>(null);
  const [editMode, setEditMode] = useState(false);
  const [dialogOpen, setDialogOpen] = useState(false);
  const [editingCard, setEditingCard] = useState<DashboardCard | null>(null);

  function openAddDialog() {
    setEditingCard(null);
    setDialogOpen(true);
  }

  function openEditDialog(card: DashboardCard) {
    setEditingCard(card);
    setDialogOpen(true);
  }

  function handleSaved(saved: DashboardCard) {
    setCards((prev) => {
      const list = prev ?? [];
      const exists = list.some((c) => c.id === saved.id);
      return exists
        ? list.map((c) => (c.id === saved.id ? saved : c))
        : [...list, saved];
    });
  }

  useEffect(() => {
    Promise.all([
      api.GET("/api/categories"),
      api.GET("/api/tags"),
      api.GET("/api/dashboard/cards"),
    ]).then(([c, tg, dc]) => {
      setCategories(c.data ?? []);
      setTags(tg.data ?? []);
      setCards(dc.data ?? []);
    });
  }, []);

  function refreshCards() {
    api.GET("/api/dashboard/cards").then(({ data }) => setCards(data ?? []));
  }

  async function moveUp(id: string) {
    await api.POST("/api/dashboard/cards/{id}/move-up", {
      params: { path: { id } },
    });
    refreshCards();
  }

  async function moveDown(id: string) {
    await api.POST("/api/dashboard/cards/{id}/move-down", {
      params: { path: { id } },
    });
    refreshCards();
  }

  async function removeCard(id: string) {
    await api.DELETE("/api/dashboard/cards/{id}", {
      params: { path: { id } },
    });
    refreshCards();
  }

  function renderCardContent(card: DashboardCard) {
    switch (card.type) {
      case "account_stat":
        return (
          <AccountStatCard
            config={card.config}
            accounts={accounts ?? []}
            balances={balances}
            displayedDecimalPlaces={displayedDecimalPlaces}
            locale={locale}
          />
        );
      case "query_stat":
        return (
          <QueryStatCard
            config={card.config}
            accounts={accounts ?? []}
            categories={categories}
            tags={tags}
            weekStart={weekStart}
            displayedDecimalPlaces={displayedDecimalPlaces}
            locale={locale}
          />
        );
      case "entry_list":
        return (
          <EntryListCard
            config={card.config}
            accounts={accounts ?? []}
            categories={categories}
            tags={tags}
            weekStart={weekStart}
            displayedDecimalPlaces={displayedDecimalPlaces}
            locale={locale}
          />
        );
      case "bar_chart":
        return (
          <BarChartCard
            config={card.config}
            accounts={accounts ?? []}
            categories={categories}
            tags={tags}
            displayedDecimalPlaces={displayedDecimalPlaces}
            locale={locale}
          />
        );
      case "line_chart":
        return (
          <LineChartCard
            config={card.config}
            accounts={accounts ?? []}
            categories={categories}
            tags={tags}
            displayedDecimalPlaces={displayedDecimalPlaces}
            locale={locale}
          />
        );
      default:
        return null;
    }
  }

  function renderCard(card: DashboardCard, allCards: DashboardCard[]) {
    return (
      <DashboardCardFrame
        key={card.id}
        editMode={editMode}
        isFirst={allCards[0]?.id === card.id}
        isLast={allCards[allCards.length - 1]?.id === card.id}
        onMoveUp={() => moveUp(card.id)}
        onMoveDown={() => moveDown(card.id)}
        onEdit={() => openEditDialog(card)}
        onRemove={() => removeCard(card.id)}
      >
        {renderCardContent(card)}
      </DashboardCardFrame>
    );
  }

  if (accounts === null || cards === null) {
    return null;
  }

  const hasAccounts = accounts.length > 0;
  const hasCards = cards.length > 0;

  return (
    <section className="mx-auto flex w-full max-w-5xl flex-col gap-6 px-6 py-12 sm:px-10">
      <div className="flex items-center justify-between gap-4">
        <h1 className="text-2xl font-semibold tracking-tight">
          {t("nav.home")}
        </h1>
        {hasAccounts && hasCards && (
          <button
            type="button"
            onClick={() => setEditMode((e) => !e)}
            className="rounded-md border border-black/10 px-3 py-1.5 text-sm font-medium text-zinc-600 transition-colors hover:bg-black/[.04] dark:border-white/10 dark:text-zinc-400 dark:hover:bg-white/[.06]"
          >
            {editMode ? t("dashboard.edit.done") : t("dashboard.edit.toggle")}
          </button>
        )}
      </div>

      {!hasAccounts ? (
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
      ) : !hasCards ? (
        <div className="flex flex-col items-center justify-center gap-3 rounded-lg border border-dashed border-black/15 px-6 py-16 text-center dark:border-white/15">
          <p className="font-medium">{t("dashboard.noCardsTitle")}</p>
          <p className="text-sm text-zinc-500 dark:text-zinc-400">
            {t("dashboard.noCardsBody")}
          </p>
          <button
            type="button"
            onClick={openAddDialog}
            className="mt-2 rounded-md bg-zinc-900 px-3 py-2 text-sm font-medium text-white transition-colors hover:bg-zinc-700 dark:bg-white dark:text-zinc-900 dark:hover:bg-zinc-200"
          >
            {t("dashboard.addCard.trigger")}
          </button>
        </div>
      ) : (
        <>
          {segmentCards(cards).map((segment) =>
            segment.kind === "chart" ? (
              <div key={segment.card.id}>{renderCard(segment.card, cards)}</div>
            ) : (
              <div
                key={segment.cards.map((c) => c.id).join(",")}
                className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4"
              >
                {segment.cards.map((c) => (
                  <div
                    key={c.id}
                    className={
                      c.type === "entry_list"
                        ? entryListSpanClass(c.config.columns)
                        : undefined
                    }
                  >
                    {renderCard(c, cards)}
                  </div>
                ))}
              </div>
            ),
          )}

          {editMode && (
            <button
              type="button"
              onClick={openAddDialog}
              className="self-start rounded-md border border-dashed border-black/15 px-3 py-2 text-sm font-medium text-zinc-600 transition-colors hover:bg-black/[.04] dark:border-white/15 dark:text-zinc-400 dark:hover:bg-white/[.06]"
            >
              {t("dashboard.addCard.trigger")}
            </button>
          )}
        </>
      )}

      <CardFormDialog
        open={dialogOpen}
        onClose={() => setDialogOpen(false)}
        editingCard={editingCard}
        onSaved={handleSaved}
        accounts={accounts}
        categories={categories}
        tags={tags}
        weekStart={weekStart}
      />
    </section>
  );
}
