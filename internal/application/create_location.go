package application

import (
	"context"
	"fmt"

	"github.com/PedroJ03/go-inventory-management/internal/domain"
)

// CreateLocationInput contains the caller-provided location attributes.
type CreateLocationInput struct {
	Name   string
	Street string
	City   string
}

// CreateLocation validates and persists a new location. The database assigns
// the location ID, so the repository receives a zero-valued ID.
func CreateLocation(ctx context.Context, repository LocationRepository, input CreateLocationInput) (domain.Location, error) {
	location := domain.Location{
		Name:   input.Name,
		Street: input.Street,
		City:   input.City,
		Active: true,
	}

	if err := location.Validate(); err != nil {
		return domain.Location{}, fmt.Errorf("%w: %v", ErrInvalidLocation, err)
	}

	persisted, err := repository.Create(ctx, location)
	if err != nil {
		return domain.Location{}, err
	}
	return persisted, nil
}
