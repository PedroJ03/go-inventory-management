package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/PedroJ03/go-inventory-management/internal/domain"
)

var ErrInvalidTransferInput = errors.New("invalid transfer input")

// StockTransferringPort owns the atomic persistence boundary for transferring stock.
// Implementations must update both stock levels and append movement history as one operation.
type StockTransferringPort interface {
	Transfer(ctx context.Context, movement domain.StockMovement) (domain.StockLevel, domain.StockLevel, domain.StockMovement, error)
}

type TransferStockInput struct {
	ProductID             uuid.UUID
	SourceLocationID      int64
	DestinationLocationID int64
	Units                 int64
}

type TransferStockResult struct {
	SourceStock      domain.StockLevel
	DestinationStock domain.StockLevel
	Movement         domain.StockMovement
}

// TransferStock validates references and delegates the state change to the atomic transfer boundary.
func TransferStock(ctx context.Context, products ProductRepository, locations LocationRepository, transferring StockTransferringPort, input TransferStockInput) (TransferStockResult, error) {
	if input.ProductID == uuid.Nil || input.SourceLocationID <= 0 || input.DestinationLocationID <= 0 || input.Units <= 0 {
		return TransferStockResult{}, fmt.Errorf("%w: product, distinct locations, and positive units are required", ErrInvalidTransferInput)
	}
	if input.SourceLocationID == input.DestinationLocationID {
		return TransferStockResult{}, fmt.Errorf("%w: source and destination locations must differ", ErrInvalidTransferInput)
	}

	product, err := products.FindByID(ctx, input.ProductID)
	if err != nil {
		return TransferStockResult{}, err
	}
	if err := product.Validate(); err != nil {
		return TransferStockResult{}, fmt.Errorf("validate transfer product: %w", err)
	}

	source, err := locations.FindByID(ctx, input.SourceLocationID)
	if err != nil {
		return TransferStockResult{}, err
	}
	if err := source.Validate(); err != nil {
		return TransferStockResult{}, fmt.Errorf("validate transfer source location: %w", err)
	}
	if !source.Active {
		return TransferStockResult{}, ErrInactiveLocation
	}

	destination, err := locations.FindByID(ctx, input.DestinationLocationID)
	if err != nil {
		return TransferStockResult{}, err
	}
	if err := destination.Validate(); err != nil {
		return TransferStockResult{}, fmt.Errorf("validate transfer destination location: %w", err)
	}
	if !destination.Active {
		return TransferStockResult{}, ErrInactiveLocation
	}

	fromID, toID := source.ID, destination.ID
	movement := domain.StockMovement{
		Type:      domain.MovementTransferring,
		Units:     input.Units,
		ProductID: product.Identity.ID,
		FromID:    &fromID,
		ToID:      &toID,
		CreatedAt: time.Now().UTC(),
	}
	if err := movement.Validate(); err != nil {
		return TransferStockResult{}, fmt.Errorf("validate transfer movement: %w", err)
	}

	sourceStock, destinationStock, persistedMovement, err := transferring.Transfer(ctx, movement)
	if err != nil {
		return TransferStockResult{}, err
	}
	return TransferStockResult{SourceStock: sourceStock, DestinationStock: destinationStock, Movement: persistedMovement}, nil
}
