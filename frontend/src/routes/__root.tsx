import { createRootRoute, Outlet } from "@tanstack/react-router";
import { TanStackRouterDevtools } from "@tanstack/react-router-devtools";
import { useEffect, useState } from "react";
import { AuthProvider } from "../components/AuthProvider";
import { Sidebar } from "../components/Sidebar";
import { TopBar } from "../components/TopBar";
import { ThemeProvider } from "../lib/theme";
import "../styles.css";

const SIDEBAR_STORAGE_KEY = "ff:sidebar-collapsed";

function readCollapsed(): boolean {
  try {
    return window.localStorage.getItem(SIDEBAR_STORAGE_KEY) === "true";
  } catch {
    return false;
  }
}

export const Route = createRootRoute({
  component: RootComponent,
});

function RootComponent() {
  const [collapsed, setCollapsed] = useState(readCollapsed);
  const [mounted, setMounted] = useState(false);

  useEffect(() => {
    setMounted(true);
  }, []);

  function toggleCollapsed() {
    setCollapsed((prev) => {
      const next = !prev;
      try {
        window.localStorage.setItem(SIDEBAR_STORAGE_KEY, String(next));
      } catch {
        // ignore persistence failure
      }
      return next;
    });
  }

  return (
    <ThemeProvider>
      <AuthProvider>
        <div className="flex min-h-screen">
          <Sidebar collapsed={collapsed} mounted={mounted} />
          <div className="flex flex-1 flex-col overflow-x-hidden">
            <TopBar collapsed={collapsed} onToggle={toggleCollapsed} />
            <main className="flex-1">
              <Outlet />
            </main>
          </div>
        </div>
        {import.meta.env.DEV ? (
          <TanStackRouterDevtools position="bottom-right" />
        ) : null}
      </AuthProvider>
    </ThemeProvider>
  );
}
