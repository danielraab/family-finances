import { createFileRoute } from "@tanstack/react-router";
import { Placeholder } from "../components/Placeholder";

export const Route = createFileRoute("/")({
  component: Home,
});

function Home() {
  return (
    <>
      <Placeholder />
    </>
  );
}
