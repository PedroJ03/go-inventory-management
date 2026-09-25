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

const (
	findStockLevelSQL             = `SELECT product_id, location_id, units FROM stock_levels WHERE product_id = $1 AND location_id = $2`
	upsertStockLevelSQL           = `INSERT INTO stock_levels (product_id, location_id, units) VALUES ($1, $2, $3) ON CONFLICT (product_id, location_id) DO UPDATE SET units = EXCLUDED.units`
	hasPositiveStockAtLocationSQL = `SELECT EXISTS (SELECT 1 FROM stock_levels WHERE location_id = $1 AND units > 0)`
)

// StockLevelRepository persists stock levels using explicit PostgreSQL SQL.
type StockLevelRepository struct {
	db *sql.DB
}

var _ application.StockLevelRepository = (*StockLevelRepository)(nil)

func NewStockLevelRepository(db *sql.DB) (*StockLevelRepository, error) {
	if db == nil {
		return nil, errors.New("stock level repository database must not be nil")
	}
	return &StockLevelRepository{db: db}, nil
}

func (r *StockLevelRepository) FindByProductAndLocation(ctx context.Context, productID uuid.UUID, locationID int64) (domain.StockLevel, error) {
	if productID == uuid.Nil || locationID <= 0 {
		return domain.StockLevel{}, fmt.Errorf("find stock level: %w", application.ErrInvalidStockLevel)
	}

	stockLevel, err := scanStockLevel(r.db.QueryRowContext(ctx, findStockLevelSQL, productID, locationID))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.StockLevel{}, application.ErrStockLevelNotFound
	}
	if err != nil {
		return domain.StockLevel{}, mapStockLevelReadError(err)
	}
	return stockLevel, nil
}

func (r *StockLevelRepository) Upsert(ctx context.Context, stockLevel domain.StockLevel) error {
	if err := stockLevel.Validate(); err != nil {
		return fmt.Errorf("upsert stock level: %w: %v", application.ErrInvalidStockLevel, err)
	}

	if _, err := r.db.ExecContext(ctx, upsertStockLevelSQL, stockLevel.ProductID, stockLevel.LocationID, stockLevel.Units); err != nil {
		return mapStockLevelWriteError(err)
	}
	return nil
}

func (r *StockLevelRepository) HasPositiveStockAtLocation(ctx context.Context, locationID int64) (bool, error) {
	if locationID <= 0 {
		return false, fmt.Errorf("check positive stock: %w", application.ErrInvalidStockLevel)
	}

	var hasPositiveStock bool
	if err := r.db.QueryRowContext(ctx, hasPositiveStockAtLocationSQL, locationID).Scan(&hasPositiveStock); err != nil {
		return false, mapStockLevelReadError(err)
	}
	return hasPositiveStock, nil
}

type stockLevelRowScanner interface {
	Scan(dest ...any) error
}

func scanStockLevel(row stockLevelRowScanner) (domain.StockLevel, error) {
	var productIDText string
	var stockLevel domain.StockLevel
	if err := row.Scan(&productIDText, &stockLevel.LocationID, &stockLevel.Units); err != nil {
		return domain.StockLevel{}, err
	}

	productID, err := uuid.Parse(productIDText)
	if err != nil {
		return domain.StockLevel{}, fmt.Errorf("parse stock level product id: %w", err)
	}
	stockLevel.ProductID = productID
	if err := stockLevel.Validate(); err != nil {
		return domain.StockLevel{}, fmt.Errorf("validate database stock level: %w", err)
	}
	return stockLevel, nil
}

func mapStockLevelWriteError(err error) error {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		switch pqErr.Code {
		case "23502", "23514", "22P02", "23503":
			return fmt.Errorf("%w: %v", application.ErrStockLevelConstraint, err)
		}
	}
	return fmt.Errorf("%w: %v", application.ErrStockLevelPersistence, err)
}

func mapStockLevelReadError(err error) error {
	return fmt.Errorf("%w: %v", application.ErrStockLevelPersistence, err)
}
