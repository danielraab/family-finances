"use client";

import { ThemeProvider } from "next-themes";
import type { ReactNode } from "react";

/**
 * Client-side context providers for the app. Currently just the theme provider:
 * three states (system / light / dark), persisted to `localStorage` under
 * `ff:theme`, applied via a `.dark` class on <html> before first paint.
 */
export function Providers({ children }: { children: ReactNode }) {
  return (
    <ThemeProvider
      attribute="class"
      defaultTheme="system"
      enableSystem
      disableTransitionOnChange
      storageKey="ff:theme"
    >
      {children}
    </ThemeProvider>
  );
}
