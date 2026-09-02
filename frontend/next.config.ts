import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  // Emit a fully static site to `out/` — no Node server in production.
  output: "export",
  // Dev only: `output: "export"` ignores rewrites at build time (Next warns),
  // so this just proxies the browser's /api calls to the Go backend under
  // `next dev`, where the two toolchains run on separate ports.
  async rewrites() {
    return [
      {
        source: "/api/:path*",
        destination: "http://localhost:8080/api/:path*",
      },
    ];
  },
};

export default nextConfig;
