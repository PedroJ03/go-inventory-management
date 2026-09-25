package application

import (
	"context"
	"errors"

	"github.com/pedroj/go-inventory-management/internal/domain"
)

var (
	ErrInvalidStockMovement     = errors.New("invalid stock movement")
	ErrStockMovementConstraint  = errors.New("stock movement persistence constraint failed")
	ErrStockMovementPersistence = errors.New("stock movement persistence failed")
)

// StockMovementRepository is the narrow append-only persistence boundary
// needed by stock movement workflows. It does not coordinate stock changes;
// the receiving workflow will own that transaction boundary later.
type StockMovementRepository interface {
	Create(ctx context.Context, movement domain.StockMovement) (domain.StockMovement, error)
}
