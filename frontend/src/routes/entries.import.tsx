import { createFileRoute, Link, useNavigate } from "@tanstack/react-router";
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

type ImportSearch = { account_id?: string | undefined };

// Nested under /entries (entries.tsx), so it already inherits that layout's
// authenticated-only redirect-to-/login gate — no separate gate needed here,
// same as entries.new.tsx.
export const Route = createFileRoute("/entries/import")({
  validateSearch: (search: Record<string, unknown>): ImportSearch => ({
    account_id:
      typeof search["account_id"] === "string"
        ? search["account_id"]
        : undefined,
  }),
  component: ImportEntries,
});

type Step = "account" | "file" | "mapping" | "dryrun" | "run" | "result";

type RunResult = { created: number; failed: RunFailure[]; canceled: boolean };

function ImportEntries() {
  const { account_id: presetAccountId } = Route.useSearch();
  const { t } = useTranslation();
  const navigate = useNavigate();

  const accountLocked = presetAccountId !== undefined;

  const [accounts, setAccounts] = useState<Account[]>([]);
  const [categories, setCategories] = useState<Category[]>([]);
  const [tags, setTags] = useState<Tag[]>([]);

  const [step, setStep] = useState<Step>(accountLocked ? "file" : "account");
  const [accountId, setAccountId] = useState(presetAccountId ?? "");
  const [parsed, setParsed] = useState<ParsedFile | null>(null);
  const [mapping, setMapping] = useState<MappingState>(initialMappingState);
  const [finalRows, setFinalRows] = useState<ClassifiedRow[] | null>(null);
  const [runResult, setRunResult] = useState<RunResult | null>(null);

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

  function startOver() {
    setStep(accountLocked ? "file" : "account");
    if (!accountLocked) setAccountId("");
    setParsed(null);
    setMapping(initialMappingState);
    setFinalRows(null);
    setRunResult(null);
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

      {step === "account" && (
        <ImportAccountStep
          accounts={accounts}
          value={accountId}
          onChange={setAccountId}
          onContinue={() => setStep("file")}
        />
      )}

      {step === "file" && (
        <ImportFileStep
          onParsed={(result) => {
            setParsed(result);
            setMapping(initialMappingState);
            setStep("mapping");
          }}
          onBack={() => {
            if (accountLocked) {
              navigate({ to: "/entries" });
            } else {
              setStep("account");
            }
          }}
        />
      )}

      {step === "mapping" && parsed && (
        <ImportMappingStep
          fields={parsed.fields}
          rows={parsed.rows}
          ignoredFields={parsed.ignoredFields}
          categories={categories}
          tags={tags}
          mapping={mapping}
          onMappingChange={setMapping}
          onContinue={() => setStep("dryrun")}
          onBack={() => setStep("file")}
        />
      )}

      {step === "dryrun" && parsed && (
        <ImportDryRunStep
          sourceRows={parsed.rows}
          // Non-null: the mapping step only enables "Continue" once
          // toRowMapping(mapping) is non-null.
          rowMapping={toRowMapping(mapping)!}
          fields={parsed.fields}
          currency={account?.currency ?? ""}
          onContinue={(rows) => {
            setFinalRows(rows);
            setStep("run");
          }}
          onBack={() => setStep("mapping")}
        />
      )}

      {step === "run" && (
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
            setStep("result");
          }}
        />
      )}

      {step === "result" && runResult && (
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
