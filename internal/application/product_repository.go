package application

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/PedroJ03/go-inventory-management/internal/domain"
)

var (
	ErrInvalidProduct       = errors.New("invalid product")
	ErrProductNotFound      = errors.New("product not found")
	ErrProductAlreadyExists = errors.New("product already exists")
	ErrProductConstraint    = errors.New("product persistence constraint failed")
)

// ProductRepository is the product persistence boundary required by the
// application core. Product identity fields are immutable after creation.
type ProductRepository interface {
	Create(ctx context.Context, product domain.Product) error
	FindByID(ctx context.Context, id uuid.UUID) (domain.Product, error)
	FindByIdentity(ctx context.Context, name, brand string, amount float64, unit domain.Unit) (domain.Product, error)
}
