import { createRootRoute, Outlet, useLocation } from "@tanstack/react-router";
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
  const [mobileOpen, setMobileOpen] = useState(false);
  const [mounted, setMounted] = useState(false);
  const pathname = useLocation({ select: (l) => l.pathname });

  useEffect(() => {
    setMounted(true);
  }, []);

  // Any navigation dismisses the mobile drawer, including links that don't
  // go through Sidebar's own onClick (e.g. the login link).
  // biome-ignore lint/correctness/useExhaustiveDependencies: pathname is a trigger, not read in the effect body.
  useEffect(() => {
    setMobileOpen(false);
  }, [pathname]);

  useEffect(() => {
    if (!mobileOpen) {
      return;
    }
    const { style } = document.body;
    const previousOverflow = style.overflow;
    style.overflow = "hidden";
    return () => {
      style.overflow = previousOverflow;
    };
  }, [mobileOpen]);

  function toggleSidebar() {
    if (window.matchMedia("(min-width: 768px)").matches) {
      setCollapsed((prev) => {
        const next = !prev;
        try {
          window.localStorage.setItem(SIDEBAR_STORAGE_KEY, String(next));
        } catch {
          // ignore persistence failure
        }
        return next;
      });
    } else {
      setMobileOpen((prev) => !prev);
    }
  }

  return (
    <ThemeProvider>
      <AuthProvider>
        <div className="flex min-h-screen">
          <Sidebar
            collapsed={collapsed}
            mounted={mounted}
            mobileOpen={mobileOpen}
            onCloseMobile={() => setMobileOpen(false)}
          />
          <div className="flex flex-1 flex-col overflow-x-hidden">
            <TopBar
              collapsed={collapsed}
              mobileOpen={mobileOpen}
              onToggle={toggleSidebar}
            />
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
