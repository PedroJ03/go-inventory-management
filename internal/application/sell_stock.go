package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/pedroj/go-inventory-management/internal/domain"
)

var (
	ErrInvalidSellingInput = errors.New("invalid selling input")
	ErrInsufficientStock   = errors.New("insufficient stock")
)

// StockSellingPort owns the atomic persistence boundary for selling stock.
// Implementations must update stock and append movement history as one operation.
type StockSellingPort interface {
	Sell(ctx context.Context, movement domain.StockMovement) (domain.StockLevel, domain.StockMovement, error)
}

type SellStockInput struct {
	ProductID  uuid.UUID
	LocationID int64
	Units      int64
}

type SellStockResult struct {
	StockLevel domain.StockLevel
	Movement   domain.StockMovement
}

// SellStock validates references and delegates the state change to the
// atomic selling persistence boundary.
func SellStock(ctx context.Context, products ProductRepository, locations LocationRepository, selling StockSellingPort, input SellStockInput) (SellStockResult, error) {
	if input.ProductID == uuid.Nil || input.LocationID <= 0 || input.Units <= 0 {
		return SellStockResult{}, fmt.Errorf("%w: product, location, and positive units are required", ErrInvalidSellingInput)
	}

	product, err := products.FindByID(ctx, input.ProductID)
	if err != nil {
		return SellStockResult{}, err
	}
	if err := product.Validate(); err != nil {
		return SellStockResult{}, fmt.Errorf("validate selling product: %w", err)
	}

	location, err := locations.FindByID(ctx, input.LocationID)
	if err != nil {
		return SellStockResult{}, err
	}
	if err := location.Validate(); err != nil {
		return SellStockResult{}, fmt.Errorf("validate selling location: %w", err)
	}
	if !location.Active {
		return SellStockResult{}, ErrInactiveLocation
	}

	fromID := location.ID
	movement := domain.StockMovement{
		Type:      domain.MovementSelling,
		Units:     input.Units,
		ProductID: product.Identity.ID,
		FromID:    &fromID,
		CreatedAt: time.Now().UTC(),
	}
	if err := movement.Validate(); err != nil {
		return SellStockResult{}, fmt.Errorf("validate selling movement: %w", err)
	}

	stockLevel, persistedMovement, err := selling.Sell(ctx, movement)
	if err != nil {
		return SellStockResult{}, err
	}
	return SellStockResult{StockLevel: stockLevel, Movement: persistedMovement}, nil
}
