package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/pedroj/go-inventory-management/internal/application"
	"github.com/pedroj/go-inventory-management/internal/domain"
)

const sellStockLevelSQL = `UPDATE stock_levels SET units = units - $3 WHERE product_id = $1 AND location_id = $2 AND units >= $3 RETURNING product_id, location_id, units`

// StockSellingRepository atomically updates stock and records a movement.
type StockSellingRepository struct {
	db *sql.DB
}

var _ application.StockSellingPort = (*StockSellingRepository)(nil)

func NewStockSellingRepository(db *sql.DB) (*StockSellingRepository, error) {
	if db == nil {
		return nil, errors.New("stock selling repository database must not be nil")
	}
	return &StockSellingRepository{db: db}, nil
}

func (r *StockSellingRepository) Sell(ctx context.Context, movement domain.StockMovement) (domain.StockLevel, domain.StockMovement, error) {
	if err := movement.Validate(); err != nil {
		return domain.StockLevel{}, domain.StockMovement{}, fmt.Errorf("sell stock: %w: %v", application.ErrInvalidStockMovement, err)
	}
	if movement.Type != domain.MovementSelling {
		return domain.StockLevel{}, domain.StockMovement{}, fmt.Errorf("sell stock: %w: movement must be selling", application.ErrInvalidStockMovement)
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.StockLevel{}, domain.StockMovement{}, fmt.Errorf("begin sell stock transaction: %w", application.ErrStockLevelPersistence)
	}

	stockLevel, err := scanStockLevel(tx.QueryRowContext(ctx, sellStockLevelSQL, movement.ProductID, *movement.FromID, movement.Units))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.StockLevel{}, domain.StockMovement{}, rollbackSell(tx, application.ErrInsufficientStock)
	}
	if err != nil {
		return domain.StockLevel{}, domain.StockMovement{}, rollbackSell(tx, mapStockLevelWriteError(err))
	}
	persistedMovement, err := scanStockMovement(tx.QueryRowContext(ctx, insertStockMovementSQL,
		movement.Type, movement.Units, movement.ProductID, movement.FromID, movement.ToID, movement.CreatedAt,
	))
	if err != nil {
		return domain.StockLevel{}, domain.StockMovement{}, rollbackSell(tx, mapStockMovementWriteError(err))
	}
	if err := tx.Commit(); err != nil {
		return domain.StockLevel{}, domain.StockMovement{}, rollbackSell(tx, fmt.Errorf("commit sell stock transaction: %w", application.ErrStockMovementPersistence))
	}
	return stockLevel, persistedMovement, nil
}

func rollbackSell(tx *sql.Tx, operationErr error) error {
	if rollbackErr := tx.Rollback(); rollbackErr != nil && !errors.Is(rollbackErr, sql.ErrTxDone) {
		return fmt.Errorf("%w (rollback failed: %v)", operationErr, rollbackErr)
	}
	return operationErr
}
