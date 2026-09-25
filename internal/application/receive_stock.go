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
	ErrInvalidReceivingInput = errors.New("invalid receiving input")
	ErrInactiveLocation      = errors.New("location is inactive")
)

// StockReceivingPort owns the atomic persistence boundary for receiving stock.
// Implementations must update stock and append movement history as one operation.
type StockReceivingPort interface {
	Receive(ctx context.Context, movement domain.StockMovement) (domain.StockLevel, domain.StockMovement, error)
}

type ReceiveStockInput struct {
	ProductID  uuid.UUID
	LocationID int64
	Units      int64
}

type ReceiveStockResult struct {
	StockLevel domain.StockLevel
	Movement   domain.StockMovement
}

// ReceiveStock validates references and delegates the state change to the
// atomic receiving persistence boundary.
func ReceiveStock(ctx context.Context, products ProductRepository, locations LocationRepository, receiving StockReceivingPort, input ReceiveStockInput) (ReceiveStockResult, error) {
	if input.ProductID == uuid.Nil || input.LocationID <= 0 || input.Units <= 0 {
		return ReceiveStockResult{}, fmt.Errorf("%w: product, location, and positive units are required", ErrInvalidReceivingInput)
	}

	product, err := products.FindByID(ctx, input.ProductID)
	if err != nil {
		return ReceiveStockResult{}, err
	}
	if err := product.Validate(); err != nil {
		return ReceiveStockResult{}, fmt.Errorf("validate receiving product: %w", err)
	}

	location, err := locations.FindByID(ctx, input.LocationID)
	if err != nil {
		return ReceiveStockResult{}, err
	}
	if err := location.Validate(); err != nil {
		return ReceiveStockResult{}, fmt.Errorf("validate receiving location: %w", err)
	}
	if !location.Active {
		return ReceiveStockResult{}, ErrInactiveLocation
	}

	toID := location.ID
	movement := domain.StockMovement{
		Type:      domain.MovementReceiving,
		Units:     input.Units,
		ProductID: product.Identity.ID,
		ToID:      &toID,
		CreatedAt: time.Now().UTC(),
	}
	if err := movement.Validate(); err != nil {
		return ReceiveStockResult{}, fmt.Errorf("validate receiving movement: %w", err)
	}

	stockLevel, persistedMovement, err := receiving.Receive(ctx, movement)
	if err != nil {
		return ReceiveStockResult{}, err
	}
	return ReceiveStockResult{StockLevel: stockLevel, Movement: persistedMovement}, nil
}
