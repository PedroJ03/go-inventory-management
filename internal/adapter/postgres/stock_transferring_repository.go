package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/pedroj/go-inventory-management/internal/application"
	"github.com/pedroj/go-inventory-management/internal/domain"
)

const transferStockLevelSQL = `UPDATE stock_levels SET units = units - $3 WHERE product_id = $1 AND location_id = $2 AND units >= $3 RETURNING product_id, location_id, units`
const transferDestinationStockLevelSQL = `INSERT INTO stock_levels (product_id, location_id, units) VALUES ($1, $2, $3) ON CONFLICT (product_id, location_id) DO UPDATE SET units = stock_levels.units + EXCLUDED.units RETURNING product_id, location_id, units`

// StockTransferringRepository atomically updates both stock levels and records a movement.
type StockTransferringRepository struct {
	db *sql.DB
}

var _ application.StockTransferringPort = (*StockTransferringRepository)(nil)

func NewStockTransferringRepository(db *sql.DB) (*StockTransferringRepository, error) {
	if db == nil {
		return nil, errors.New("stock transferring repository database must not be nil")
	}
	return &StockTransferringRepository{db: db}, nil
}

func (r *StockTransferringRepository) Transfer(ctx context.Context, movement domain.StockMovement) (domain.StockLevel, domain.StockLevel, domain.StockMovement, error) {
	if err := movement.Validate(); err != nil {
		return domain.StockLevel{}, domain.StockLevel{}, domain.StockMovement{}, fmt.Errorf("transfer stock: %w: %v", application.ErrInvalidStockMovement, err)
	}
	if movement.Type != domain.MovementTransferring {
		return domain.StockLevel{}, domain.StockLevel{}, domain.StockMovement{}, fmt.Errorf("transfer stock: %w: movement must be transferring", application.ErrInvalidStockMovement)
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.StockLevel{}, domain.StockLevel{}, domain.StockMovement{}, fmt.Errorf("begin transfer stock transaction: %w", application.ErrStockLevelPersistence)
	}

	sourceStock, err := scanStockLevel(tx.QueryRowContext(ctx, transferStockLevelSQL, movement.ProductID, *movement.FromID, movement.Units))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.StockLevel{}, domain.StockLevel{}, domain.StockMovement{}, rollbackTransfer(tx, application.ErrInsufficientStock)
	}
	if err != nil {
		return domain.StockLevel{}, domain.StockLevel{}, domain.StockMovement{}, rollbackTransfer(tx, mapStockLevelWriteError(err))
	}

	destinationStock, err := scanStockLevel(tx.QueryRowContext(ctx, transferDestinationStockLevelSQL, movement.ProductID, *movement.ToID, movement.Units))
	if err != nil {
		return domain.StockLevel{}, domain.StockLevel{}, domain.StockMovement{}, rollbackTransfer(tx, mapStockLevelWriteError(err))
	}

	persistedMovement, err := scanStockMovement(tx.QueryRowContext(ctx, insertStockMovementSQL,
		movement.Type, movement.Units, movement.ProductID, movement.FromID, movement.ToID, movement.CreatedAt,
	))
	if err != nil {
		return domain.StockLevel{}, domain.StockLevel{}, domain.StockMovement{}, rollbackTransfer(tx, mapStockMovementWriteError(err))
	}
	if err := tx.Commit(); err != nil {
		return domain.StockLevel{}, domain.StockLevel{}, domain.StockMovement{}, rollbackTransfer(tx, fmt.Errorf("commit transfer stock transaction: %w", application.ErrStockMovementPersistence))
	}
	return sourceStock, destinationStock, persistedMovement, nil
}

func rollbackTransfer(tx *sql.Tx, operationErr error) error {
	if rollbackErr := tx.Rollback(); rollbackErr != nil && !errors.Is(rollbackErr, sql.ErrTxDone) {
		return fmt.Errorf("%w (rollback failed: %v)", operationErr, rollbackErr)
	}
	return operationErr
}
