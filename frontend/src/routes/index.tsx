import { createFileRoute } from "@tanstack/react-router";
import { FeatureOverview } from "../components/FeatureOverview";

export const Route = createFileRoute("/")({
  component: Home,
});

function Home() {
  return (
    <>
      <FeatureOverview />
    </>
  );
}
