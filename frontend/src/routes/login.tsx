import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { type FormEvent, useEffect, useState } from "react";
import { useAuth } from "../components/AuthProvider";

export const Route = createFileRoute("/login")({
  component: LoginPage,
});

const EMAIL_RE = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

function LoginPage() {
  const { status } = useAuth();
  const navigate = useNavigate();

  const [email, setEmail] = useState("");
  const [phase, setPhase] = useState<"form" | "sent">("form");
  const [sentTo, setSentTo] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    if (status === "authenticated") {
      navigate({ to: "/", replace: true });
    }
  }, [status, navigate]);

  // Already signed in — the effect above is navigating away.
  if (status === "authenticated") {
    return null;
  }

  async function onSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const value = email.trim();

    if (!EMAIL_RE.test(value)) {
      setError("Enter a valid email address.");
      return;
    }
    setError(null);
    setSubmitting(true);
    try {
      const res = await fetch("/api/auth/email/start", {
        method: "POST",
        credentials: "same-origin",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ email: value }),
      });
      if (!res.ok) {
        throw new Error(`unexpected status ${res.status}`);
      }
      setSentTo(value);
      setPhase("sent");
    } catch {
      setError("Something went wrong sending the link. Please try again.");
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <section className="mx-auto flex w-full max-w-md flex-col gap-6 px-6 py-12 sm:px-10">
      <header className="flex flex-col gap-2">
        <h1 className="text-2xl font-semibold tracking-tight">Sign in</h1>
        <p className="text-zinc-600 dark:text-zinc-400">
          We&rsquo;ll email you a single-use link to sign in — no password.
        </p>
      </header>

      {phase === "form" ? (
        <form
          onSubmit={onSubmit}
          className="flex flex-col gap-4 rounded-lg border border-black/15 p-6 dark:border-white/15"
        >
          <label className="flex flex-col gap-1.5 text-sm font-medium">
            Email
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
            {submitting ? "Sending…" : "Send me a sign-in link"}
          </button>
        </form>
      ) : (
        <div className="flex flex-col gap-3 rounded-lg border border-black/15 p-6 dark:border-white/15">
          <p className="font-medium">Check your inbox</p>
          <p className="text-sm text-zinc-600 dark:text-zinc-400">
            We sent a sign-in link to{" "}
            <span className="font-medium text-zinc-900 dark:text-zinc-100">
              {sentTo}
            </span>
            . Open it on this device to finish signing in.
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
            Use a different address
          </button>
        </div>
      )}
    </section>
  );
}
