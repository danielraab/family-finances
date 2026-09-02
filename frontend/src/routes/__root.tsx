import { createRootRoute, Outlet } from "@tanstack/react-router";
import { TanStackRouterDevtools } from "@tanstack/react-router-devtools";
import { AuthProvider } from "../components/AuthProvider";
import { Sidebar } from "../components/Sidebar";
import { ThemeProvider } from "../lib/theme";
import "../styles.css";

export const Route = createRootRoute({
  component: RootComponent,
});

function RootComponent() {
  return (
    <ThemeProvider>
      <AuthProvider>
        <div className="flex min-h-screen">
          <Sidebar />
          <main className="flex-1 overflow-x-hidden">
            <Outlet />
          </main>
        </div>
        {import.meta.env.DEV ? <TanStackRouterDevtools /> : null}
      </AuthProvider>
    </ThemeProvider>
  );
}
