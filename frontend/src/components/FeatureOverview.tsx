import type { LucideIcon } from "lucide-react";
import { ArrowLeftRight, ListTree, Tags, Wallet } from "lucide-react";
import { useTranslation } from "react-i18next";

const FEATURES: {
  icon: LucideIcon;
  titleKey: string;
  descriptionKey: string;
}[] = [
  {
    icon: Wallet,
    titleKey: "home.features.accounts.title",
    descriptionKey: "home.features.accounts.description",
  },
  {
    icon: ArrowLeftRight,
    titleKey: "home.features.entries.title",
    descriptionKey: "home.features.entries.description",
  },
  {
    icon: Tags,
    titleKey: "home.features.categorization.title",
    descriptionKey: "home.features.categorization.description",
  },
  {
    icon: ListTree,
    titleKey: "home.features.categoryTree.title",
    descriptionKey: "home.features.categoryTree.description",
  },
];

/**
 * The root page's (`/`) content: a fixed, audience-independent overview of
 * the app's core capabilities, shown identically whether the visitor is
 * signed in or not. Replaces the old `Placeholder`.
 */
export function FeatureOverview() {
  const { t } = useTranslation();

  return (
    <section className="mx-auto flex w-full max-w-3xl flex-col gap-8 px-6 py-12 sm:px-10">
      <header className="flex flex-col gap-2">
        <h1 className="text-2xl font-semibold tracking-tight">
          Family Finances
        </h1>
        <p className="text-zinc-600 dark:text-zinc-400">{t("home.tagline")}</p>
      </header>

      <div className="flex flex-col gap-4">
        {FEATURES.map((feature, index) => (
          <FeatureCard
            key={feature.titleKey}
            {...feature}
            reverse={index % 2 === 1}
          />
        ))}
      </div>
    </section>
  );
}

function FeatureCard({
  icon: Icon,
  titleKey,
  descriptionKey,
  reverse,
}: {
  icon: LucideIcon;
  titleKey: string;
  descriptionKey: string;
  reverse: boolean;
}) {
  const { t } = useTranslation();

  return (
    <article
      className={`flex flex-col overflow-hidden rounded-lg border border-black/10 sm:flex-row dark:border-white/10 ${
        reverse ? "sm:flex-row-reverse" : ""
      }`}
    >
      <div className="flex h-40 shrink-0 items-center justify-center bg-emerald-600/10 sm:h-auto sm:w-2/5 dark:bg-emerald-400/10">
        <Icon
          className="text-emerald-700 dark:text-emerald-400"
          size={48}
          strokeWidth={1.5}
          aria-hidden="true"
        />
      </div>
      <div className="flex flex-col justify-center gap-1 p-6">
        <h2 className="font-medium">{t(titleKey)}</h2>
        <p className="text-sm text-zinc-500 dark:text-zinc-400">
          {t(descriptionKey)}
        </p>
      </div>
    </article>
  );
}
