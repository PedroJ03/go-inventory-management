package application

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/pedroj/go-inventory-management/internal/domain"
)

type productRepositoryFake struct {
	created  []domain.Product
	createFn func(context.Context, domain.Product) error
}

func (f *productRepositoryFake) Create(ctx context.Context, product domain.Product) error {
	f.created = append(f.created, product)
	if f.createFn != nil {
		return f.createFn(ctx, product)
	}
	return nil
}

func (f *productRepositoryFake) FindByID(context.Context, uuid.UUID) (domain.Product, error) {
	return domain.Product{}, ErrProductNotFound
}

func (f *productRepositoryFake) FindByIdentity(context.Context, string, string, float64, domain.Unit) (domain.Product, error) {
	return domain.Product{}, ErrProductNotFound
}

func TestCreateProductPersistsGeneratedProduct(t *testing.T) {
	repository := &productRepositoryFake{}
	input := CreateProductInput{Name: "Milk", Brand: "Acme", Amount: 0.5, Unit: domain.UnitLiters}

	product, err := CreateProduct(context.Background(), repository, input)
	if err != nil {
		t.Fatalf("CreateProduct() error = %v", err)
	}
	if product.Identity.ID == uuid.Nil {
		t.Fatal("CreateProduct() generated a nil product ID")
	}
	if product.Identity.Name != input.Name || product.Identity.Brand != input.Brand || product.Identity.Amount != input.Amount || product.Identity.Unit != input.Unit {
		t.Fatalf("CreateProduct() product = %+v, want input attributes %+v", product.Identity, input)
	}
	if len(repository.created) != 1 || repository.created[0] != product {
		t.Fatalf("repository received %+v, want %+v", repository.created, product)
	}
}

func TestCreateProductRejectsInvalidInputBeforePersistence(t *testing.T) {
	repository := &productRepositoryFake{}

	_, err := CreateProduct(context.Background(), repository, CreateProductInput{Name: "Milk", Brand: "Acme", Amount: 2, Unit: domain.UnitEach})
	if !errors.Is(err, ErrInvalidProduct) {
		t.Fatalf("CreateProduct() error = %v, want invalid product error", err)
	}
	if len(repository.created) != 0 {
		t.Fatalf("repository received %d products, want none", len(repository.created))
	}
}

func TestCreateProductPropagatesRepositoryError(t *testing.T) {
	wantErr := ErrProductAlreadyExists
	repository := &productRepositoryFake{createFn: func(context.Context, domain.Product) error {
		return wantErr
	}}

	_, err := CreateProduct(context.Background(), repository, CreateProductInput{Name: "Milk", Brand: "Acme", Amount: 1, Unit: domain.UnitEach})
	if !errors.Is(err, wantErr) {
		t.Fatalf("CreateProduct() error = %v, want repository error %v", err, wantErr)
	}
}
