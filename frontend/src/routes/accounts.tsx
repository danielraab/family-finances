import { createFileRoute, Outlet, useNavigate } from "@tanstack/react-router";
import { useEffect } from "react";
import { useAuth } from "../components/AuthProvider";

export const Route = createFileRoute("/accounts")({
  component: AccountsLayout,
});

/**
 * The accounts section's auth gate, same redirect-to-/login pattern as
 * /settings. Renders nothing while useAuth is loading or the redirect is
 * pending, so there's no flash of a page before the decision is made.
 */
function AccountsLayout() {
  const { status } = useAuth();
  const navigate = useNavigate();

  useEffect(() => {
    if (status === "anonymous") {
      navigate({ to: "/login", replace: true });
    }
  }, [status, navigate]);

  if (status !== "authenticated") {
    return null;
  }

  return <Outlet />;
}
