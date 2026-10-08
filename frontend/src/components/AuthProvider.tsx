import {
  createContext,
  type ReactNode,
  useCallback,
  useContext,
  useEffect,
  useState,
} from "react";
import { api } from "../api/client";
import type { components } from "../api/schema";
import i18n from "../i18n";

/** The authenticated user, as returned by `GET /api/auth/me`. */
export type User = components["schemas"]["User"];

export type AuthStatus = "loading" | "anonymous" | "authenticated";

type AuthContextValue = {
  status: AuthStatus;
  user: User | null;
  /**
   * Replace the cached user record — e.g. after the profile form saves a new
   * display name via `PATCH /api/auth/me`, so the sidebar updates without a
   * reload or a refetch.
   */
  setUser: (user: User) => void;
  /** Revoke the session and drop to the anonymous state, no page reload. */
  logout: () => Promise<void>;
  /**
   * Re-resolve `GET /api/auth/me` — after a sign-in that happened in place
   * (a passkey ceremony sets the session cookie via `fetch`, no page load),
   * or to pick up the new session's freshness after re-authenticating.
   */
  refresh: () => Promise<void>;
  /**
   * Drop to the anonymous state without calling the backend — for when the
   * session is already gone server-side (e.g. its passkey was just deleted).
   */
  signOutLocally: () => void;
};

const AuthContext = createContext<AuthContextValue | null>(null);

/**
 * Resolves the visitor's auth state once, on mount, by calling
 * `GET /api/auth/me` — the only way to know, since the build is static and the
 * `ff_session` cookie is `HttpOnly`. `200` → authenticated; `401`, a non-ok
 * status, or a network error → anonymous. No polling, no focus-refetch: a
 * magic-link or OIDC sign-in is a full page load (the callback redirects
 * here); a passkey sign-in happens in place and calls `refresh()`.
 *
 * When the resolved user carries a non-null `language`, it is applied via
 * `i18n.changeLanguage` — an explicit account preference takes priority over
 * the browser-detected language (see web-client-i18n). A visitor with no
 * preference set keeps whatever the browser detector already resolved.
 */
export function AuthProvider({ children }: { children: ReactNode }) {
  const [status, setStatus] = useState<AuthStatus>("loading");
  const [user, setUser] = useState<User | null>(null);

  const applyMe = useCallback((data: User | null) => {
    if (data) {
      setUser(data);
      setStatus("authenticated");
      if (data.language) {
        i18n.changeLanguage(data.language);
      }
    } else {
      setUser(null);
      setStatus("anonymous");
    }
  }, []);

  useEffect(() => {
    let cancelled = false;
    resolveMe().then((data) => {
      if (!cancelled) applyMe(data);
    });
    return () => {
      cancelled = true;
    };
  }, [applyMe]);

  async function refresh() {
    applyMe(await resolveMe());
  }

  function signOutLocally() {
    setUser(null);
    setStatus("anonymous");
  }

  async function logout() {
    try {
      await api.POST("/api/auth/logout");
    } finally {
      setUser(null);
      setStatus("anonymous");
    }
  }

  return (
    <AuthContext
      value={{ status, user, setUser, logout, refresh, signOutLocally }}
    >
      {children}
    </AuthContext>
  );
}

/** `GET /api/auth/me`: the user, or null when anonymous or unreachable. */
async function resolveMe(): Promise<User | null> {
  try {
    const { data } = await api.GET("/api/auth/me");
    return data ?? null;
  } catch {
    return null;
  }
}

/** Read the auth context. Throws if used outside `<AuthProvider>`. */
export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) {
    throw new Error("useAuth must be used within <AuthProvider>");
  }
  return ctx;
}
