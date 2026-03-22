# Schema Design Decisions

## Table Structure

We use two tables: `shopping_carts` and `cart_items`.

**Why two tables instead of one?**

A single table with embedded JSON items would simplify writes, but makes it hard to update individual items, enforce constraints, or query by product. Two normalized tables give us:

- Independent add/update/remove of individual items without rewriting the entire cart
- Referential integrity via foreign keys
- Efficient JOINs for cart retrieval (`GET /shopping-carts/{id}`)

## Key Strategy

### Primary Keys

- `shopping_carts.cart_id` — AUTO_INCREMENT BIGINT. Simple, sequential, InnoDB-friendly (clustered index inserts are append-only, avoiding page splits).
- `cart_items.item_id` — AUTO_INCREMENT BIGINT. Same rationale.

### Foreign Keys

- `cart_items.cart_id` references `shopping_carts.cart_id` with `ON DELETE CASCADE`. Deleting a cart automatically removes all its items, preventing orphaned data.

### Unique Constraint

- `UNIQUE KEY uk_cart_product (cart_id, product_id)` — Ensures one row per product per cart. This enables `INSERT ... ON DUPLICATE KEY UPDATE quantity = quantity + VALUES(quantity)` for the `POST /shopping-carts/{id}/items` endpoint, handling both "add new item" and "update existing item" in a single atomic operation.

## Index Strategy

| Index | Columns | Purpose |
|-------|---------|---------|
| `PRIMARY KEY` on `shopping_carts` | `(cart_id)` | Fast cart retrieval by ID (`GET /shopping-carts/{id}`) — single row lookup, <50ms |
| `idx_carts_customer_created` | `(customer_id, created_at DESC)` | Customer purchase history queries — covers both filter and sort without filesort |
| `PRIMARY KEY` on `cart_items` | `(item_id)` | Row-level identification |
| `uk_cart_product` | `(cart_id, product_id)` | 1) Prevents duplicate products in a cart. 2) Serves as the lookup index for JOIN when retrieving cart items by `cart_id` — the leftmost column matches the JOIN condition |

**Why no additional index on `cart_items.cart_id`?**

The unique key `uk_cart_product (cart_id, product_id)` already has `cart_id` as its leftmost column. MySQL can use this index for any query filtering by `cart_id`, so a separate single-column index would be redundant.

## Cart-Item Relationship

- One-to-many: one cart has many items
- Enforced by foreign key constraint
- `ON DELETE CASCADE` ensures data integrity — no orphaned items
- The unique key `(cart_id, product_id)` models the business rule that each product appears at most once per cart (quantity is incremented instead of creating duplicate rows)

## Transaction Design

- **Create cart**: Single INSERT, no transaction needed
- **Add/update item**: `INSERT ... ON DUPLICATE KEY UPDATE` — single atomic statement, handles concurrent modifications to different products in the same cart without conflicts
- **Get cart**: `SELECT ... JOIN` — read-only, no transaction needed
- For concurrent access to the *same* product in the *same* cart, InnoDB's row-level locking on the unique key ensures correctness

## Trade-offs Considered

| Decision | Trade-off |
|----------|-----------|
| Two tables vs. single table with JSON items | More JOINs, but better data integrity, indexing, and per-item operations |
| AUTO_INCREMENT vs. UUID | Sequential IDs are faster for InnoDB inserts but expose ordering info; acceptable for internal cart IDs |
| `ON DELETE CASCADE` vs. application-level cleanup | Less control but guarantees no orphaned data at the database level |
| No `unit_price` in `cart_items` | Matches the OpenAPI spec (only `product_id` + `quantity`); price lookup would be handled by a separate product service if needed |
| `BIGINT UNSIGNED` vs. `INT` | Over-provisioned for a homework assignment, but matches production practice and avoids future migration |
