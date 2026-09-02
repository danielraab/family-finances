import {
  createContext,
  type ReactNode,
  useContext,
  useEffect,
  useState,
} from "react";

/** The authenticated user, as returned by `GET /api/auth/me`. */
export type User = {
  id: string;
  email: string;
  display_name?: string;
  is_admin: boolean;
};

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
 */
export function AuthProvider({ children }: { children: ReactNode }) {
  const [status, setStatus] = useState<AuthStatus>("loading");
  const [user, setUser] = useState<User | null>(null);

  useEffect(() => {
    let cancelled = false;

    fetch("/api/auth/me", { credentials: "same-origin" })
      .then(async (res) => {
        if (cancelled) return;
        if (res.ok) {
          setUser((await res.json()) as User);
          setStatus("authenticated");
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
      await fetch("/api/auth/logout", {
        method: "POST",
        credentials: "same-origin",
      });
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
