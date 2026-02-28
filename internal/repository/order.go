// Package repository provides raw-SQL database access for all Ghostfolio domain
// entities against a Postgres database originally managed by Prisma. Each file
// in this package owns one domain entity and exposes a thin repository struct
// whose methods map 1-to-1 with the service-layer operations used throughout
// the NestJS reference implementation.
//
// All repositories accept a db.DBTX interface so they work transparently with
// both a plain *sqlx.DB and a *sqlx.Tx — callers that need atomicity simply
// pass an active transaction.
//
// Column lists are defined as package-level constants (e.g. orderColumns) to
// guarantee that SELECT and RETURNING clauses stay in sync with each other and
// with the model struct field tags used by sqlx struct scanning.
package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/db"
	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/model"
	"github.com/jmoiron/sqlx"
)

// OrderRepository provides database access for the Order entity, which
// represents a single investment transaction (buy, sell, dividend, etc.)
// belonging to a user. Orders are the primary unit of portfolio activity and
// are referenced by analytics, export, and performance-calculation pipelines.
type OrderRepository struct {
	db db.DBTX
}

// NewOrderRepository constructs an OrderRepository backed by the provided
// database handle. The handle may be a connection pool or an open transaction.
func NewOrderRepository(db db.DBTX) *OrderRepository {
	return &OrderRepository{db: db}
}

// orderColumns enumerates every column returned from the Order table.
// Centralising the list here prevents SELECT/RETURNING drift and makes
// schema changes a single-site edit.
const orderColumns = `id, "accountId", "accountUserId", comment, "createdAt",
	currency, date, fee, "isDraft", quantity, "symbolProfileId",
	type, "unitPrice", "updatedAt", "userId"`

// FindByUserID returns all orders owned by userID, sorted newest-first.
// Returns an empty slice (not an error) when the user has no orders.
func (r *OrderRepository) FindByUserID(ctx context.Context, userID string) ([]model.Order, error) {
	var items []model.Order
	err := sqlx.SelectContext(ctx, r.db,
		&items,
		`SELECT `+orderColumns+`
		 FROM "Order"
		 WHERE "userId" = $1
		 ORDER BY date DESC`, userID)
	if err != nil {
		return nil, fmt.Errorf("query orders: %w", err)
	}
	return items, nil
}

// FindByID fetches a single order by its primary key regardless of owner.
// Returns pgx.ErrNoRows (wrapped) when no matching record exists.
// WHY: admin-level operations need ID-only lookup without a user scope check.
func (r *OrderRepository) FindByID(ctx context.Context, id string) (model.Order, error) {
	var item model.Order
	err := sqlx.GetContext(ctx, r.db,
		&item,
		`SELECT `+orderColumns+`
		 FROM "Order"
		 WHERE id = $1`, id)
	if err != nil {
		return model.Order{}, fmt.Errorf("query order: %w", err)
	}
	return item, nil
}

// FindByIDAndUserID fetches a single order scoped to a specific owner.
// The combined WHERE clause acts as an ownership check — a non-owner receives
// ErrNoRows rather than a permissions error, which matches NestJS behaviour.
func (r *OrderRepository) FindByIDAndUserID(ctx context.Context, id string, userID string) (model.Order, error) {
	var item model.Order
	err := sqlx.GetContext(ctx, r.db,
		&item,
		`SELECT `+orderColumns+`
		 FROM "Order"
		 WHERE id = $1 AND "userId" = $2`, id, userID)
	if err != nil {
		return model.Order{}, fmt.Errorf("query order by user: %w", err)
	}
	return item, nil
}

// Create inserts a new Order row and returns the full persisted record,
// including database-generated timestamps. The caller is responsible for
// generating the ID before calling this method.
func (r *OrderRepository) Create(ctx context.Context, order model.Order) (model.Order, error) {
	var item model.Order
	err := sqlx.GetContext(ctx, r.db,
		&item,
		`INSERT INTO "Order" (id, "accountId", "accountUserId", comment,
		                      currency, date, fee, "isDraft", quantity,
		                      "symbolProfileId", type, "unitPrice", "userId")
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		 RETURNING `+orderColumns,
		order.ID, order.AccountId, order.AccountUserId, order.Comment,
		order.Currency, order.Date, order.Fee, order.IsDraft, order.Quantity,
		order.SymbolProfileId, order.Type, order.UnitPrice, order.UserId)
	if err != nil {
		return model.Order{}, fmt.Errorf("insert order: %w", err)
	}
	return item, nil
}

// Update replaces mutable fields on an existing order identified solely by ID.
// WHY: used by admin endpoints that are allowed to update any order without
// verifying ownership at the query level (ownership is enforced upstream).
func (r *OrderRepository) Update(ctx context.Context, order model.Order) (model.Order, error) {
	var item model.Order
	err := sqlx.GetContext(ctx, r.db,
		&item,
		`UPDATE "Order"
		 SET "accountId" = $2, "accountUserId" = $3, comment = $4,
		     currency = $5, date = $6, fee = $7, "isDraft" = $8,
		     quantity = $9, "symbolProfileId" = $10, type = $11,
		     "unitPrice" = $12
		 WHERE id = $1
		 RETURNING `+orderColumns,
		order.ID, order.AccountId, order.AccountUserId, order.Comment,
		order.Currency, order.Date, order.Fee, order.IsDraft, order.Quantity,
		order.SymbolProfileId, order.Type, order.UnitPrice)
	if err != nil {
		return model.Order{}, fmt.Errorf("update order: %w", err)
	}
	return item, nil
}

// UpdateByUserID replaces mutable fields on an order only when the record
// belongs to userID. Returns ErrNoRows (wrapped) if the order does not exist
// or is owned by a different user, preventing cross-user mutation.
func (r *OrderRepository) UpdateByUserID(ctx context.Context, order model.Order, userID string) (model.Order, error) {
	var item model.Order
	err := sqlx.GetContext(ctx, r.db,
		&item,
		`UPDATE "Order"
		 SET "accountId" = $2, "accountUserId" = $3, comment = $4,
		     currency = $5, date = $6, fee = $7, "isDraft" = $8,
		     quantity = $9, "symbolProfileId" = $10, type = $11,
		     "unitPrice" = $12
		 WHERE id = $1 AND "userId" = $13
		 RETURNING `+orderColumns,
		order.ID, order.AccountId, order.AccountUserId, order.Comment,
		order.Currency, order.Date, order.Fee, order.IsDraft, order.Quantity,
		order.SymbolProfileId, order.Type, order.UnitPrice, userID)
	if err != nil {
		return model.Order{}, fmt.Errorf("update order by user: %w", err)
	}
	return item, nil
}

// CountAll returns the total number of orders across all users.
// Used by the admin dashboard to surface platform-wide activity metrics.
func (r *OrderRepository) CountAll(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRowxContext(ctx, `SELECT COUNT(*) FROM "Order"`).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count orders: %w", err)
	}
	return count, nil
}

// CountByDataSourceAndSymbol returns how many orders reference a given
// dataSource+symbol pair via the SymbolProfile join. This is used before
// deleting a SymbolProfile to check whether any orders would be orphaned.
func (r *OrderRepository) CountByDataSourceAndSymbol(ctx context.Context, dataSource string, symbol string) (int, error) {
	var count int
	err := r.db.QueryRowxContext(ctx,
		`SELECT COUNT(*) FROM "Order" o
		 JOIN "SymbolProfile" sp ON o."symbolProfileId" = sp.id
		 WHERE sp."dataSource" = $1 AND sp.symbol = $2`, dataSource, symbol).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count orders by symbol: %w", err)
	}
	return count, nil
}

// OrderWithProfile extends Order with SymbolProfile fields needed by the
// portfolio calculator. Column aliases use the sp_ prefix to avoid collision.
type OrderWithProfile struct {
	model.Order
	SPCurrency      string          `db:"sp_currency"`
	SPDataSource    string          `db:"sp_dataSource"`
	SPSymbol        string          `db:"sp_symbol"`
	SPAssetClass    *string         `db:"sp_assetClass"`
	SPAssetSubClass *string         `db:"sp_assetSubClass"`
	SPUserID        *string         `db:"sp_userId"`
	SPName          *string         `db:"sp_name"`
	SPUrl           *string         `db:"sp_url"`
	SPCountries     json.RawMessage `db:"sp_countries"`
	SPSectors       json.RawMessage `db:"sp_sectors"`
	TagIDs          json.RawMessage `db:"tag_ids"`
}

// FindActiveByUserIDWithProfile returns all non-draft orders for a user, joined
// with SymbolProfile fields needed by the portfolio calculator. Results are
// ordered by date ascending for chronological processing.
func (r *OrderRepository) FindActiveByUserIDWithProfile(ctx context.Context, userID string) ([]OrderWithProfile, error) {
	var items []OrderWithProfile
	err := sqlx.SelectContext(ctx, r.db,
		&items,
		`SELECT o.id, o."accountId", o."accountUserId", o.comment, o."createdAt",
		        o.currency, o.date, o.fee, o."isDraft", o.quantity, o."symbolProfileId",
		        o.type, o."unitPrice", o."updatedAt", o."userId",
		        sp.currency AS sp_currency,
		        sp."dataSource" AS "sp_dataSource",
		        sp.symbol AS sp_symbol,
		        sp."assetClass" AS "sp_assetClass",
		        sp."assetSubClass" AS "sp_assetSubClass",
		        sp."userId" AS "sp_userId",
		        sp.name AS sp_name,
		        sp.url AS sp_url,
		        COALESCE(sp.countries, '[]'::jsonb) AS sp_countries,
		        COALESCE(sp.sectors, '[]'::jsonb) AS sp_sectors,
		        COALESCE(tags.tag_ids, '[]'::json) AS tag_ids
		 FROM "Order" o
		 JOIN "SymbolProfile" sp ON o."symbolProfileId" = sp.id
		 LEFT JOIN LATERAL (
		   SELECT json_agg(t.id ORDER BY t.id) AS tag_ids
		   FROM "_OrderToTag" ot
		   JOIN "Tag" t ON t.id = ot."B"
		   WHERE ot."A" = o.id
		 ) tags ON TRUE
		 WHERE o."userId" = $1 AND o."isDraft" = false
		 ORDER BY o.date ASC`, userID)
	if err != nil {
		return nil, fmt.Errorf("query orders with profile: %w", err)
	}
	return items, nil
}

// Delete removes an order by ID without an ownership check.
// WHY: admin paths need unconditional deletion; user-facing paths should use
// DeleteByIDAndUserID instead.
func (r *OrderRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM "Order" WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete order: %w", err)
	}
	return nil
}

// DeleteByIDAndUserID removes a specific order owned by userID and returns
// the number of rows affected (0 or 1). A count of 0 means either the order
// does not exist or belongs to a different user — callers should treat this
// as a not-found / forbidden condition.
func (r *OrderRepository) DeleteByIDAndUserID(ctx context.Context, id string, userID string) (int64, error) {
	result, err := r.db.ExecContext(ctx, `DELETE FROM "Order" WHERE id = $1 AND "userId" = $2`, id, userID)
	if err != nil {
		return 0, fmt.Errorf("delete order by user: %w", err)
	}
	return result.RowsAffected()
}

// ReplaceTagsForOrders clears existing tags and assigns new ones for a set of
// order IDs. Used by the holding tags update endpoint.
func (r *OrderRepository) ReplaceTagsForOrders(ctx context.Context, orderIDs []string, tagIDs []string) error {
	if len(orderIDs) == 0 {
		return nil
	}

	// Delete existing tag associations
	delQuery, delArgs, err := sqlx.In(
		`DELETE FROM "_OrderToTag" WHERE "A" IN (?)`, orderIDs)
	if err != nil {
		return fmt.Errorf("build delete tags query: %w", err)
	}
	delQuery = sqlx.Rebind(sqlx.DOLLAR, delQuery)
	if _, err := r.db.ExecContext(ctx, delQuery, delArgs...); err != nil {
		return fmt.Errorf("delete order tags: %w", err)
	}

	// Insert new associations
	if len(tagIDs) == 0 {
		return nil
	}
	for _, orderID := range orderIDs {
		for _, tagID := range tagIDs {
			if _, err := r.db.ExecContext(ctx,
				`INSERT INTO "_OrderToTag" ("A", "B") VALUES ($1, $2) ON CONFLICT DO NOTHING`,
				orderID, tagID); err != nil {
				return fmt.Errorf("insert order tag: %w", err)
			}
		}
	}
	return nil
}

// FindIDsByUserAndSymbol returns order IDs for a user matching a given symbol.
func (r *OrderRepository) FindIDsByUserAndSymbol(ctx context.Context, userID, dataSource, symbol string) ([]string, error) {
	var ids []string
	err := sqlx.SelectContext(ctx, r.db, &ids,
		`SELECT o.id FROM "Order" o
		 JOIN "SymbolProfile" sp ON o."symbolProfileId" = sp.id
		 WHERE o."userId" = $1 AND sp."dataSource" = $2 AND sp.symbol = $3`, userID, dataSource, symbol)
	if err != nil {
		return nil, fmt.Errorf("find order ids by symbol: %w", err)
	}
	return ids, nil
}

// DeleteByUserID removes all orders belonging to userID, returning the count
// of deleted rows. Used during account deletion to cascade-remove activity
// before the User row itself is dropped.
func (r *OrderRepository) DeleteByUserID(ctx context.Context, userID string) (int64, error) {
	result, err := r.db.ExecContext(ctx, `DELETE FROM "Order" WHERE "userId" = $1`, userID)
	if err != nil {
		return 0, fmt.Errorf("delete orders by user: %w", err)
	}
	return result.RowsAffected()
}
