// Generates src/api/schema.d.ts from ../openapi/openapi.yaml (the hand-written
// API contract). Run via `pnpm generate:api`. CI regenerates and fails on a
// diff. See openapi/README.md.

import { mkdirSync, readFileSync, writeFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import openapiTS, { astToString } from "openapi-typescript";
import { parse } from "yaml";

const here = dirname(fileURLToPath(import.meta.url));
const specPath = resolve(here, "../../openapi/openapi.yaml");
const outPath = resolve(here, "../src/api/schema.d.ts");

const METHODS = ["get", "put", "post", "delete", "options", "head", "patch", "trace"];

const doc = parse(readFileSync(specPath, "utf8"));

// Drop browser-navigation operations (x-internal: true) so the typed client
// exposes no fetch helper for a redirect flow.
for (const [path, item] of Object.entries(doc.paths ?? {})) {
  for (const method of METHODS) {
    if (item?.[method]?.["x-internal"]) delete item[method];
  }
  if (!METHODS.some((m) => item?.[m])) delete doc.paths[path];
}

const ast = await openapiTS(doc, { alphabetize: true });
const banner =
  "// GENERATED from openapi/openapi.yaml by scripts/generate-api.mjs — do not edit.\n" +
  "// Run `pnpm generate:api` after changing the spec.\n\n";

mkdirSync(dirname(outPath), { recursive: true });
writeFileSync(outPath, banner + astToString(ast));
console.log(`wrote ${outPath}`);
