import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { type FormEvent, useEffect, useState } from "react";
import { Trans, useTranslation } from "react-i18next";
import { api } from "../api/client";
import type { components } from "../api/schema";
import { useAuth } from "../components/AuthProvider";

export const Route = createFileRoute("/login")({
  component: LoginPage,
});

const EMAIL_RE = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

type OidcLogin = components["schemas"]["OidcLogin"];

function LoginPage() {
  const { status } = useAuth();
  const navigate = useNavigate();
  const { t } = useTranslation();

  const [email, setEmail] = useState("");
  const [phase, setPhase] = useState<"form" | "sent">("form");
  const [sentTo, setSentTo] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);
  // OIDC sign-in affordance, or null when the backend offers none / the
  // request is still in flight or failed. The email form never waits on it.
  const [oidc, setOidc] = useState<OidcLogin | null>(null);

  useEffect(() => {
    if (status === "authenticated") {
      navigate({ to: "/", replace: true });
    }
  }, [status, navigate]);

  useEffect(() => {
    let cancelled = false;
    api
      .GET("/api/auth/config")
      .then(({ data }) => {
        if (!cancelled && data?.oidc) {
          setOidc(data.oidc);
        }
      })
      .catch(() => {
        /* no OIDC affordance — the email form stands on its own */
      });
    return () => {
      cancelled = true;
    };
  }, []);

  // Already signed in — the effect above is navigating away.
  if (status === "authenticated") {
    return null;
  }

  async function onSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const value = email.trim();

    if (!EMAIL_RE.test(value)) {
      setError(t("login.invalidEmail"));
      return;
    }
    setError(null);
    setSubmitting(true);
    try {
      const { response } = await api.POST("/api/auth/email/start", {
        body: { email: value },
      });
      if (!response.ok) {
        throw new Error(`unexpected status ${response.status}`);
      }
      setSentTo(value);
      setPhase("sent");
    } catch {
      setError(t("login.sendError"));
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <section className="mx-auto flex w-full max-w-md flex-col gap-6 px-6 py-12 sm:px-10">
      <header className="flex flex-col gap-2">
        <h1 className="text-2xl font-semibold tracking-tight">
          {t("login.title")}
        </h1>
        <p className="text-zinc-600 dark:text-zinc-400">
          {t("login.subtitle")}
        </p>
      </header>

      {phase === "form" ? (
        <>
          {oidc && (
            <div className="flex flex-col gap-4">
              <a
                href={oidc.start_path}
                className="flex items-center justify-center rounded-md border border-black/15 px-3 py-2 text-sm font-medium transition-colors hover:bg-black/[.04] dark:border-white/15 dark:hover:bg-white/[.06]"
              >
                {oidc.label}
              </a>
              <div className="flex items-center gap-3 text-xs text-zinc-500 dark:text-zinc-400">
                <span className="h-px flex-1 bg-black/10 dark:bg-white/10" />
                {t("login.or")}
                <span className="h-px flex-1 bg-black/10 dark:bg-white/10" />
              </div>
            </div>
          )}

          <form
            onSubmit={onSubmit}
            className="flex flex-col gap-4 rounded-lg border border-black/15 p-6 dark:border-white/15"
          >
            <label className="flex flex-col gap-1.5 text-sm font-medium">
              {t("login.emailLabel")}
              <input
                type="email"
                name="email"
                autoComplete="email"
                required
                value={email}
                onChange={(event) => setEmail(event.target.value)}
                className="rounded-md border border-black/15 bg-transparent px-3 py-2 text-sm font-normal outline-none transition-colors focus:border-black/40 dark:border-white/15 dark:focus:border-white/40"
              />
            </label>

            {error && (
              <p className="text-sm text-red-600 dark:text-red-400">{error}</p>
            )}

            <button
              type="submit"
              disabled={submitting}
              className="rounded-md bg-zinc-900 px-3 py-2 text-sm font-medium text-white transition-colors hover:bg-zinc-700 disabled:opacity-60 dark:bg-white dark:text-zinc-900 dark:hover:bg-zinc-200"
            >
              {submitting ? t("login.submitting") : t("login.submit")}
            </button>
          </form>
        </>
      ) : (
        <div className="flex flex-col gap-3 rounded-lg border border-black/15 p-6 dark:border-white/15">
          <p className="font-medium">{t("login.checkInbox")}</p>
          <p className="text-sm text-zinc-600 dark:text-zinc-400">
            <Trans
              i18nKey="login.sentTo"
              values={{ email: sentTo }}
              components={{
                bold: (
                  <span className="font-medium text-zinc-900 dark:text-zinc-100" />
                ),
              }}
            />
          </p>
          <button
            type="button"
            onClick={() => {
              setPhase("form");
              setEmail("");
              setError(null);
            }}
            className="self-start text-sm text-zinc-600 underline underline-offset-2 hover:text-zinc-900 dark:text-zinc-400 dark:hover:text-zinc-100"
          >
            {t("login.useDifferentAddress")}
          </button>
        </div>
      )}
    </section>
  );
}
