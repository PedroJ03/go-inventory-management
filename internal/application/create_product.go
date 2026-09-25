package application

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/pedroj/go-inventory-management/internal/domain"
)

// CreateProductInput contains the caller-provided product attributes.
type CreateProductInput struct {
	Name   string
	Brand  string
	Amount float64
	Unit   domain.Unit
}

// CreateProduct creates and persists a product with an application-generated ID.
func CreateProduct(ctx context.Context, repository ProductRepository, input CreateProductInput) (domain.Product, error) {
	product := domain.Product{Identity: domain.ProductIdentity{
		ID:     uuid.New(),
		Name:   input.Name,
		Brand:  input.Brand,
		Amount: input.Amount,
		Unit:   input.Unit,
	}}

	if err := product.Validate(); err != nil {
		return domain.Product{}, fmt.Errorf("%w: %v", ErrInvalidProduct, err)
	}
	if err := repository.Create(ctx, product); err != nil {
		return domain.Product{}, err
	}
	return product, nil
}
