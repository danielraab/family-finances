-- 0019_account_category_icon_color: accounts and categories each gain an
-- optional icon and colour — short opaque presentation tokens the backend
-- stores and echoes but never interprets (the client owns the icon set and
-- palette). See openspec change `account-category-icon-color`.
--
-- Purely additive: existing rows get NULL, which the API represents as the
-- field being absent and the client renders as no badge.

ALTER TABLE accounts
    ADD COLUMN icon  text,
    ADD COLUMN color text;

ALTER TABLE categories
    ADD COLUMN icon  text,
    ADD COLUMN color text;
