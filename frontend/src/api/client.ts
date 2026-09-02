import createClient from "openapi-fetch";
import type { paths } from "./schema";

/**
 * The typed backend client. The app ships no backend URL: every call is a
 * relative `/api/...` path, same-origin, with credentials so the `ff_session`
 * cookie is sent. In dev the Vite proxy forwards `/api` to the Go backend; in
 * production the Go binary serves both.
 *
 * Types come from `./schema` (generated from `openapi/openapi.yaml` — run
 * `pnpm generate:api`). Browser-redirect endpoints are deliberately absent from
 * the generated `paths`, so they cannot be called through here.
 */
export const api = createClient<paths>({
  baseUrl: "/",
  credentials: "same-origin",
});
