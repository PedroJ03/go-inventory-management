package application

import (
	"context"
	"errors"

	"github.com/pedroj/go-inventory-management/internal/domain"
)

var (
	ErrInvalidLocation    = errors.New("invalid location")
	ErrLocationNotFound   = errors.New("location not found")
	ErrLocationConstraint = errors.New("location persistence constraint failed")
)

// LocationRepository is the location persistence boundary required by the
// application core. Location descriptive fields are immutable after creation.
type LocationRepository interface {
	Create(ctx context.Context, location domain.Location) (domain.Location, error)
	FindByID(ctx context.Context, id int64) (domain.Location, error)
	Activate(ctx context.Context, id int64) (domain.Location, error)
	Deactivate(ctx context.Context, id int64) (domain.Location, error)
}
