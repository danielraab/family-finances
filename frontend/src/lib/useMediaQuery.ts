import { useSyncExternalStore } from "react";

/**
 * Reactive `matchMedia` subscription, e.g. `useMediaQuery("(min-width: 768px)")`
 * to mirror Tailwind's `md:` breakpoint in JS-driven layout decisions.
 */
export function useMediaQuery(query: string): boolean {
  return useSyncExternalStore(
    (onChange) => {
      const mql = window.matchMedia(query);
      mql.addEventListener("change", onChange);
      return () => mql.removeEventListener("change", onChange);
    },
    () => window.matchMedia(query).matches,
    () => false,
  );
}
