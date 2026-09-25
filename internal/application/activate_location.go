package application

import (
	"context"
	"fmt"

	"github.com/pedroj/go-inventory-management/internal/domain"
)

// ActivateLocation marks a location as active through the persistence
// boundary. The repository remains responsible for the status transition.
func ActivateLocation(ctx context.Context, repository LocationRepository, id int64) (domain.Location, error) {
	if id <= 0 {
		return domain.Location{}, fmt.Errorf("%w: location id must be positive", ErrInvalidLocation)
	}

	return repository.Activate(ctx, id)
}
