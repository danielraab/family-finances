/**
 * Temporary home content. Replace with the real dashboard once there is data
 * to show.
 */
export function Placeholder() {
  return (
    <section className="mx-auto flex w-full max-w-3xl flex-col gap-6 px-6 py-12 sm:px-10">
      <header className="flex flex-col gap-2">
        <h1 className="text-2xl font-semibold tracking-tight">
          Family Finances
        </h1>
        <p className="text-zinc-600 dark:text-zinc-400">
          Track the household&rsquo;s money in one place.
        </p>
      </header>

      <div className="flex flex-col items-center justify-center gap-2 rounded-lg border border-dashed border-black/15 px-6 py-16 text-center dark:border-white/15">
        <p className="font-medium">Nothing here yet</p>
        <p className="text-sm text-zinc-500 dark:text-zinc-400">
          This is a placeholder. Real content will land here soon.
        </p>
      </div>
    </section>
  );
}
