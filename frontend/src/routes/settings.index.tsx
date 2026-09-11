import { createFileRoute } from "@tanstack/react-router";
import { type ReactNode, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { api } from "../api/client";
import type { components } from "../api/schema";
import { useAuth } from "../components/AuthProvider";
import { CurrencySelect } from "../components/CurrencySelect";
import i18n from "../i18n";

export const Route = createFileRoute("/settings/")({
  component: ProfileSettingsTab,
});

type UserSettings = components["schemas"]["UserSettings"];
type Language = UserSettings["language"];

const LANGUAGES: Language[] = ["en", "de"];

/** Feature-detects Intl.supportedValuesOf, absent from older engines. */
function listTimezones(): string[] {
  const supportedValuesOf = (
    Intl as unknown as { supportedValuesOf?: (key: string) => string[] }
  ).supportedValuesOf;
  if (!supportedValuesOf) {
    return [];
  }
  try {
    return supportedValuesOf("timeZone");
  } catch {
    return [];
  }
}

const inputClass =
  "rounded-md border border-black/15 bg-transparent px-3 py-2 text-sm font-normal outline-none transition-colors focus:border-black/40 dark:border-white/15 dark:focus:border-white/40";

function SettingField({
  id,
  label,
  error,
  children,
}: {
  id: string;
  label: string;
  error: string | null;
  children: ReactNode;
}) {
  return (
    <div className="flex flex-col gap-1.5 text-sm font-medium">
      <label htmlFor={id}>{label}</label>
      {children}
      {error && (
        <span className="text-xs font-normal text-red-600 dark:text-red-400">
          {error}
        </span>
      )}
    </div>
  );
}

/**
 * The Profile tab: full name, then display language, timezone, default
 * currency and displayed decimal places. Each field saves immediately (the
 * name on blur via PATCH /api/auth/me, the rest on change via PUT /api/settings
 * with only that field), no separate save button — mirrors ThemeSwitch's
 * click-applies-immediately interaction. A failed save reverts the field to
 * its last-known-good value.
 */
function ProfileSettingsTab() {
  const { t } = useTranslation();
  const { user, setUser } = useAuth();
  const [settings, setSettings] = useState<UserSettings | null>(null);
  const [timezones] = useState(listTimezones);
  const [errorField, setErrorField] = useState<keyof UserSettings | null>(null);
  const savedName = user?.display_name ?? "";
  const [nameDraft, setNameDraft] = useState(savedName);
  const [nameError, setNameError] = useState(false);

  // Keep the field in step with the shared user record (initial resolve, or a
  // successful save pushing a new value back through the context).
  useEffect(() => {
    setNameDraft(savedName);
  }, [savedName]);

  async function saveName() {
    const next = nameDraft.trim();
    if (next === savedName) return;
    setNameError(false);
    const { data, response } = await api.PATCH("/api/auth/me", {
      body: { display_name: next },
    });
    if (!response.ok || !data) {
      setNameDraft(savedName);
      setNameError(true);
      return;
    }
    setUser(data);
  }

  useEffect(() => {
    let cancelled = false;
    api.GET("/api/settings").then(({ data }) => {
      if (cancelled || !data) return;
      setSettings(data);
    });
    return () => {
      cancelled = true;
    };
  }, []);

  async function update(patch: Partial<UserSettings>) {
    if (!settings) return;
    const field = Object.keys(patch)[0] as keyof UserSettings;
    const previous = settings;
    setSettings({ ...settings, ...patch });
    setErrorField(null);

    const { data, response } = await api.PUT("/api/settings", {
      body: patch,
    });
    if (!response.ok || !data) {
      setSettings(previous);
      setErrorField(field);
      return;
    }
    setSettings(data);
    if (patch.language) {
      i18n.changeLanguage(patch.language);
    }
  }

  if (!settings) {
    return null;
  }

  return (
    <div className="flex flex-col gap-6">
      <SettingField
        id="settings-name"
        label={t("settings.profile.name")}
        error={nameError ? t("settings.profile.nameError") : null}
      >
        <input
          id="settings-name"
          value={nameDraft}
          maxLength={150}
          onChange={(event) => setNameDraft(event.target.value)}
          onBlur={() => {
            void saveName();
          }}
          className={inputClass}
        />
      </SettingField>

      <SettingField
        id="settings-language"
        label={t("settings.profile.language")}
        error={
          errorField === "language" ? t("settings.profile.saveError") : null
        }
      >
        <select
          id="settings-language"
          value={settings.language}
          onChange={(event) =>
            update({ language: event.target.value as NonNullable<Language> })
          }
          className={inputClass}
        >
          {LANGUAGES.map((lang) => (
            <option key={lang} value={lang}>
              {t(`settings.profile.languageOption.${lang}`)}
            </option>
          ))}
        </select>
      </SettingField>

      <SettingField
        id="settings-timezone"
        label={t("settings.profile.timezone")}
        error={
          errorField === "timezone" ? t("settings.profile.saveError") : null
        }
      >
        <select
          id="settings-timezone"
          value={settings.timezone}
          onChange={(event) => update({ timezone: event.target.value })}
          className={inputClass}
        >
          {!timezones.includes(settings.timezone) && (
            <option value={settings.timezone}>{settings.timezone}</option>
          )}
          {timezones.map((tz) => (
            <option key={tz} value={tz}>
              {tz}
            </option>
          ))}
        </select>
      </SettingField>

      <SettingField
        id="settings-default-currency"
        label={t("settings.profile.defaultCurrency")}
        error={
          errorField === "default_currency"
            ? t("settings.profile.saveError")
            : null
        }
      >
        <CurrencySelect
          id="settings-default-currency"
          value={settings.default_currency}
          onChange={(value) => update({ default_currency: value })}
          className={`${inputClass} w-24`}
        />
      </SettingField>

      <SettingField
        id="settings-displayed-decimal-places"
        label={t("settings.profile.displayedDecimalPlaces")}
        error={
          errorField === "displayed_decimal_places"
            ? t("settings.profile.saveError")
            : null
        }
      >
        <select
          id="settings-displayed-decimal-places"
          value={settings.displayed_decimal_places}
          onChange={(event) =>
            update({ displayed_decimal_places: Number(event.target.value) })
          }
          className={`${inputClass} w-24`}
        >
          {[0, 1, 2, 3, 4].map((n) => (
            <option key={n} value={n}>
              {n}
            </option>
          ))}
        </select>
      </SettingField>
    </div>
  );
}
