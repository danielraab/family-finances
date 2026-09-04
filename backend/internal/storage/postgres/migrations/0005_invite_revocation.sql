-- 0005_invite_revocation: lets an invite's own inviter (or an admin) revoke
-- it, blocking acceptance, and lets an admin soft-delete an already-revoked
-- invite to hide it from listings. See openspec change `invite-revocation`.

ALTER TABLE invites
    ADD COLUMN revoked_at timestamptz,
    ADD COLUMN deleted_at timestamptz;
