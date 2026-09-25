package application

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/pedroj/go-inventory-management/internal/domain"
)

var (
	ErrInvalidStockLevel     = errors.New("invalid stock level")
	ErrStockLevelNotFound    = errors.New("stock level not found")
	ErrStockLevelConstraint  = errors.New("stock level persistence constraint failed")
	ErrStockLevelPersistence = errors.New("stock level persistence failed")
)

// StockLevelRepository is the narrow stock-state persistence boundary needed
// by application workflows. It does not apply movement or lifecycle policy.
type StockLevelRepository interface {
	FindByProductAndLocation(ctx context.Context, productID uuid.UUID, locationID int64) (domain.StockLevel, error)
	Upsert(ctx context.Context, stockLevel domain.StockLevel) error
	HasPositiveStockAtLocation(ctx context.Context, locationID int64) (bool, error)
}
