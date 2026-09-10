// GENERATED from openapi/openapi.yaml by scripts/generate-api.mjs — do not edit.
// Run `pnpm generate:api` after changing the spec.

export interface paths {
    "/api/account-types": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /**
         * List the caller's distinct in-use account type labels
         * @description The distinct, non-empty `type` values on the caller's own non-deleted accounts, trimmed, compared verbatim (case-sensitive), sorted case-insensitively ascending. For client autocomplete only; there is no endpoint to create, rename, disable, or delete a type — a type exists by being written on an account.
         */
        get: operations["getAccountTypes"];
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/accounts": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /** List the caller's accounts */
        get: operations["getAccounts"];
        put?: never;
        /**
         * Create an account
         * @description 422 when `type` is empty or only whitespace.
         */
        post: operations["postAccounts"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/accounts/{id}": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /**
         * Get an account
         * @description 404 for an account the caller does not own, identical to a nonexistent one — existence is not disclosed across owners.
         */
        get: operations["getAccount"];
        put?: never;
        post?: never;
        /**
         * Soft-delete an account
         * @description One-way — there is no undelete endpoint.
         */
        delete: operations["deleteAccount"];
        options?: never;
        head?: never;
        /**
         * Update an account
         * @description 422 when `type` is present but empty or only whitespace. A shared `owner`-tier caller may change `type` like any other field.
         */
        patch: operations["patchAccount"];
        trace?: never;
    };
    "/api/accounts/{id}/balance": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /**
         * The account's live balance
         * @description Always computed live, never cached — see the account-entries capability. Equals the latest non-deleted balance_adjustment at or before as_of (or 0 if none), plus every non-deleted transaction after it, up to as_of.
         */
        get: operations["getAccountBalance"];
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/accounts/{id}/disable": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * Disable an account
         * @description Reversible via enable. Blocks creating new entries against the account; does not hide it or affect its existing entries.
         */
        post: operations["postAccountDisable"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/accounts/{id}/enable": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /** Re-enable a disabled account */
        post: operations["postAccountEnable"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/accounts/{id}/shares": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /**
         * List an account's shares
         * @description Visible to any caller holding at least view permission (real ownership or any share) — every user with access to an account can see who else has access. Does not include the real owner as a row (their identity is on the Account itself via owner_name); a client renders that as the fixed first row separately.
         */
        get: operations["getAccountShares"];
        put?: never;
        /**
         * Share an account with a user by email
         * @description Owner-tier only (real owner or a shared owner). Unlike authentication's anti-enumeration endpoints, this deliberately does not hide whether email matched a registered user — the caller already holds an authenticated, owner-tier grant on a real account. A match creates or updates (200-201, share overwrites any existing one for that user in place) a share and emails the recipient a notification with an application link. No match returns 200 with matched: false and invite_allowed reflecting whether the instance currently permits sending a new application invite for that email; no share is created. Sharing with the caller's own email or the account's real owner's email is rejected (400).
         */
        post: operations["postAccountShare"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/accounts/{id}/shares/{userId}": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        post?: never;
        /**
         * Revoke a share, or leave a shared account
         * @description Owner-tier callers may target any user's share (revoke). Any user may target their own (userId equal to the caller's own id) — self-leave, requiring no owner-tier permission of its own. Either way access is removed immediately and unconditionally (no soft delete); entries the removed user created remain on the account, attributed to them, for every remaining permission holder. The real owner can never be a target (400) — they carry no share row, so self-leave is unavailable to them.
         */
        delete: operations["deleteAccountShare"];
        options?: never;
        head?: never;
        /**
         * Change a share's permission
         * @description Owner-tier only. The real owner can never be a target (400) — they carry no share row to change.
         */
        patch: operations["patchAccountShare"];
        trace?: never;
    };
    "/api/auth/config": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /**
         * Sign-in methods the client should offer
         * @description Unauthenticated. Reports the sign-in affordances the web client should render on the login page. Exposes no provider secrets.
         */
        get: operations["getAuthConfig"];
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/auth/email/start": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * Start a magic-link sign-in
         * @description Always returns 200 regardless of whether an account exists or a mail was sent, to prevent account enumeration.
         */
        post: operations["postAuthEmailStart"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/auth/invites": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /**
         * List every invitation (admin only)
         * @description Every non-soft-deleted invite regardless of status (pending, accepted, expired, or revoked), newest first, each carrying the inviter's identity.
         */
        get: operations["getAuthInvites"];
        put?: never;
        /** Invite an email address */
        post: operations["postAuthInvites"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/auth/invites/{id}": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        post?: never;
        /**
         * Soft-delete a revoked invitation (admin only)
         * @description Rejected with 409 unless the invitation has already been revoked. One-way — there is no undelete endpoint. A soft-deleted invitation is excluded from both invitation listings.
         */
        delete: operations["deleteAuthInvite"];
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/auth/invites/{id}/revoke": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * Revoke an invitation
         * @description Permitted for the invite's own inviter or an admin. Idempotent — a repeat call on an already-revoked invite succeeds without changing its revoked_at. Permitted regardless of the invite's current status; a revoked invite stays visible in every listing.
         */
        post: operations["postAuthInviteRevoke"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/auth/invites/mine": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /**
         * List the authenticated user's own invitations
         * @description Every non-soft-deleted invite the caller personally created, regardless of status. No admin requirement.
         */
        get: operations["getAuthInvitesMine"];
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/auth/logout": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /** Revoke the current session */
        post: operations["postAuthLogout"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/auth/me": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /** The authenticated user */
        get: operations["getAuthMe"];
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/auth/users": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /** List every non-deleted user (admin only) */
        get: operations["getAuthUsers"];
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/auth/users/{id}": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        post?: never;
        /**
         * Soft-delete a user and revoke their sessions immediately (admin only)
         * @description One-way in this API — there is no undelete endpoint. No check prevents an admin from deleting their own account, including as the only remaining admin.
         */
        delete: operations["deleteAuthUser"];
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/auth/users/{id}/disable": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * Disable a user and revoke their sessions immediately (admin only)
         * @description No check prevents an admin from disabling their own account, including as the only remaining admin.
         */
        post: operations["postAuthUserDisable"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/auth/users/{id}/enable": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * Re-enable a disabled user (admin only)
         * @description Does not restore any session revoked while disabled.
         */
        post: operations["postAuthUserEnable"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/categories": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /** List the caller's category tree */
        get: operations["getCategories"];
        put?: never;
        /**
         * Create a category
         * @description Appended to the end of its sibling group (same parent_id, or root if omitted).
         */
        post: operations["postCategories"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/categories/{id}": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        post?: never;
        /**
         * Delete a category
         * @description A soft delete (irreversible — no undelete). 409 if it has a non-deleted child category or is referenced by a non-deleted entry — deletion is never cascaded.
         */
        delete: operations["deleteCategory"];
        options?: never;
        head?: never;
        /**
         * Update a category
         * @description Reparenting onto the category itself or one of its own descendants is rejected with 422. A reparent is appended to the end of the new parent's sibling group.
         */
        patch: operations["patchCategory"];
        trace?: never;
    };
    "/api/categories/{id}/disable": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * Disable a category
         * @description Reversible via enable. Blocks the category from being newly selected on an entry; does not affect any entry or child category already referencing it.
         */
        post: operations["postCategoryDisable"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/categories/{id}/enable": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /** Re-enable a disabled category */
        post: operations["postCategoryEnable"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/categories/{id}/move-down": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * Move a category down among its siblings
         * @description Swaps sort_order with the immediate next sibling (same parent_id). A no-op — 200, order unchanged — if the category is already last among its siblings.
         */
        post: operations["postCategoryMoveDown"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/categories/{id}/move-up": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * Move a category up among its siblings
         * @description Swaps sort_order with the immediate previous sibling (same parent_id). A no-op — 200, order unchanged — if the category is already first among its siblings.
         */
        post: operations["postCategoryMoveUp"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/entries": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /** List the caller's entries, filtered/searched/sorted, cursor-paginated */
        get: operations["getEntries"];
        put?: never;
        /**
         * Create an entry
         * @description 422 when account_id does not name an account the caller owns, or names a disabled account.
         */
        post: operations["postEntries"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/entries/{id}": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /** Get an entry */
        get: operations["getEntry"];
        put?: never;
        post?: never;
        /**
         * Soft-delete an entry
         * @description One-way — there is no undelete endpoint.
         */
        delete: operations["deleteEntry"];
        options?: never;
        head?: never;
        /**
         * Update an entry
         * @description kind is immutable — there is no field for it on this request body. account_id may be changed to move the entry to a different account the caller owns; a disabled target account is rejected (422), the same as entry creation.
         */
        patch: operations["patchEntry"];
        trace?: never;
    };
    "/api/entries/balance-series": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /**
         * The caller's running account balance sampled per day over a month
         * @description Returns the running balance of the caller's own, non-deleted accounts sampled at each local midnight of year/month — one point at 00:00 on every calendar day of the month, plus a closing point at 00:00 on the first day of the following month — using the caller's resolved timezone setting (see user-settings) to place those midnight boundaries, default UTC. Each point's value is the balance computed exactly as GET /api/accounts/{id}/balance computes it as of that instant (a balance_adjustment acts as an anchor), grouped per currency (the currency of the contributing accounts); accounts sharing a currency are summed. Every currency present in the selected accounts appears on every point, including with an amount of 0 — unlike flow-summary, a balance line needs a value at every point. A month with no activity is a run of identical points. Unlike GET /api/entries and GET /api/entries/summary, this operation accepts no category_id, category_mode, tag_id, from, to, or q — a category- or tag-filtered "balance" is not a balance, since a balance_adjustment carries neither; supplying any of them is a 400.
         */
        get: operations["getEntriesBalanceSeries"];
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/entries/flow-summary": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /**
         * Bucket the caller's matching entries into per-month or per-day income/outcome totals
         * @description Buckets the caller's own, non-deleted accounts' non-deleted entries by booking_timestamp — one bucket per calendar month of year when unit=month, or one bucket per calendar day of year/month when unit=day — using the caller's resolved timezone setting (see user-settings) to decide bucket boundaries, default UTC. Within each bucket, entries whose amount is positive are summed into income and the absolute value of entries whose amount is negative into outcome, grouped per currency (the currency of the entry's account) the same way GET /api/entries/summary groups its sum. Unlike GET /api/entries/summary, a balance_adjustment's amount (always a signed delta — see account-entries) is included, not excluded. Every period in the requested range is present, even one with no matching entries at all (empty income/outcome).
         */
        get: operations["getEntriesFlowSummary"];
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/entries/summary": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /**
         * Sum the caller's matching entries per currency, without paging
         * @description Accepts the same account_id, category_id/category_mode, tag_id, from/to, and q filters as GET /api/entries (no sort, dir, after, or limit — this is an aggregate, not a page). Always additionally restricted to kind=transaction, regardless of the caller's other filters — a balance_adjustment is an absolute reading, not a categorized delta. Computed directly rather than by paging through results, so it is accurate however many entries match.
         */
        get: operations["getEntriesSummary"];
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/healthz": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /** Readiness probe — pings the database */
        get: operations["getHealthz"];
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/openapi.yaml": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /** This OpenAPI document */
        get: operations["getOpenAPIDocument"];
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/settings": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /**
         * The authenticated user's resolved settings
         * @description Always fully populated — a field never set by the user resolves to its hardcoded application default.
         */
        get: operations["getSettings"];
        /**
         * Update one or more of the authenticated user's settings
         * @description Partial update: only the fields present in the body are changed.
         */
        put: operations["putSettings"];
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/tags": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /** List the caller's tags */
        get: operations["getTags"];
        put?: never;
        /** Create a tag */
        post: operations["postTags"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/tags/{id}": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        post?: never;
        /**
         * Delete a tag
         * @description Always allowed — detaches the tag from every entry it was attached to rather than being blocked by use, unlike a category or account type.
         */
        delete: operations["deleteTag"];
        options?: never;
        head?: never;
        /** Rename a tag */
        patch: operations["patchTag"];
        trace?: never;
    };
    "/api/tags/{id}/disable": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * Disable a tag
         * @description Reversible via enable. Blocks the tag from being newly attached to an entry; does not affect any entry already carrying it.
         */
        post: operations["postTagDisable"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/tags/{id}/enable": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /** Re-enable a disabled tag */
        post: operations["postTagEnable"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
}
export type webhooks = Record<string, never>;
export interface components {
    schemas: {
        Account: {
            /** Format: date */
            closing_date?: string;
            /** @description Optional opaque presentation token naming a client colour. Stored and echoed verbatim; never interpreted by the backend. Absent when unset. */
            color?: string;
            /** Format: date-time */
            created_at: string;
            /** @description ISO-4217 shape (three uppercase letters), not a canonical list. */
            currency: string;
            description?: string;
            /** @description Reversible; blocks creating new entries against the account. Independent of closing_date (informational only) and of soft delete. */
            disabled: boolean;
            financial_institute?: string;
            /** @description Optional opaque presentation token naming a client icon. Stored and echoed verbatim; never interpreted by the backend. Absent when unset. */
            icon?: string;
            id: string;
            /** Format: date */
            opening_date: string;
            /** @description The real owner's display name (or email, as a fallback). Present for every account, but only meaningful to render when shared is true. */
            owner_name?: string;
            /** @description The caller's own effective permission on this account — always "owner" for the real owner, never absent. */
            permission: components["schemas"]["AccountPermission"];
            /** @description True when the caller is not this account's real owner (i.e. they hold a share). Always false for the real owner, even when they have shared the account with others. */
            shared: boolean;
            title: string;
            /** @description Free-text label (e.g. "Checking"). Trimmed of surrounding whitespace on write; otherwise stored verbatim, not case-folded or checked against any list. Never empty. */
            type: string;
            /** Format: date-time */
            updated_at: string;
        };
        AccountCreate: {
            /** Format: date */
            closing_date?: string;
            /** @description Optional opaque client colour token; not interpreted by the backend. */
            color?: string;
            currency: string;
            description?: string;
            financial_institute?: string;
            /** @description Optional opaque client icon token; not interpreted by the backend. */
            icon?: string;
            /** Format: date */
            opening_date: string;
            title: string;
            /** @description Free-text label; trimmed on write, never empty. */
            type: string;
        };
        /**
         * @description Four tiers, each a strict superset of the one before it: view (read entries and balance), append (+ create entries, edit/delete only ones created by the same user), entry_admin (+ edit/delete any entry on the account), owner (+ edit account metadata, disable/enable/soft-delete the account, manage shares).
         * @enum {string}
         */
        AccountPermission: "view" | "append" | "entry_admin" | "owner";
        AccountShare: {
            /** Format: date-time */
            created_at: string;
            email: string;
            granted_by: string;
            granted_by_name: string;
            name: string;
            permission: components["schemas"]["AccountPermission"];
            /** Format: date-time */
            updated_at: string;
            user_id: string;
        };
        AccountShareInvite: {
            email: string;
            permission: components["schemas"]["AccountPermission"];
        };
        AccountShareInviteResult: {
            /** @description Only meaningful when matched is false: whether the instance currently allows sending a new application invite for this email (mirrors POST /api/auth/invites' own gating). */
            invite_allowed: boolean;
            /** @description True when email matched an existing, active user and a share was created or updated (share is then present). False when no such user exists — see design.md's deliberate, non-anti-enumeration decision for this endpoint. */
            matched: boolean;
            share?: components["schemas"]["AccountShare"];
        };
        AccountSharePermissionUpdate: {
            permission: components["schemas"]["AccountPermission"];
        };
        AccountUpdate: {
            /**
             * Format: date
             * @description Explicit null clears it (re-opening the account).
             */
            closing_date?: string | null;
            /** @description Optional opaque client colour token. An empty string clears it; omitting the field leaves it unchanged. */
            color?: string;
            currency?: string;
            description?: string;
            financial_institute?: string;
            /** @description Optional opaque client icon token. An empty string clears it; omitting the field leaves it unchanged. */
            icon?: string;
            /** Format: date */
            opening_date?: string;
            title?: string;
            /** @description Free-text label; trimmed on write. When present it must be non-empty after trimming. Omitting the field leaves it unchanged. */
            type?: string;
        };
        AdminUser: {
            /** Format: date-time */
            created_at: string;
            disabled: boolean;
            display_name?: string;
            /** Format: email */
            email: string;
            id: string;
            is_admin: boolean;
        };
        AuthConfig: {
            oidc: components["schemas"]["OidcLogin"] | null;
        };
        Balance: {
            /** Format: int64 */
            balance: number;
        };
        /** @description One sample of the running account balance — see GET /api/entries/balance-series. balances lists one entry per currency present in the selected accounts, always including that currency even when its amount is 0. */
        BalancePoint: {
            balances: components["schemas"]["CurrencySum"][];
            /**
             * Format: date
             * @description The point's local calendar day (YYYY-MM-DD). The closing point is labelled with the first day of the following month.
             */
            period: string;
        };
        Category: {
            /** @description Optional opaque presentation token naming a client colour. Stored and echoed verbatim; never interpreted by the backend. Absent when unset. */
            color?: string;
            /** Format: date-time */
            created_at: string;
            /** @description Blocks the category from being newly selected on an entry; existing entries and child categories referencing it are unaffected. */
            disabled: boolean;
            /** @description The number of the caller's own non-deleted entries directly categorized under this category. Direct references only — entries under a descendant category are not counted. */
            entry_count: number;
            /** @description Optional opaque presentation token naming a client icon. Stored and echoed verbatim; never interpreted by the backend. Absent when unset. */
            icon?: string;
            id: string;
            name: string;
            /** @description Absent for a root category. */
            parent_id?: string;
            /** @description Meaningful only among siblings sharing the same parent_id. */
            sort_order: number;
        };
        CategoryWrite: {
            /** @description Optional opaque client colour token. An empty string clears it; omitting the field leaves it unchanged. */
            color?: string;
            /** @description Optional opaque client icon token. An empty string clears it; omitting the field leaves it unchanged. */
            icon?: string;
            name?: string;
            /** @description Explicit null makes it a root category. */
            parent_id?: string | null;
        };
        CurrencySum: {
            /** Format: int64 */
            amount: number;
            currency: string;
        };
        EmailStartRequest: {
            /** Format: email */
            email: string;
        };
        Entry: {
            account_id: string;
            /**
             * Format: int64
             * @description Integer minor units at a fixed 4 decimal places (e.g. 105000 represents 10.5000 in the account's currency). Not configurable — see account-entries. Always a signed delta applied to the account's running balance: for a transaction, exactly what was submitted; for a balance_adjustment, computed automatically as the change from the balance immediately before it — never client-supplied for that kind.
             */
            amount: number;
            /**
             * Format: int64
             * @description The absolute reading a balance_adjustment's amount is computed against — present only when kind is balance_adjustment, null for a transaction. This is what the client supplies when creating or editing a balance_adjustment (see EntryCreate/ EntryUpdate), at the same 4-decimal-place scale as amount.
             */
            balance?: number | null;
            /** Format: date-time */
            booking_timestamp: string;
            /** @description Required for a transaction, optional for a balance_adjustment. */
            category_id?: string;
            /** Format: date-time */
            created_at: string;
            /** @description The id of the user who logged this entry — immutable after creation, not necessarily the account's real owner once it has been shared (see account-sharing). */
            created_by: string;
            /** @description The creator's display name (or email, as a fallback), resolved server-side so a viewer can see who logged an entry without a separate lookup. */
            created_by_name?: string;
            description?: string;
            id: string;
            kind: components["schemas"]["EntryKind"];
            tag_ids: string[];
            title: string;
            /** Format: date-time */
            updated_at: string;
        };
        /** @description amount is required when kind is transaction and rejected when kind is balance_adjustment; balance is required when kind is balance_adjustment and rejected when kind is transaction — exactly one of the two, per kind (400). */
        EntryCreate: {
            account_id: string;
            /**
             * Format: int64
             * @description Required for kind=transaction; must be omitted otherwise.
             */
            amount?: number;
            /**
             * Format: int64
             * @description Required for kind=balance_adjustment; must be omitted otherwise.
             */
            balance?: number;
            /** Format: date-time */
            booking_timestamp: string;
            category_id?: string;
            description?: string;
            kind: components["schemas"]["EntryKind"];
            tag_ids?: string[];
            title: string;
        };
        /** @enum {string} */
        EntryKind: "transaction" | "balance_adjustment";
        EntryPage: {
            items: components["schemas"]["Entry"][];
            next_cursor: string | null;
        };
        EntrySummary: {
            /** Format: int64 */
            count: number;
            sums: components["schemas"]["CurrencySum"][];
        };
        /** @description No kind field — it is immutable after creation. account_id may be set to move the entry to a different account the caller owns (see account-entries); it must not be disabled, the same rule creation applies. No currency conversion or validation is performed. amount is only settable when the entry's kind is transaction, balance only when it is balance_adjustment — supplying the other one is rejected (400). */
        EntryUpdate: {
            account_id?: string;
            /** Format: int64 */
            amount?: number;
            /** Format: int64 */
            balance?: number;
            /** Format: date-time */
            booking_timestamp?: string;
            /** @description Explicit null clears it (only valid when the entry's kind is balance_adjustment). */
            category_id?: string | null;
            description?: string;
            /** @description Replaces the full set, including clearing it with []. */
            tag_ids?: string[];
            title?: string;
        };
        Error: {
            error: string;
            request_id?: string;
        };
        /** @description One period's income/outcome totals — see GET /api/entries/flow-summary. A currency with only income (or only outcome) entries in this period appears in just that one array, never also in the other with an amount of 0. */
        FlowBucket: {
            income: components["schemas"]["CurrencySum"][];
            outcome: components["schemas"]["CurrencySum"][];
            /**
             * Format: date
             * @description The bucket's first calendar day (YYYY-MM-DD), regardless of unit — a day-unit bucket is just a single day.
             */
            period: string;
        };
        Invite: {
            /** Format: date-time */
            accepted_at: string | null;
            /** Format: email */
            email: string;
            /** Format: date-time */
            expires_at: string;
            id: string;
            invited_by: components["schemas"]["InviteInviter"];
            /**
             * Format: date-time
             * @description Set once, by the invite's own inviter or an admin. Blocks acceptance but does not remove the invite from any listing.
             */
            revoked_at: string | null;
        };
        InviteInviter: {
            display_name?: string;
            /** Format: email */
            email: string;
            id: string;
        };
        InviteRequest: {
            /** Format: email */
            email: string;
        };
        OidcLogin: {
            label: string;
            start_path: string;
        };
        StatusOk: {
            /** @enum {string} */
            status: "ok";
        };
        Tag: {
            /** Format: date-time */
            created_at: string;
            disabled: boolean;
            /** @description The number of the caller's own non-deleted entries currently carrying this tag. */
            entry_count: number;
            id: string;
            name: string;
        };
        TagWrite: {
            name: string;
        };
        User: {
            /** Format: date-time */
            created_at: string;
            display_name?: string;
            /** Format: email */
            email: string;
            id: string;
            is_admin: boolean;
            /**
             * @description The account's raw, unresolved language preference, or null when none is set — used by the client to prioritize it over browser detection (see the web-client-i18n capability). Distinct from GET /api/settings, whose language field is always resolved to a default when unset.
             * @enum {string|null}
             */
            language: "en" | "de" | null;
        };
        UserSettings: {
            default_currency: string;
            /** @description Display-only rounding preference — never affects how an amount is stored or edited (see account-entries' fixed 4-decimal-place storage). */
            displayed_decimal_places: number;
            /** @enum {string} */
            language: "en" | "de";
            timezone: string;
        };
        UserSettingsUpdate: {
            default_currency?: string;
            displayed_decimal_places?: number;
            /** @enum {string} */
            language?: "en" | "de";
            timezone?: string;
        };
    };
    responses: {
        /** @description A field in the request failed validation. */
        BadRequest: {
            headers: {
                [name: string]: unknown;
            };
            content: {
                "application/json": components["schemas"]["Error"];
            };
        };
        /** @description The request conflicts with the resource's current state. */
        Conflict: {
            headers: {
                [name: string]: unknown;
            };
            content: {
                "application/json": components["schemas"]["Error"];
            };
        };
        /** @description The authenticated user is not an admin. */
        Forbidden: {
            headers: {
                [name: string]: unknown;
            };
            content: {
                "application/json": components["schemas"]["Error"];
            };
        };
        /** @description No such resource — including one that belongs to a different owner, which behaves identically to nonexistent (existence is not disclosed across owners). */
        NotFound: {
            headers: {
                [name: string]: unknown;
            };
            content: {
                "application/json": components["schemas"]["Error"];
            };
        };
        /** @description No valid session was presented. */
        Unauthorized: {
            headers: {
                [name: string]: unknown;
            };
            content: {
                "application/json": components["schemas"]["Error"];
            };
        };
        /** @description The request is well-formed but violates a business rule (e.g. a reparent that would create a category cycle, or an entry against a disabled account). */
        UnprocessableEntity: {
            headers: {
                [name: string]: unknown;
            };
            content: {
                "application/json": components["schemas"]["Error"];
            };
        };
    };
    parameters: {
        AccountId: string;
        EntryId: string;
        ShareUserId: string;
    };
    requestBodies: never;
    headers: never;
    pathItems: never;
}
export type $defs = Record<string, never>;
export interface operations {
    getAccountTypes: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description The caller's distinct in-use type labels. */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": string[];
                };
            };
            401: components["responses"]["Unauthorized"];
        };
    };
    getAccounts: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description Every non-deleted account the caller owns. */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Account"][];
                };
            };
            401: components["responses"]["Unauthorized"];
        };
    };
    postAccounts: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody: {
            content: {
                "application/json": components["schemas"]["AccountCreate"];
            };
        };
        responses: {
            /** @description The created account, owned by the caller. */
            201: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Account"];
                };
            };
            400: components["responses"]["BadRequest"];
            401: components["responses"]["Unauthorized"];
            422: components["responses"]["UnprocessableEntity"];
        };
    };
    getAccount: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                id: components["parameters"]["AccountId"];
            };
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description The account. */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Account"];
                };
            };
            401: components["responses"]["Unauthorized"];
            404: components["responses"]["NotFound"];
        };
    };
    deleteAccount: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                id: components["parameters"]["AccountId"];
            };
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description The account is soft-deleted. */
            204: {
                headers: {
                    [name: string]: unknown;
                };
                content?: never;
            };
            401: components["responses"]["Unauthorized"];
            404: components["responses"]["NotFound"];
        };
    };
    patchAccount: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                id: components["parameters"]["AccountId"];
            };
            cookie?: never;
        };
        requestBody: {
            content: {
                "application/json": components["schemas"]["AccountUpdate"];
            };
        };
        responses: {
            /** @description The updated account. */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Account"];
                };
            };
            400: components["responses"]["BadRequest"];
            401: components["responses"]["Unauthorized"];
            404: components["responses"]["NotFound"];
            422: components["responses"]["UnprocessableEntity"];
        };
    };
    getAccountBalance: {
        parameters: {
            query?: {
                /** @description RFC3339 timestamp. Defaults to now. */
                as_of?: string;
            };
            header?: never;
            path: {
                id: components["parameters"]["AccountId"];
            };
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description The computed balance. */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Balance"];
                };
            };
            401: components["responses"]["Unauthorized"];
            404: components["responses"]["NotFound"];
        };
    };
    postAccountDisable: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                id: components["parameters"]["AccountId"];
            };
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description The account, now disabled. */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Account"];
                };
            };
            401: components["responses"]["Unauthorized"];
            404: components["responses"]["NotFound"];
        };
    };
    postAccountEnable: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                id: components["parameters"]["AccountId"];
            };
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description The account, now enabled. */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Account"];
                };
            };
            401: components["responses"]["Unauthorized"];
            404: components["responses"]["NotFound"];
        };
    };
    getAccountShares: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                id: components["parameters"]["AccountId"];
            };
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description Every current share on the account. */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["AccountShare"][];
                };
            };
            401: components["responses"]["Unauthorized"];
            404: components["responses"]["NotFound"];
        };
    };
    postAccountShare: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                id: components["parameters"]["AccountId"];
            };
            cookie?: never;
        };
        requestBody: {
            content: {
                "application/json": components["schemas"]["AccountShareInvite"];
            };
        };
        responses: {
            /** @description No user matched that email; see invite_allowed. */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["AccountShareInviteResult"];
                };
            };
            /** @description The share was created or updated. */
            201: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["AccountShareInviteResult"];
                };
            };
            400: components["responses"]["BadRequest"];
            401: components["responses"]["Unauthorized"];
            403: components["responses"]["Forbidden"];
            404: components["responses"]["NotFound"];
        };
    };
    deleteAccountShare: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                id: components["parameters"]["AccountId"];
                userId: components["parameters"]["ShareUserId"];
            };
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description The share is removed. */
            204: {
                headers: {
                    [name: string]: unknown;
                };
                content?: never;
            };
            400: components["responses"]["BadRequest"];
            401: components["responses"]["Unauthorized"];
            403: components["responses"]["Forbidden"];
            404: components["responses"]["NotFound"];
        };
    };
    patchAccountShare: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                id: components["parameters"]["AccountId"];
                userId: components["parameters"]["ShareUserId"];
            };
            cookie?: never;
        };
        requestBody: {
            content: {
                "application/json": components["schemas"]["AccountSharePermissionUpdate"];
            };
        };
        responses: {
            /** @description The updated share. */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["AccountShare"];
                };
            };
            400: components["responses"]["BadRequest"];
            401: components["responses"]["Unauthorized"];
            403: components["responses"]["Forbidden"];
            404: components["responses"]["NotFound"];
        };
    };
    getAuthConfig: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description The available sign-in affordances. */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["AuthConfig"];
                };
            };
        };
    };
    postAuthEmailStart: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody: {
            content: {
                "application/json": components["schemas"]["EmailStartRequest"];
            };
        };
        responses: {
            /** @description Request accepted. */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["StatusOk"];
                };
            };
            /** @description Malformed request body. */
            400: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Error"];
                };
            };
        };
    };
    getAuthInvites: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description Every invitation. */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Invite"][];
                };
            };
            401: components["responses"]["Unauthorized"];
            403: components["responses"]["Forbidden"];
        };
    };
    postAuthInvites: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody: {
            content: {
                "application/json": components["schemas"]["InviteRequest"];
            };
        };
        responses: {
            /** @description Invite created and an acceptance email sent. */
            201: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Invite"];
                };
            };
            /** @description Malformed request body. */
            400: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Error"];
                };
            };
            401: components["responses"]["Unauthorized"];
            /** @description Inviting is disabled on this instance. */
            403: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Error"];
                };
            };
        };
    };
    deleteAuthInvite: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                id: string;
            };
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description The invitation is soft-deleted. */
            204: {
                headers: {
                    [name: string]: unknown;
                };
                content?: never;
            };
            401: components["responses"]["Unauthorized"];
            403: components["responses"]["Forbidden"];
            /** @description No such invitation (or it is already soft-deleted). */
            404: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Error"];
                };
            };
            409: components["responses"]["Conflict"];
        };
    };
    postAuthInviteRevoke: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                id: string;
            };
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description The invitation, now revoked. */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Invite"];
                };
            };
            401: components["responses"]["Unauthorized"];
            /** @description The caller neither created this invitation nor is an admin. */
            403: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Error"];
                };
            };
            /** @description No such invitation (or it has been soft-deleted). */
            404: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Error"];
                };
            };
        };
    };
    getAuthInvitesMine: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description The caller's own invitations. */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Invite"][];
                };
            };
            401: components["responses"]["Unauthorized"];
        };
    };
    postAuthLogout: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description Session revoked. For a browser the ff_session cookie is cleared. */
            204: {
                headers: {
                    [name: string]: unknown;
                };
                content?: never;
            };
            401: components["responses"]["Unauthorized"];
        };
    };
    getAuthMe: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description The current user. */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["User"];
                };
            };
            401: components["responses"]["Unauthorized"];
        };
    };
    getAuthUsers: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description Every non-soft-deleted user. */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["AdminUser"][];
                };
            };
            401: components["responses"]["Unauthorized"];
            403: components["responses"]["Forbidden"];
        };
    };
    deleteAuthUser: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                id: string;
            };
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description The user is soft-deleted. */
            204: {
                headers: {
                    [name: string]: unknown;
                };
                content?: never;
            };
            401: components["responses"]["Unauthorized"];
            403: components["responses"]["Forbidden"];
            /** @description No such user. */
            404: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Error"];
                };
            };
        };
    };
    postAuthUserDisable: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                id: string;
            };
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description The user, now disabled. */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["AdminUser"];
                };
            };
            401: components["responses"]["Unauthorized"];
            403: components["responses"]["Forbidden"];
            /** @description No such user. */
            404: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Error"];
                };
            };
        };
    };
    postAuthUserEnable: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                id: string;
            };
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description The user, now enabled. */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["AdminUser"];
                };
            };
            401: components["responses"]["Unauthorized"];
            403: components["responses"]["Forbidden"];
            /** @description No such user. */
            404: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Error"];
                };
            };
        };
    };
    getCategories: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description Every non-deleted category the caller owns, including its parent_id. */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Category"][];
                };
            };
            401: components["responses"]["Unauthorized"];
        };
    };
    postCategories: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody: {
            content: {
                "application/json": components["schemas"]["CategoryWrite"];
            };
        };
        responses: {
            /** @description The created category. */
            201: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Category"];
                };
            };
            400: components["responses"]["BadRequest"];
            401: components["responses"]["Unauthorized"];
        };
    };
    deleteCategory: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                id: string;
            };
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description The category is deleted. */
            204: {
                headers: {
                    [name: string]: unknown;
                };
                content?: never;
            };
            401: components["responses"]["Unauthorized"];
            404: components["responses"]["NotFound"];
            409: components["responses"]["Conflict"];
        };
    };
    patchCategory: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                id: string;
            };
            cookie?: never;
        };
        requestBody: {
            content: {
                "application/json": components["schemas"]["CategoryWrite"];
            };
        };
        responses: {
            /** @description The updated category. */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Category"];
                };
            };
            400: components["responses"]["BadRequest"];
            401: components["responses"]["Unauthorized"];
            404: components["responses"]["NotFound"];
            422: components["responses"]["UnprocessableEntity"];
        };
    };
    postCategoryDisable: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                id: string;
            };
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description The category, now disabled. */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Category"];
                };
            };
            401: components["responses"]["Unauthorized"];
            404: components["responses"]["NotFound"];
        };
    };
    postCategoryEnable: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                id: string;
            };
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description The category, now enabled. */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Category"];
                };
            };
            401: components["responses"]["Unauthorized"];
            404: components["responses"]["NotFound"];
        };
    };
    postCategoryMoveDown: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                id: string;
            };
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description The category, with its updated (or unchanged) sort_order. */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Category"];
                };
            };
            401: components["responses"]["Unauthorized"];
            404: components["responses"]["NotFound"];
        };
    };
    postCategoryMoveUp: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                id: string;
            };
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description The category, with its updated (or unchanged) sort_order. */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Category"];
                };
            };
            401: components["responses"]["Unauthorized"];
            404: components["responses"]["NotFound"];
        };
    };
    getEntries: {
        parameters: {
            query?: {
                /** @description Repeatable. Omitted means every account the caller owns. */
                account_id?: string[];
                /** @description Opaque cursor from a previous response's next_cursor. */
                after?: string;
                /** @description Matches this category, plus every descendant unless category_mode=exact. */
                category_id?: string;
                /** @description Only meaningful together with category_id. subtree (the default) matches the category and every descendant; exact matches only that category. */
                category_mode?: "subtree" | "exact";
                dir?: "asc" | "desc";
                /** @description Inclusive booking_timestamp lower bound, RFC3339. */
                from?: string;
                kind?: components["schemas"]["EntryKind"];
                limit?: number;
                /** @description Case-insensitive substring match against title or description. */
                q?: string;
                sort?: "booking_timestamp" | "amount";
                tag_id?: string;
                /** @description Inclusive booking_timestamp upper bound, RFC3339. */
                to?: string;
            };
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description A page of matching entries. */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["EntryPage"];
                };
            };
            400: components["responses"]["BadRequest"];
            401: components["responses"]["Unauthorized"];
        };
    };
    postEntries: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody: {
            content: {
                "application/json": components["schemas"]["EntryCreate"];
            };
        };
        responses: {
            /** @description The created entry. */
            201: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Entry"];
                };
            };
            400: components["responses"]["BadRequest"];
            401: components["responses"]["Unauthorized"];
            422: components["responses"]["UnprocessableEntity"];
        };
    };
    getEntry: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                id: components["parameters"]["EntryId"];
            };
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description The entry. */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Entry"];
                };
            };
            401: components["responses"]["Unauthorized"];
            404: components["responses"]["NotFound"];
        };
    };
    deleteEntry: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                id: components["parameters"]["EntryId"];
            };
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description The entry is soft-deleted. */
            204: {
                headers: {
                    [name: string]: unknown;
                };
                content?: never;
            };
            401: components["responses"]["Unauthorized"];
            404: components["responses"]["NotFound"];
        };
    };
    patchEntry: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                id: components["parameters"]["EntryId"];
            };
            cookie?: never;
        };
        requestBody: {
            content: {
                "application/json": components["schemas"]["EntryUpdate"];
            };
        };
        responses: {
            /** @description The updated entry. */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Entry"];
                };
            };
            400: components["responses"]["BadRequest"];
            401: components["responses"]["Unauthorized"];
            404: components["responses"]["NotFound"];
            422: components["responses"]["UnprocessableEntity"];
        };
    };
    getEntriesBalanceSeries: {
        parameters: {
            query: {
                /** @description Repeatable. Omitted means every account the caller owns. */
                account_id?: string[];
                /** @description 1-12. */
                month: number;
                /** @description Only "day" is defined. */
                unit: "day";
                year: number;
            };
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description One point per day of the requested month, plus a closing point. */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": {
                        points: components["schemas"]["BalancePoint"][];
                    };
                };
            };
            400: components["responses"]["BadRequest"];
            401: components["responses"]["Unauthorized"];
        };
    };
    getEntriesFlowSummary: {
        parameters: {
            query: {
                /** @description Repeatable. Omitted means every account the caller owns. */
                account_id?: string[];
                /** @description 1-12. Required when unit=day, rejected when unit=month. */
                month?: number;
                unit: "month" | "day";
                year: number;
            };
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description One bucket per period in the requested range. */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": {
                        buckets: components["schemas"]["FlowBucket"][];
                    };
                };
            };
            400: components["responses"]["BadRequest"];
            401: components["responses"]["Unauthorized"];
        };
    };
    getEntriesSummary: {
        parameters: {
            query?: {
                /** @description Repeatable. Omitted means every account the caller owns. */
                account_id?: string[];
                /** @description Matches this category, plus every descendant unless category_mode=exact. */
                category_id?: string;
                category_mode?: "subtree" | "exact";
                /** @description Inclusive booking_timestamp lower bound, RFC3339. */
                from?: string;
                /** @description Case-insensitive substring match against title or description. */
                q?: string;
                tag_id?: string;
                /** @description Inclusive booking_timestamp upper bound, RFC3339. */
                to?: string;
            };
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description The matching transaction entries' amounts, summed per currency. */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["EntrySummary"];
                };
            };
            400: components["responses"]["BadRequest"];
            401: components["responses"]["Unauthorized"];
        };
    };
    getHealthz: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description The database answered a bounded ping. */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    /** @example ok */
                    "text/plain": string;
                };
            };
            /** @description No database is configured or it did not answer. */
            503: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "text/plain": string;
                };
            };
        };
    };
    getOpenAPIDocument: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description The OpenAPI document. */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/yaml": string;
                };
            };
        };
    };
    getSettings: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description The resolved settings. */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["UserSettings"];
                };
            };
            401: components["responses"]["Unauthorized"];
        };
    };
    putSettings: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody: {
            content: {
                "application/json": components["schemas"]["UserSettingsUpdate"];
            };
        };
        responses: {
            /** @description The resolved settings after the update. */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["UserSettings"];
                };
            };
            /** @description A field included in the body failed validation. */
            400: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Error"];
                };
            };
            401: components["responses"]["Unauthorized"];
        };
    };
    getTags: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description Every tag the caller owns. */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Tag"][];
                };
            };
            401: components["responses"]["Unauthorized"];
        };
    };
    postTags: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody: {
            content: {
                "application/json": components["schemas"]["TagWrite"];
            };
        };
        responses: {
            /** @description The created tag, owned by the caller. */
            201: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Tag"];
                };
            };
            400: components["responses"]["BadRequest"];
            401: components["responses"]["Unauthorized"];
            /** @description The caller already has a tag with this name. */
            409: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Error"];
                };
            };
        };
    };
    deleteTag: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                id: string;
            };
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description The tag is deleted. */
            204: {
                headers: {
                    [name: string]: unknown;
                };
                content?: never;
            };
            401: components["responses"]["Unauthorized"];
            404: components["responses"]["NotFound"];
        };
    };
    patchTag: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                id: string;
            };
            cookie?: never;
        };
        requestBody: {
            content: {
                "application/json": components["schemas"]["TagWrite"];
            };
        };
        responses: {
            /** @description The updated tag. */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Tag"];
                };
            };
            400: components["responses"]["BadRequest"];
            401: components["responses"]["Unauthorized"];
            404: components["responses"]["NotFound"];
            /** @description The caller already has a tag with this name. */
            409: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Error"];
                };
            };
        };
    };
    postTagDisable: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                id: string;
            };
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description The tag, now disabled. */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Tag"];
                };
            };
            401: components["responses"]["Unauthorized"];
            404: components["responses"]["NotFound"];
        };
    };
    postTagEnable: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                id: string;
            };
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description The tag, now enabled. */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Tag"];
                };
            };
            401: components["responses"]["Unauthorized"];
            404: components["responses"]["NotFound"];
        };
    };
}
