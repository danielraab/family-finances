import { createFileRoute, Outlet, useNavigate } from "@tanstack/react-router";
import { useEffect } from "react";
import { useAuth } from "../components/AuthProvider";

export const Route = createFileRoute("/recurring")({
  component: RecurringLayout,
});

/**
 * The recurring-transactions section's auth gate, same redirect-to-/login
 * pattern as /entries and /accounts.
 */
function RecurringLayout() {
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
