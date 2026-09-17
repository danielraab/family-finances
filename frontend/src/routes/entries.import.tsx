import { createFileRoute, Link } from "@tanstack/react-router";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { api } from "../api/client";
import type { components } from "../api/schema";
import { ImportAccountStep } from "../components/import/ImportAccountStep";
import { ImportDryRunStep } from "../components/import/ImportDryRunStep";
import { ImportFileStep } from "../components/import/ImportFileStep";
import { ImportMappingStep } from "../components/import/ImportMappingStep";
import { ImportResultStep } from "../components/import/ImportResultStep";
import { ImportRunStep } from "../components/import/ImportRunStep";
import {
  initialMappingState,
  type MappingState,
  type RunFailure,
  toRowMapping,
} from "../components/import/types";
import type { ClassifiedRow } from "../lib/import/mapRow";
import type { ParsedFile } from "../lib/import/parseFile";

type Account = components["schemas"]["Account"];
type Category = components["schemas"]["Category"];
type Tag = components["schemas"]["Tag"];

// The steps a visitor can reach via forward/back navigation — each one is a
// URL search param so the browser's own Back button steps back through them
// (see the module-level comment above ImportEntries). "run"/"result"
// deliberately aren't part of this: they're local-only (`runPhase`) so
// back/forward can never remount the run step and resubmit entries.
type NavigableStep = "account" | "file" | "mapping" | "dryrun";
const NAVIGABLE_STEPS: NavigableStep[] = [
  "account",
  "file",
  "mapping",
  "dryrun",
];

type ImportSearch = {
  account_id?: string | undefined;
  step?: NavigableStep | undefined;
};

// Nested under /entries (entries.tsx), so it already inherits that layout's
// authenticated-only redirect-to-/login gate — no separate gate needed here,
// same as entries.new.tsx.
export const Route = createFileRoute("/entries/import")({
  validateSearch: (search: Record<string, unknown>): ImportSearch => {
    const account_id =
      typeof search["account_id"] === "string"
        ? search["account_id"]
        : undefined;
    const defaultStep: NavigableStep = account_id ? "file" : "account";
    const rawStep = search["step"];
    const step =
      typeof rawStep === "string" &&
      (NAVIGABLE_STEPS as string[]).includes(rawStep)
        ? (rawStep as NavigableStep)
        : defaultStep;
    return { account_id, step };
  },
  component: ImportEntries,
});

type RunPhase = "run" | "result";

type RunResult = { created: number; failed: RunFailure[]; canceled: boolean };

/**
 * The wizard's "account"/"file"/"mapping"/"dryrun" steps live in the URL
 * (`?step=`) so the browser's physical Back button steps back exactly one
 * wizard step instead of leaving the page — while all the in-memory state
 * (parsed file, mapping, overrides) survives untouched, since a search-param
 * -only navigation re-renders this same route component rather than
 * remounting it. In-page "Back" links call `window.history.back()` for the
 * same effect, so both paths are exactly symmetric.
 *
 * "run" and "result" are deliberately kept out of the URL as local state
 * (`runPhase`): they take rendering priority over the URL step, and a
 * `useEffect` clears `runPhase` whenever the URL step changes from under
 * it (i.e. on a real back/forward navigation) so a stale run/result view
 * can't get stuck on screen — but nothing in history can ever cause
 * `runPhase` to become "run" again on its own, so `ImportRunStep` can never
 * be remounted (and its entries resubmitted) via back/forward.
 */
function ImportEntries() {
  const { account_id: presetAccountId, step } = Route.useSearch();
  const { t } = useTranslation();
  const navigate = Route.useNavigate();

  const accountLocked = presetAccountId !== undefined;

  const [accounts, setAccounts] = useState<Account[]>([]);
  const [categories, setCategories] = useState<Category[]>([]);
  const [tags, setTags] = useState<Tag[]>([]);

  const [accountId, setAccountId] = useState(presetAccountId ?? "");
  const [parsed, setParsed] = useState<ParsedFile | null>(null);
  const [mapping, setMapping] = useState<MappingState>(initialMappingState);
  const [finalRows, setFinalRows] = useState<ClassifiedRow[] | null>(null);
  const [runPhase, setRunPhase] = useState<RunPhase | null>(null);
  const [runResult, setRunResult] = useState<RunResult | null>(null);
  const rowMapping = toRowMapping(mapping);

  useEffect(() => {
    Promise.all([
      api.GET("/api/accounts"),
      api.GET("/api/categories"),
      api.GET("/api/tags"),
    ]).then(([a, c, tg]) => {
      setAccounts(a.data ?? []);
      setCategories(c.data ?? []);
      setTags(tg.data ?? []);
    });
  }, []);

  // A real back/forward navigation changed the URL step out from under an
  // in-progress/finished run — drop back to the step-driven view instead of
  // leaving a stale run/result screen showing.
  // biome-ignore lint/correctness/useExhaustiveDependencies: intentionally keyed on step alone, to re-run on every step change even though the body doesn't read it
  useEffect(() => {
    setRunPhase(null);
  }, [step]);

  // A hard reload can land on step=mapping/dryrun with no in-memory parsed
  // file to show (that state doesn't survive a reload) — fall back to the
  // file step rather than rendering blank.
  useEffect(() => {
    if ((step === "mapping" || step === "dryrun") && parsed === null) {
      navigate({
        search: (prev) => ({ ...prev, step: "file" }),
        replace: true,
      });
    }
  }, [step, parsed, navigate]);

  function startOver() {
    setRunPhase(null);
    if (!accountLocked) setAccountId("");
    setParsed(null);
    setMapping(initialMappingState);
    setFinalRows(null);
    setRunResult(null);
    navigate({
      search: (prev) => ({
        ...prev,
        step: accountLocked ? "file" : "account",
      }),
      replace: true,
    });
  }

  const account = accounts.find((a) => a.id === accountId);
  const submittableRows =
    finalRows?.filter((r) => r.classification !== "failed") ?? [];
  const dryRunFailedRows =
    finalRows?.filter((r) => r.classification === "failed") ?? [];

  return (
    <section className="mx-auto flex w-full max-w-3xl flex-col gap-6 px-6 py-12 sm:px-10">
      <header className="flex flex-col gap-1">
        <h1 className="text-2xl font-semibold tracking-tight">
          {t("entries.import.title")}
        </h1>
        {account && (
          <p className="text-sm text-zinc-500 dark:text-zinc-400">
            {t("entries.import.intoAccount", { account: account.title })}
          </p>
        )}
      </header>

      {!runPhase && step === "account" && (
        <ImportAccountStep
          accounts={accounts}
          value={accountId}
          onChange={setAccountId}
          onContinue={() =>
            navigate({ search: (prev) => ({ ...prev, step: "file" }) })
          }
        />
      )}

      {!runPhase && step === "file" && (
        <ImportFileStep
          onParsed={(result) => {
            setParsed(result);
            setMapping(initialMappingState);
            navigate({ search: (prev) => ({ ...prev, step: "mapping" }) });
          }}
          onBack={() => {
            if (accountLocked) {
              navigate({ to: "/entries" });
            } else {
              window.history.back();
            }
          }}
        />
      )}

      {!runPhase && step === "mapping" && parsed && (
        <ImportMappingStep
          fields={parsed.fields}
          rows={parsed.rows}
          ignoredFields={parsed.ignoredFields}
          categories={categories}
          tags={tags}
          mapping={mapping}
          onMappingChange={setMapping}
          onContinue={() =>
            navigate({ search: (prev) => ({ ...prev, step: "dryrun" }) })
          }
          onBack={() => window.history.back()}
        />
      )}

      {!runPhase && step === "dryrun" && parsed && rowMapping && (
        <ImportDryRunStep
          sourceRows={parsed.rows}
          rowMapping={rowMapping}
          fields={parsed.fields}
          currency={account?.currency ?? ""}
          onContinue={(rows) => {
            setFinalRows(rows);
            setRunPhase("run");
          }}
          onBack={() => window.history.back()}
        />
      )}

      {runPhase === "run" && (
        <ImportRunStep
          accountId={accountId}
          rows={submittableRows}
          tagNames={mapping.tagNames}
          tags={tags}
          onFinished={(result) => {
            setRunResult({
              created: result.created,
              failed: result.failed,
              canceled: result.canceled,
            });
            setRunPhase("result");
          }}
        />
      )}

      {runPhase === "result" && runResult && (
        <ImportResultStep
          created={runResult.created}
          dryRunFailedRows={dryRunFailedRows}
          runFailures={runResult.failed}
          canceled={runResult.canceled}
          onStartOver={startOver}
        />
      )}

      <div>
        <Link
          to="/entries"
          search={{}}
          className="text-sm font-medium text-zinc-600 underline underline-offset-2 hover:text-zinc-900 dark:text-zinc-400 dark:hover:text-zinc-100"
        >
          {t("entries.import.cancel")}
        </Link>
      </div>
    </section>
  );
}
