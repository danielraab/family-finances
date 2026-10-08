import { Description } from "@headlessui/react";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { KeyRound } from "lucide-react";
import { type FormEvent, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { api } from "../api/client";
import { useAuth } from "../components/AuthProvider";
import { Modal } from "../components/Modal";
import {
  type Passkey,
  registerPasskey,
  signInWithPasskey,
} from "../lib/passkeys";
import { passkeysSupported } from "../lib/webauthn";

export const Route = createFileRoute("/settings/passkeys")({
  component: PasskeysTab,
});

type PasskeyProvider = NonNullable<Passkey["provider"]>;

/**
 * The passkey's provider logo — the light or dark variant to match the theme
 * (either one when only one exists) — or the generic key glyph when the
 * provider is unknown. The icons are SVG data URIs from the backend's
 * embedded AAGUID list; an <img> renders them without running any script.
 */
function ProviderIcon({ provider }: { provider: PasskeyProvider | null }) {
  const light = provider?.icon_light ?? provider?.icon_dark;
  const dark = provider?.icon_dark ?? provider?.icon_light;
  if (!light || !dark) {
    // Same 20px box as the logos, so names line up across rows.
    return (
      <span className="flex size-5 shrink-0 items-center justify-center">
        <KeyRound
          aria-hidden="true"
          className="size-4 text-zinc-500 dark:text-zinc-400"
        />
      </span>
    );
  }
  return (
    <>
      <img src={light} alt="" className="size-5 shrink-0 dark:hidden" />
      <img src={dark} alt="" className="hidden size-5 shrink-0 dark:block" />
    </>
  );
}

function formatDate(iso: string, lang: string): string {
  return new Intl.DateTimeFormat(lang, { dateStyle: "medium" }).format(
    new Date(iso),
  );
}

const BUTTON_PRIMARY =
  "inline-flex items-center gap-2 rounded-md bg-zinc-900 px-3 py-2 text-sm font-medium text-white hover:bg-zinc-700 disabled:opacity-50 dark:bg-white dark:text-zinc-900 dark:hover:bg-zinc-200";
const BUTTON_SECONDARY =
  "rounded-md border border-black/15 px-3 py-2 text-sm font-medium hover:bg-black/[.04] disabled:opacity-50 dark:border-white/15 dark:hover:bg-white/[.06]";
const BUTTON_CANCEL =
  "rounded-md px-3 py-2 text-sm font-medium text-zinc-600 hover:bg-black/[.04] dark:text-zinc-400 dark:hover:bg-white/[.06]";

/**
 * The Passkeys tab: list, add and remove the visitor's passkeys.
 *
 * Adding one needs a fresh sign-in — the backend only registers a passkey on
 * a browser session younger than its re-authentication window, so a stolen
 * session cookie cannot plant a credential. `GET /api/auth/me` reports the
 * deadline (`session.passkey_registration_until`); the tab enables "Add
 * passkey" until then, flips to the stale state when it passes (or when the
 * server answers 403 anyway), and offers "Sign in again". Removing a passkey
 * is always allowed; it ends every session signed in with that passkey,
 * including this one when it is the current session's passkey.
 */
function PasskeysTab() {
  const { user, refresh, logout, signOutLocally } = useAuth();
  const navigate = useNavigate();
  const { t, i18n } = useTranslation();
  const lang = i18n.resolvedLanguage ?? "en";

  const [passkeys, setPasskeys] = useState<Passkey[] | null>(null);
  const [now, setNow] = useState(() => Date.now());
  // Set when the server refused a registration as stale despite our clock.
  const [serverSaysStale, setServerSaysStale] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const [adding, setAdding] = useState(false);
  const [name, setName] = useState("");
  const [busy, setBusy] = useState(false);
  const [removing, setRemoving] = useState<Passkey | null>(null);

  const until = user?.session.passkey_registration_until ?? null;
  const untilMs = until ? Date.parse(until) : null;
  const fresh = !serverSaysStale && untilMs !== null && untilMs > now;
  // The window is server configuration (default 5 minutes); read it back off
  // this session rather than hardcoding it in the copy.
  const windowMinutes =
    untilMs === null
      ? 5
      : Math.max(
          1,
          Math.round(
            (untilMs - Date.parse(user?.session.created_at ?? "")) / 60000,
          ) || 5,
        );

  useEffect(() => {
    let cancelled = false;
    api.GET("/api/auth/passkeys").then(({ data }) => {
      if (!cancelled && data) setPasskeys(data);
    });
    return () => {
      cancelled = true;
    };
  }, []);

  // A new session (after re-authenticating) clears a server-reported stale
  // state, and re-arms the timer that flips the tab to stale on screen.
  useEffect(() => {
    setServerSaysStale(false);
    setNow(Date.now());
    if (untilMs === null) return;
    const remaining = untilMs - Date.now();
    if (remaining <= 0) return;
    const timer = window.setTimeout(() => setNow(Date.now()), remaining + 50);
    return () => window.clearTimeout(timer);
  }, [untilMs]);

  if (!user) {
    return null;
  }

  async function reloadPasskeys() {
    const { data } = await api.GET("/api/auth/passkeys");
    if (data) setPasskeys(data);
  }

  async function onAdd(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setBusy(true);
    setError(null);
    try {
      const result = await registerPasskey(name);
      if (result.status === "ok") {
        setPasskeys((prev) => [...(prev ?? []), result.passkey]);
        setAdding(false);
        setName("");
      } else if (result.status === "reauth") {
        setServerSaysStale(true);
        setAdding(false);
      } else if (result.status === "failed") {
        setError(t("settings.passkeys.addError"));
      }
    } catch {
      setError(t("settings.passkeys.addError"));
    } finally {
      setBusy(false);
    }
  }

  async function signInAgain() {
    setError(null);
    if (!passkeys?.length) {
      await signOutAndSignIn();
      return;
    }
    setBusy(true);
    try {
      const result = await signInWithPasskey();
      if (result === "ok") {
        await refresh();
        await reloadPasskeys();
      } else if (result === "rateLimited") {
        setError(t("login.rateLimited"));
      } else if (result !== "cancelled") {
        setError(t("settings.passkeys.reauthError"));
      }
    } catch {
      setError(t("settings.passkeys.reauthError"));
    } finally {
      setBusy(false);
    }
  }

  async function signOutAndSignIn() {
    await logout();
    navigate({ to: "/login", replace: true });
  }

  async function confirmRemove() {
    if (!removing) return;
    const target = removing;
    setRemoving(null);
    setError(null);
    const { response } = await api.DELETE("/api/auth/passkeys/{id}", {
      params: { path: { id: target.id } },
    });
    if (response.status === 204) {
      if (target.current) {
        // The server ended this session together with the passkey.
        signOutLocally();
        navigate({ to: "/login", replace: true });
        return;
      }
      setPasskeys((prev) => prev?.filter((p) => p.id !== target.id) ?? null);
    } else {
      setError(t("settings.passkeys.removeError"));
    }
  }

  return (
    <div className="flex flex-col gap-6">
      <div className="flex flex-col gap-3">
        <h2 className="text-sm font-semibold text-zinc-500 dark:text-zinc-400">
          {t("settings.passkeys.heading")}
        </h2>
        <p className="text-sm text-zinc-600 dark:text-zinc-400">
          {t("settings.passkeys.intro")}
        </p>

        {passkeys === null ? null : passkeys.length === 0 ? (
          <p className="text-sm text-zinc-500 dark:text-zinc-400">
            {t("settings.passkeys.empty")}
          </p>
        ) : (
          <ul className="flex flex-col gap-2">
            {passkeys.map((p) => (
              <li
                key={p.id}
                className="flex items-center justify-between gap-3 rounded-md border border-black/10 px-3 py-2 text-sm dark:border-white/10"
              >
                <span className="flex items-center gap-3">
                  <ProviderIcon provider={p.provider} />
                  <span className="flex flex-col gap-0.5 leading-tight">
                    <span className="flex flex-wrap items-center gap-2 font-medium">
                      {p.name}
                      {p.current && (
                        <span className="rounded-full bg-black/[.06] px-2 py-0.5 text-xs font-normal text-zinc-600 dark:bg-white/[.08] dark:text-zinc-300">
                          {t("settings.passkeys.thisSession")}
                        </span>
                      )}
                    </span>
                    <span className="text-xs text-zinc-500 dark:text-zinc-400">
                      {p.provider && p.provider.name !== p.name && (
                        <>
                          {p.provider.name}
                          {" · "}
                        </>
                      )}
                      {t("settings.passkeys.added", {
                        date: formatDate(p.created_at, lang),
                      })}
                      {" · "}
                      {p.last_used_at
                        ? t("settings.passkeys.lastUsed", {
                            date: formatDate(p.last_used_at, lang),
                          })
                        : t("settings.passkeys.neverUsed")}
                    </span>
                  </span>
                </span>
                <button
                  type="button"
                  onClick={() => setRemoving(p)}
                  className="shrink-0 text-xs font-medium text-red-600 underline underline-offset-2 hover:text-red-800 dark:text-red-400 dark:hover:text-red-300"
                >
                  {t("settings.passkeys.remove")}
                </button>
              </li>
            ))}
          </ul>
        )}
      </div>

      <div className="flex flex-col gap-3 rounded-lg border border-black/10 p-4 dark:border-white/10">
        <p className="text-sm text-zinc-600 dark:text-zinc-400">
          {t("settings.passkeys.freshNotice", { minutes: windowMinutes })}
        </p>
        {!passkeysSupported() && (
          <p className="text-sm text-zinc-500 dark:text-zinc-400">
            {t("settings.passkeys.unsupported")}
          </p>
        )}
        <div className="flex flex-wrap items-center gap-2">
          <button
            type="button"
            disabled={!fresh || busy || !passkeysSupported()}
            onClick={() => {
              setError(null);
              setAdding(true);
            }}
            className={BUTTON_PRIMARY}
          >
            <KeyRound aria-hidden="true" className="size-4" />
            {t("settings.passkeys.add")}
          </button>
          {!fresh && (
            <>
              <button
                type="button"
                disabled={busy}
                onClick={signInAgain}
                className={BUTTON_SECONDARY}
              >
                {t("settings.passkeys.signInAgain")}
              </button>
              {(passkeys?.length ?? 0) > 0 && (
                <button
                  type="button"
                  disabled={busy}
                  onClick={signOutAndSignIn}
                  className="text-sm text-zinc-600 underline underline-offset-2 hover:text-zinc-900 dark:text-zinc-400 dark:hover:text-zinc-100"
                >
                  {t("settings.passkeys.signInWithOther")}
                </button>
              )}
            </>
          )}
        </div>
        {!fresh && (
          <p className="text-xs text-zinc-500 dark:text-zinc-400">
            {t("settings.passkeys.staleHint", { minutes: windowMinutes })}
          </p>
        )}
      </div>

      {error && (
        <p className="text-sm text-red-600 dark:text-red-400">{error}</p>
      )}

      <Modal
        open={adding}
        onClose={() => setAdding(false)}
        dismissable={!busy}
        title={t("settings.passkeys.addTitle")}
      >
        <form onSubmit={onAdd} className="flex flex-col gap-4">
          <label className="flex flex-col gap-1.5 text-sm font-medium">
            {t("settings.passkeys.nameLabel")}
            <input
              value={name}
              maxLength={100}
              placeholder={t("settings.passkeys.namePlaceholder")}
              onChange={(event) => setName(event.target.value)}
              className="rounded-md border border-black/15 bg-transparent px-3 py-2 text-sm font-normal outline-none transition-colors focus:border-black/40 dark:border-white/15 dark:focus:border-white/40"
            />
          </label>
          {error && (
            <p className="text-sm text-red-600 dark:text-red-400">{error}</p>
          )}
          <div className="flex justify-end gap-2">
            <button
              type="button"
              disabled={busy}
              onClick={() => setAdding(false)}
              className={BUTTON_CANCEL}
            >
              {t("settings.users.confirm.cancel")}
            </button>
            <button type="submit" disabled={busy} className={BUTTON_PRIMARY}>
              {busy
                ? t("settings.passkeys.adding")
                : t("settings.passkeys.addConfirm")}
            </button>
          </div>
        </form>
      </Modal>

      <Modal
        open={removing !== null}
        onClose={() => setRemoving(null)}
        title={
          removing
            ? t("settings.passkeys.confirmRemoveTitle", { name: removing.name })
            : ""
        }
      >
        {removing && (
          <>
            <Description className="text-sm text-zinc-600 dark:text-zinc-400">
              {removing.current
                ? t("settings.passkeys.confirmRemoveCurrent")
                : t("settings.passkeys.confirmRemoveBody")}
            </Description>
            <div className="flex justify-end gap-2">
              <button
                type="button"
                onClick={() => setRemoving(null)}
                className={BUTTON_CANCEL}
              >
                {t("settings.users.confirm.cancel")}
              </button>
              <button
                type="button"
                onClick={confirmRemove}
                className="rounded-md bg-red-600 px-3 py-2 text-sm font-medium text-white hover:bg-red-700"
              >
                {t("settings.passkeys.remove")}
              </button>
            </div>
          </>
        )}
      </Modal>
    </div>
  );
}
