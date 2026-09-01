import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  // Emit a fully static site to `out/` — no Node server in production.
  output: "export",
};

export default nextConfig;
