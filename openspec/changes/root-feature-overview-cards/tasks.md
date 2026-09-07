## 1. Dependency

- [x] 1.1 Add `lucide-react` to `frontend/package.json` via
      `pnpm add lucide-react` (from `frontend/`); verify `pnpm-lock.yaml`
      updates and `pnpm-workspace.yaml`'s `onlyBuiltDependencies` needs no
      change (pure JS package, no build scripts).

## 2. `FeatureOverview` component

- [x] 2.1 Create `frontend/src/components/FeatureOverview.tsx` with a
      static config array of four `{ Icon, titleKey, descriptionKey }`
      entries using `lucide-react`'s `Wallet`, `ArrowLeftRight`, `Tags`,
      and `ListTree`.
- [x] 2.2 Render a heading + translated tagline (mirroring
      `Placeholder.tsx`'s current header) above a `flex flex-col gap-4`
      list of four cards, built via `.map()` over the config array.
- [x] 2.3 Each card: `flex flex-col sm:flex-row` (alternating
      `sm:flex-row-reverse` on odd index), a `shrink-0 self-stretch`
      tinted panel (`bg-emerald-600/10 dark:bg-emerald-400/10`, rounded,
      fixed height on mobile e.g. `h-40`, fractional width at `sm:`+ e.g.
      `sm:w-2/5`) centering the icon (`text-emerald-700
      dark:text-emerald-400`, sized ~40–48px, not stretched), and a text
      side with title (`font-medium`) + description (`text-sm
      text-zinc-500 dark:text-zinc-400`), vertically centered. No link,
      button, or click handler on the card.
- [x] 2.4 Update `frontend/src/routes/index.tsx` to render
      `<FeatureOverview />` instead of `<Placeholder />`; delete
      `frontend/src/components/Placeholder.tsx`.

## 3. i18n

- [x] 3.1 In `frontend/src/i18n/locales/en.json`, add keys for the page
      heading/tagline (reusing or rewording the existing `home.tagline`)
      and `home.features.<accounts|entries|categorization|categoryTree>
      .title` / `.description` (four features, per proposal.md); remove
      `home.emptyTitle` and `home.emptyBody` (no longer referenced).
- [x] 3.2 Add the same keys to `frontend/src/i18n/locales/de.json`
      (German may lag per policy, but add it now since the change is
      small).
- [x] 3.3 Run `pnpm lint` from `frontend/`; verify clean (Biome also
      catches unused imports if `Placeholder.tsx`'s deletion leaves
      anything dangling).

## 4. Verification

- [x] 4.1 Run `pnpm exec tsc` and `pnpm build` from `frontend/`; verify
      both pass and `frontend/out/index.html` is produced.
- [x] 4.2 `pnpm dev` and manually check `/`: four cards render with the
      correct icon/title/description per feature, panels alternate
      sides, light and dark themes both look correct, and the layout
      stacks sensibly on a narrow (mobile-width) viewport.
- [x] 4.3 Confirm `/` renders identically whether visited signed-out or
      signed-in (no redirect, no content difference), and that no card is
      clickable/navigable.
