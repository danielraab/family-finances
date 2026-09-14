import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { useEffect } from "react";
import { useAuth } from "../components/AuthProvider";
import { FeatureOverview } from "../components/FeatureOverview";

export const Route = createFileRoute("/")({
  component: Home,
});

/**
 * The marketing landing page for anonymous visitors. An authenticated
 * visitor is redirected to /home, mirroring /home's own redirect-when-
 * anonymous in reverse.
 */
function Home() {
  const { status } = useAuth();
  const navigate = useNavigate();

  useEffect(() => {
    if (status === "authenticated") {
      navigate({ to: "/home", replace: true });
    }
  }, [status, navigate]);

  if (status === "authenticated") {
    return null;
  }

  return <FeatureOverview />;
}
