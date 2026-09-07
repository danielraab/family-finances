-- 0016_category_seed_defaults: seeds every existing user who currently has
-- zero categories with the same starter set new signups get going forward
-- (internal/category's SeedDefaults). Purely additive — a user who already
-- built their own tree is untouched. See openspec change
-- account-types-per-user.

INSERT INTO categories (owner_id, name, sort_order)
SELECT u.id, v.name, v.ord
FROM users u
CROSS JOIN (VALUES ('Salary', 0), ('Groceries', 1), ('Rent', 2),
                    ('Utilities', 3), ('Transportation', 4),
                    ('Entertainment', 5), ('Health', 6), ('Other', 7))
    AS v(name, ord)
WHERE NOT EXISTS (
    SELECT 1 FROM categories c WHERE c.owner_id = u.id
);
