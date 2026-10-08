/**
 * Thin wrapper over the browser's WebAuthn API for the passkey ceremonies.
 *
 * The backend speaks the WebAuthn JSON forms: start endpoints return options
 * whose binary fields (challenge, user id, credential ids) are base64url
 * strings, and finish endpoints expect the credential as
 * `PublicKeyCredential.toJSON()` produces it. Modern browsers convert both
 * ways natively (`parseCreationOptionsFromJSON`, `parseRequestOptionsFromJSON`,
 * `toJSON`); for older ones (e.g. Safari < 18.4) this module does the same
 * base64url conversion by hand.
 */

/** Outcome of a browser prompt: the credential JSON, or the visitor cancelled. */
export type CeremonyResult =
  | { status: "ok"; credential: Record<string, unknown> }
  | { status: "cancelled" };

/** Whether this browser exposes WebAuthn at all. */
export function passkeysSupported(): boolean {
  return (
    typeof window !== "undefined" &&
    typeof window.PublicKeyCredential !== "undefined" &&
    typeof navigator.credentials?.create === "function"
  );
}

/** Run `navigator.credentials.create` for registration options from the server. */
export async function createPasskey(
  options: Record<string, unknown>,
): Promise<CeremonyResult> {
  const publicKey = parseCreationOptions(options);
  return runCeremony(() => navigator.credentials.create({ publicKey }));
}

/** Run `navigator.credentials.get` for (discoverable) sign-in options from the server. */
export async function getPasskey(
  options: Record<string, unknown>,
): Promise<CeremonyResult> {
  const publicKey = parseRequestOptions(options);
  return runCeremony(() => navigator.credentials.get({ publicKey }));
}

async function runCeremony(
  prompt: () => Promise<Credential | null>,
): Promise<CeremonyResult> {
  try {
    const credential = await prompt();
    if (!(credential instanceof PublicKeyCredential)) {
      return { status: "cancelled" };
    }
    return { status: "ok", credential: credentialToJSON(credential) };
  } catch (error) {
    // NotAllowedError covers both "dismissed" and "timed out"; AbortError an
    // aborted prompt. Neither is something to show an error for.
    if (
      error instanceof DOMException &&
      (error.name === "NotAllowedError" || error.name === "AbortError")
    ) {
      return { status: "cancelled" };
    }
    throw error;
  }
}

// --- options: JSON → native ----------------------------------------------

type DescriptorJSON = { id: string; type: string; transports?: string[] };

function parseCreationOptions(
  json: Record<string, unknown>,
): PublicKeyCredentialCreationOptions {
  if (typeof PublicKeyCredential.parseCreationOptionsFromJSON === "function") {
    return PublicKeyCredential.parseCreationOptionsFromJSON(
      json as unknown as PublicKeyCredentialCreationOptionsJSON,
    );
  }
  const o = json as unknown as PublicKeyCredentialCreationOptionsJSON;
  return {
    ...(o as unknown as PublicKeyCredentialCreationOptions),
    challenge: fromBase64URL(o.challenge),
    user: { ...o.user, id: fromBase64URL(o.user.id) },
    excludeCredentials: (o.excludeCredentials ?? []).map(descriptor),
  };
}

function parseRequestOptions(
  json: Record<string, unknown>,
): PublicKeyCredentialRequestOptions {
  if (typeof PublicKeyCredential.parseRequestOptionsFromJSON === "function") {
    return PublicKeyCredential.parseRequestOptionsFromJSON(
      json as unknown as PublicKeyCredentialRequestOptionsJSON,
    );
  }
  const o = json as unknown as PublicKeyCredentialRequestOptionsJSON;
  return {
    ...(o as unknown as PublicKeyCredentialRequestOptions),
    challenge: fromBase64URL(o.challenge),
    allowCredentials: (o.allowCredentials ?? []).map(descriptor),
  };
}

function descriptor(d: DescriptorJSON): PublicKeyCredentialDescriptor {
  return {
    type: "public-key",
    id: fromBase64URL(d.id),
    ...(d.transports
      ? { transports: d.transports as AuthenticatorTransport[] }
      : {}),
  };
}

// --- credential: native → JSON -------------------------------------------

function credentialToJSON(
  credential: PublicKeyCredential,
): Record<string, unknown> {
  if (typeof credential.toJSON === "function") {
    return credential.toJSON() as unknown as Record<string, unknown>;
  }
  const r = credential.response;
  const response: Record<string, unknown> = {
    clientDataJSON: toBase64URL(r.clientDataJSON),
  };
  if (r instanceof AuthenticatorAttestationResponse) {
    response["attestationObject"] = toBase64URL(r.attestationObject);
    response["transports"] = r.getTransports?.() ?? [];
  } else if (r instanceof AuthenticatorAssertionResponse) {
    response["authenticatorData"] = toBase64URL(r.authenticatorData);
    response["signature"] = toBase64URL(r.signature);
    if (r.userHandle) {
      response["userHandle"] = toBase64URL(r.userHandle);
    }
  }
  return {
    id: credential.id,
    rawId: toBase64URL(credential.rawId),
    type: credential.type,
    response,
    clientExtensionResults: credential.getClientExtensionResults(),
    authenticatorAttachment: credential.authenticatorAttachment ?? undefined,
  };
}

// --- base64url -------------------------------------------------------------

export function toBase64URL(buffer: ArrayBuffer): string {
  let binary = "";
  for (const byte of new Uint8Array(buffer)) {
    binary += String.fromCharCode(byte);
  }
  return btoa(binary)
    .replace(/\+/g, "-")
    .replace(/\//g, "_")
    .replace(/=+$/, "");
}

export function fromBase64URL(value: string): ArrayBuffer {
  const base64 = value.replace(/-/g, "+").replace(/_/g, "/");
  const padded = base64 + "=".repeat((4 - (base64.length % 4)) % 4);
  const binary = atob(padded);
  const bytes = new Uint8Array(binary.length);
  for (let i = 0; i < binary.length; i++) {
    bytes[i] = binary.charCodeAt(i);
  }
  return bytes.buffer;
}
