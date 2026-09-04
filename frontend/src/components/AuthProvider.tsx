import {
  createContext,
  type ReactNode,
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
  /** Revoke the session and drop to the anonymous state, no page reload. */
  logout: () => Promise<void>;
};

const AuthContext = createContext<AuthContextValue | null>(null);

/**
 * Resolves the visitor's auth state once, on mount, by calling
 * `GET /api/auth/me` — the only way to know, since the build is static and the
 * `ff_session` cookie is `HttpOnly`. `200` → authenticated; `401`, a non-ok
 * status, or a network error → anonymous. No polling, no focus-refetch: signing
 * in is a full page load (the magic-link callback redirects here), so the only
 * transition this provider drives itself is `logout()`.
 *
 * When the resolved user carries a non-null `language`, it is applied via
 * `i18n.changeLanguage` — an explicit account preference takes priority over
 * the browser-detected language (see web-client-i18n). A visitor with no
 * preference set keeps whatever the browser detector already resolved.
 */
export function AuthProvider({ children }: { children: ReactNode }) {
  const [status, setStatus] = useState<AuthStatus>("loading");
  const [user, setUser] = useState<User | null>(null);

  useEffect(() => {
    let cancelled = false;

    api
      .GET("/api/auth/me")
      .then(({ data }) => {
        if (cancelled) return;
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
      })
      .catch(() => {
        if (cancelled) return;
        setUser(null);
        setStatus("anonymous");
      });

    return () => {
      cancelled = true;
    };
  }, []);

  async function logout() {
    try {
      await api.POST("/api/auth/logout");
    } finally {
      setUser(null);
      setStatus("anonymous");
    }
  }

  return <AuthContext value={{ status, user, logout }}>{children}</AuthContext>;
}

/** Read the auth context. Throws if used outside `<AuthProvider>`. */
export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) {
    throw new Error("useAuth must be used within <AuthProvider>");
  }
  return ctx;
}
