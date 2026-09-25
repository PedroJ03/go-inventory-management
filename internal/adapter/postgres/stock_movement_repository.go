package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/pedroj/go-inventory-management/internal/application"
	"github.com/pedroj/go-inventory-management/internal/domain"
)

const insertStockMovementSQL = `INSERT INTO stock_movements (type, units, product_id, from_location_id, to_location_id, created_at) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id, type, units, product_id, from_location_id, to_location_id, created_at`

// StockMovementRepository persists stock movements using explicit PostgreSQL SQL.
type StockMovementRepository struct {
	db *sql.DB
}

var _ application.StockMovementRepository = (*StockMovementRepository)(nil)

func NewStockMovementRepository(db *sql.DB) (*StockMovementRepository, error) {
	if db == nil {
		return nil, errors.New("stock movement repository database must not be nil")
	}
	return &StockMovementRepository{db: db}, nil
}

func (r *StockMovementRepository) Create(ctx context.Context, movement domain.StockMovement) (domain.StockMovement, error) {
	if err := movement.Validate(); err != nil {
		return domain.StockMovement{}, fmt.Errorf("create stock movement: %w: %v", application.ErrInvalidStockMovement, err)
	}

	created, err := scanStockMovement(r.db.QueryRowContext(ctx, insertStockMovementSQL,
		movement.Type,
		movement.Units,
		movement.ProductID,
		movement.FromID,
		movement.ToID,
		movement.CreatedAt,
	))
	if err != nil {
		return domain.StockMovement{}, mapStockMovementWriteError(err)
	}
	return created, nil
}

func scanStockMovement(row rowScanner) (domain.StockMovement, error) {
	var productIDText string
	var movement domain.StockMovement
	if err := row.Scan(
		&movement.ID,
		&movement.Type,
		&movement.Units,
		&productIDText,
		&movement.FromID,
		&movement.ToID,
		&movement.CreatedAt,
	); err != nil {
		return domain.StockMovement{}, err
	}

	productID, err := uuid.Parse(productIDText)
	if err != nil {
		return domain.StockMovement{}, fmt.Errorf("parse stock movement product id: %w", err)
	}
	movement.ProductID = productID
	if err := movement.Validate(); err != nil {
		return domain.StockMovement{}, fmt.Errorf("validate database stock movement: %w", err)
	}
	return movement, nil
}

func mapStockMovementWriteError(err error) error {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		switch pqErr.Code {
		case "23502", "23503", "23514", "22P02", "22003":
			return fmt.Errorf("%w: %v", application.ErrStockMovementConstraint, err)
		}
	}
	return fmt.Errorf("%w: %v", application.ErrStockMovementPersistence, err)
}
