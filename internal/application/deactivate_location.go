package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/PedroJ03/go-inventory-management/internal/domain"
)

var ErrLocationHasStock = errors.New("location has positive stock")

// DeactivateLocation deactivates a location only after confirming that it
// exists and has no positive stock. Zero-stock product-location assignments
// remain untouched.
func DeactivateLocation(ctx context.Context, locations LocationRepository, stockLevels StockLevelRepository, id int64) (domain.Location, error) {
	if id <= 0 {
		return domain.Location{}, fmt.Errorf("%w: location id must be positive", ErrInvalidLocation)
	}

	if _, err := locations.FindByID(ctx, id); err != nil {
		return domain.Location{}, err
	}

	hasPositiveStock, err := stockLevels.HasPositiveStockAtLocation(ctx, id)
	if err != nil {
		return domain.Location{}, err
	}
	if hasPositiveStock {
		return domain.Location{}, ErrLocationHasStock
	}

	return locations.Deactivate(ctx, id)
}
