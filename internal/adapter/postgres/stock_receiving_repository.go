package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/PedroJ03/go-inventory-management/internal/application"
	"github.com/PedroJ03/go-inventory-management/internal/domain"
)

const receiveStockLevelSQL = `INSERT INTO stock_levels (product_id, location_id, units) VALUES ($1, $2, $3) ON CONFLICT (product_id, location_id) DO UPDATE SET units = stock_levels.units + EXCLUDED.units RETURNING product_id, location_id, units`

// StockReceivingRepository atomically updates stock and records a movement.
type StockReceivingRepository struct {
	db *sql.DB
}

var _ application.StockReceivingPort = (*StockReceivingRepository)(nil)

func NewStockReceivingRepository(db *sql.DB) (*StockReceivingRepository, error) {
	if db == nil {
		return nil, errors.New("stock receiving repository database must not be nil")
	}
	return &StockReceivingRepository{db: db}, nil
}

func (r *StockReceivingRepository) Receive(ctx context.Context, movement domain.StockMovement) (domain.StockLevel, domain.StockMovement, error) {
	if err := movement.Validate(); err != nil {
		return domain.StockLevel{}, domain.StockMovement{}, fmt.Errorf("receive stock: %w: %v", application.ErrInvalidStockMovement, err)
	}
	if movement.Type != domain.MovementReceiving {
		return domain.StockLevel{}, domain.StockMovement{}, fmt.Errorf("receive stock: %w: movement must be receiving", application.ErrInvalidStockMovement)
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.StockLevel{}, domain.StockMovement{}, fmt.Errorf("begin receive stock transaction: %w", application.ErrStockLevelPersistence)
	}

	var productIDText string
	var stockLevel domain.StockLevel
	err = tx.QueryRowContext(ctx, receiveStockLevelSQL, movement.ProductID, *movement.ToID, movement.Units).Scan(&productIDText, &stockLevel.LocationID, &stockLevel.Units)
	if err != nil {
		return domain.StockLevel{}, domain.StockMovement{}, rollbackReceive(tx, mapStockLevelWriteError(err))
	}
	stockLevel.ProductID, err = uuid.Parse(productIDText)
	if err != nil {
		return domain.StockLevel{}, domain.StockMovement{}, rollbackReceive(tx, fmt.Errorf("parse received stock product id: %w", err))
	}
	if err := stockLevel.Validate(); err != nil {
		return domain.StockLevel{}, domain.StockMovement{}, rollbackReceive(tx, fmt.Errorf("validate received stock level: %w", err))
	}

	persistedMovement, err := scanStockMovement(tx.QueryRowContext(ctx, insertStockMovementSQL,
		movement.Type, movement.Units, movement.ProductID, movement.FromID, movement.ToID, movement.CreatedAt,
	))
	if err != nil {
		return domain.StockLevel{}, domain.StockMovement{}, rollbackReceive(tx, mapStockMovementWriteError(err))
	}
	if err := tx.Commit(); err != nil {
		return domain.StockLevel{}, domain.StockMovement{}, rollbackReceive(tx, fmt.Errorf("commit receive stock transaction: %w", application.ErrStockMovementPersistence))
	}
	return stockLevel, persistedMovement, nil
}

func rollbackReceive(tx *sql.Tx, operationErr error) error {
	if rollbackErr := tx.Rollback(); rollbackErr != nil && !errors.Is(rollbackErr, sql.ErrTxDone) {
		return fmt.Errorf("%w (rollback failed: %v)", operationErr, rollbackErr)
	}
	return operationErr
}
