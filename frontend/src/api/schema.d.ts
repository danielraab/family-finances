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
        /** List every account type */
        get: operations["getAccountTypes"];
        put?: never;
        /** Create an account type (admin only) */
        post: operations["postAccountTypes"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/account-types/{id}": {
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
         * Delete an account type (admin only)
         * @description 409 if a non-deleted account still references it — deletion is never cascaded.
         */
        delete: operations["deleteAccountType"];
        options?: never;
        head?: never;
        /** Rename an account type (admin only) */
        patch: operations["patchAccountType"];
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
        /** Create an account */
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
        /** Update an account */
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
        /** List the full category tree */
        get: operations["getCategories"];
        put?: never;
        /** Create a category (admin only) */
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
         * Delete a category (admin only)
         * @description 409 if it has child categories or is referenced by a non-deleted entry — deletion is never cascaded.
         */
        delete: operations["deleteCategory"];
        options?: never;
        head?: never;
        /**
         * Update a category (admin only)
         * @description Reparenting onto the category itself or one of its own descendants is rejected with 422.
         */
        patch: operations["patchCategory"];
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
         * @description account_id and kind are immutable — there is no field for them on this request body at all.
         */
        patch: operations["patchEntry"];
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
}
export type webhooks = Record<string, never>;
export interface components {
    schemas: {
        Account: {
            /** Format: date */
            closing_date?: string;
            /** Format: date-time */
            created_at: string;
            /** @description ISO-4217 shape (three uppercase letters), not a canonical list. */
            currency: string;
            description?: string;
            /** @description Reversible; blocks creating new entries against the account. Independent of closing_date (informational only) and of soft delete. */
            disabled: boolean;
            financial_institute?: string;
            id: string;
            /** Format: date */
            opening_date: string;
            title: string;
            type_id: string;
            /** Format: date-time */
            updated_at: string;
        };
        AccountCreate: {
            /** Format: date */
            closing_date?: string;
            currency: string;
            description?: string;
            financial_institute?: string;
            /** Format: date */
            opening_date: string;
            title: string;
            type_id: string;
        };
        AccountType: {
            /** Format: date-time */
            created_at: string;
            id: string;
            name: string;
        };
        AccountTypeWrite: {
            name: string;
        };
        AccountUpdate: {
            /**
             * Format: date
             * @description Explicit null clears it (re-opening the account).
             */
            closing_date?: string | null;
            currency?: string;
            description?: string;
            financial_institute?: string;
            /** Format: date */
            opening_date?: string;
            title?: string;
            type_id?: string;
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
        Category: {
            /** Format: date-time */
            created_at: string;
            id: string;
            name: string;
            /** @description Absent for a root category. */
            parent_id?: string;
        };
        CategoryWrite: {
            name?: string;
            /** @description Explicit null makes it a root category. */
            parent_id?: string | null;
        };
        EmailStartRequest: {
            /** Format: email */
            email: string;
        };
        Entry: {
            account_id: string;
            /**
             * Format: int64
             * @description Integer minor units at a fixed 4 decimal places (e.g. 105000 represents 10.5000 in the account's currency). Not configurable — see account-entries.
             */
            amount: number;
            /** Format: date-time */
            booking_timestamp: string;
            /** @description Required for a transaction, optional for a balance_adjustment. */
            category_id?: string;
            /** Format: date-time */
            created_at: string;
            description?: string;
            id: string;
            kind: components["schemas"]["EntryKind"];
            tag_ids: string[];
            title: string;
            /** Format: date-time */
            updated_at: string;
        };
        EntryCreate: {
            account_id: string;
            /** Format: int64 */
            amount: number;
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
        /** @description No account_id or kind field — both are immutable after creation. */
        EntryUpdate: {
            /** Format: int64 */
            amount?: number;
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
            /** @description Every account type. */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["AccountType"][];
                };
            };
            401: components["responses"]["Unauthorized"];
        };
    };
    postAccountTypes: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody: {
            content: {
                "application/json": components["schemas"]["AccountTypeWrite"];
            };
        };
        responses: {
            /** @description The created account type. */
            201: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["AccountType"];
                };
            };
            400: components["responses"]["BadRequest"];
            401: components["responses"]["Unauthorized"];
            403: components["responses"]["Forbidden"];
        };
    };
    deleteAccountType: {
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
            /** @description The account type is deleted. */
            204: {
                headers: {
                    [name: string]: unknown;
                };
                content?: never;
            };
            401: components["responses"]["Unauthorized"];
            403: components["responses"]["Forbidden"];
            404: components["responses"]["NotFound"];
            409: components["responses"]["Conflict"];
        };
    };
    patchAccountType: {
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
                "application/json": components["schemas"]["AccountTypeWrite"];
            };
        };
        responses: {
            /** @description The updated account type. */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["AccountType"];
                };
            };
            400: components["responses"]["BadRequest"];
            401: components["responses"]["Unauthorized"];
            403: components["responses"]["Forbidden"];
            404: components["responses"]["NotFound"];
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
            /** @description Every category, including its parent_id. */
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
            403: components["responses"]["Forbidden"];
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
            403: components["responses"]["Forbidden"];
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
            403: components["responses"]["Forbidden"];
            404: components["responses"]["NotFound"];
            422: components["responses"]["UnprocessableEntity"];
        };
    };
    getEntries: {
        parameters: {
            query?: {
                /** @description Repeatable. Omitted means every account the caller owns. */
                account_id?: string[];
                /** @description Opaque cursor from a previous response's next_cursor. */
                after?: string;
                /** @description Matches this category and every descendant. */
                category_id?: string;
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
}
