"use client";

import { useEffect, useState } from "react";

/**
 * Temporary: proves the browser can reach the Go backend over a relative
 * `/api/...` path. Remove once the frontend/backend wiring is confirmed.
 */
export function HealthCheck() {
  const [result, setResult] = useState("loading…");

  useEffect(() => {
    fetch("/api/healthz")
      .then((res) => res.text())
      .then((text) => setResult(text))
      .catch((err) => setResult(`error: ${err}`));
  }, []);

  return (
    <p className="px-6 py-4 font-mono text-sm sm:px-10">
      Backend <code>/api/healthz</code>: {result}
    </p>
  );
}
