import { api } from "../api/client";
import type { components } from "../api/schema";
import { createPasskey, getPasskey } from "./webauthn";

export type Passkey = components["schemas"]["Passkey"];

/**
 * Outcome of a passkey sign-in: `ok` (a session cookie is now set — refresh
 * the auth state), `cancelled` (the visitor dismissed the browser prompt;
 * show nothing), `rateLimited` (429), `disabled` (the account is disabled),
 * or `failed` (any other rejection).
 */
export type PasskeySignInResult =
  | "ok"
  | "cancelled"
  | "rateLimited"
  | "disabled"
  | "failed";

/** Run the discoverable sign-in ceremony: start → browser prompt → finish. */
export async function signInWithPasskey(): Promise<PasskeySignInResult> {
  const start = await api.POST("/api/auth/passkeys/login/start");
  if (start.response.status === 429) return "rateLimited";
  if (!start.data) return "failed";

  const prompt = await getPasskey(start.data.options);
  if (prompt.status === "cancelled") return "cancelled";

  const finish = await api.POST("/api/auth/passkeys/login/finish", {
    body: {
      ceremony_id: start.data.ceremony_id,
      credential: prompt.credential,
    },
  });
  if (finish.data) return "ok";
  if (finish.response.status === 429) return "rateLimited";
  if (finish.response.status === 403) return "disabled";
  return "failed";
}

/**
 * Outcome of registering a passkey: `ok` with the new passkey, `cancelled`,
 * `reauth` (403 — the session is too old or not a browser session; a fresh
 * sign-in is needed), or `failed`.
 */
export type RegisterPasskeyResult =
  | { status: "ok"; passkey: Passkey }
  | { status: "cancelled" }
  | { status: "reauth" }
  | { status: "failed" };

/** Run the registration ceremony on the current (fresh) session. */
export async function registerPasskey(
  name: string,
): Promise<RegisterPasskeyResult> {
  const start = await api.POST("/api/auth/passkeys/register/start");
  if (start.response.status === 403) return { status: "reauth" };
  if (!start.data) return { status: "failed" };

  const prompt = await createPasskey(start.data.options);
  if (prompt.status === "cancelled") return { status: "cancelled" };

  const finish = await api.POST("/api/auth/passkeys/register/finish", {
    body: {
      ceremony_id: start.data.ceremony_id,
      credential: prompt.credential,
      ...(name.trim() ? { name: name.trim() } : {}),
    },
  });
  if (finish.data) return { status: "ok", passkey: finish.data };
  if (finish.response.status === 403) return { status: "reauth" };
  return { status: "failed" };
}
