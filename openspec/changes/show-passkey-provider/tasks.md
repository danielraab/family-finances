# Tasks

## 1. Provider lookup

- [x] 1.1 Add `internal/aaguid` with the embedded upstream `aaguid.json` snapshot, `Lookup([]byte) (Provider, bool)` (zero/short/unknown → false; icons kept only as SVG data URIs) and the refresh command in the package doc; verify with unit tests for a known AAGUID, zero, unknown, a wrong length, a non-SVG icon being dropped, and the snapshot being non-empty

## 2. Backend

- [x] 2.1 Add `auth.PasskeyProvider`, the `ProviderLookup` interface and `WithPasskeyProviders`; fill `PasskeyInfo.Provider` in the list and register-finish responses; wire `aaguid` in `main.go`; verify with handler tests (software authenticator reporting a known AAGUID → provider set; zero AAGUID → null)
- [x] 2.2 Default an empty registration name to the provider name (≤100 chars) or "Passkey"; verify with handler tests for known-provider default, zero-AAGUID default, and an explicit name winning
- [x] 2.3 Add `provider` to the `Passkey` schema in `openapi/openapi.yaml` and regenerate `backend/openapi.yaml` and `frontend/src/api/schema.d.ts`; verify the openapicheck conformance in the handler tests, spectral lint and no drift

## 3. Web client

- [x] 3.1 Show the provider icon (theme-aware, falling back to the key glyph) and name on Passkeys tab rows; verify with `pnpm lint`, `pnpm exec tsc`, `pnpm build`, and a browser check against the embedded build with a virtual authenticator

## 4. Docs

- [x] 4.1 Note `internal/aaguid` (purpose, manual refresh) in `backend/AGENTS.md` and the provider display in `frontend/AGENTS.md`; verify the package layout block lists it
