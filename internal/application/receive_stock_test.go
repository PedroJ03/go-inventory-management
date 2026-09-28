package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/PedroJ03/go-inventory-management/internal/domain"
)

type receivingProductRepositoryFake struct {
	product domain.Product
	err     error
}

func (f receivingProductRepositoryFake) FindByID(context.Context, uuid.UUID) (domain.Product, error) {
	return f.product, f.err
}
func (f receivingProductRepositoryFake) Create(context.Context, domain.Product) error { return nil }
func (f receivingProductRepositoryFake) FindByIdentity(context.Context, string, string, float64, domain.Unit) (domain.Product, error) {
	return domain.Product{}, ErrProductNotFound
}

type receivingLocationRepositoryFake struct {
	location domain.Location
	err      error
}

func (f receivingLocationRepositoryFake) FindByID(context.Context, int64) (domain.Location, error) {
	return f.location, f.err
}
func (f receivingLocationRepositoryFake) Create(context.Context, domain.Location) (domain.Location, error) {
	return domain.Location{}, nil
}
func (f receivingLocationRepositoryFake) Activate(context.Context, int64) (domain.Location, error) {
	return domain.Location{}, nil
}
func (f receivingLocationRepositoryFake) Deactivate(context.Context, int64) (domain.Location, error) {
	return domain.Location{}, nil
}

type receivingPortFake struct {
	stock    domain.StockLevel
	movement domain.StockMovement
	err      error
}

func (f *receivingPortFake) Receive(_ context.Context, movement domain.StockMovement) (domain.StockLevel, domain.StockMovement, error) {
	f.movement = movement
	return f.stock, f.movement, f.err
}

func validReceivingFixtures() (uuid.UUID, domain.Product, domain.Location) {
	productID := uuid.New()
	return productID, domain.Product{Identity: domain.ProductIdentity{ID: productID, Name: "Milk", Brand: "Acme", Amount: 1, Unit: domain.UnitEach}}, domain.Location{ID: 7, Name: "Warehouse", Street: "Main", City: "Springfield", Active: true}
}

func TestReceiveStockCreatesValidatedReceivingMovement(t *testing.T) {
	productID, product, location := validReceivingFixtures()
	port := &receivingPortFake{stock: domain.StockLevel{ProductID: productID, LocationID: location.ID, Units: 12}}

	result, err := ReceiveStock(context.Background(), receivingProductRepositoryFake{product: product}, receivingLocationRepositoryFake{location: location}, port, ReceiveStockInput{ProductID: productID, LocationID: location.ID, Units: 12})
	if err != nil {
		t.Fatalf("ReceiveStock() error = %v", err)
	}
	if result.StockLevel.Units != 12 || result.Movement.Type != domain.MovementReceiving || result.Movement.Units != 12 {
		t.Fatalf("ReceiveStock() result = %+v, want receiving result", result)
	}
	if result.Movement.CreatedAt.Location() != time.UTC || result.Movement.ToID == nil || *result.Movement.ToID != location.ID {
		t.Fatalf("ReceiveStock() movement = %+v, want UTC destination movement", result.Movement)
	}
}

func TestReceiveStockRejectsInactiveLocationBeforeAtomicPort(t *testing.T) {
	productID, product, location := validReceivingFixtures()
	location.Active = false
	port := &receivingPortFake{}

	_, err := ReceiveStock(context.Background(), receivingProductRepositoryFake{product: product}, receivingLocationRepositoryFake{location: location}, port, ReceiveStockInput{ProductID: productID, LocationID: location.ID, Units: 1})
	if !errors.Is(err, ErrInactiveLocation) {
		t.Fatalf("ReceiveStock() error = %v, want inactive location error", err)
	}
	if port.movement.Units != 0 {
		t.Fatal("ReceiveStock() called atomic port for inactive location")
	}
}

func TestReceiveStockRejectsNonPositiveUnitsWithoutLookups(t *testing.T) {
	productID, product, location := validReceivingFixtures()
	port := &receivingPortFake{}

	_, err := ReceiveStock(context.Background(), receivingProductRepositoryFake{product: product}, receivingLocationRepositoryFake{location: location}, port, ReceiveStockInput{ProductID: productID, LocationID: location.ID, Units: 0})
	if !errors.Is(err, ErrInvalidReceivingInput) {
		t.Fatalf("ReceiveStock() error = %v, want invalid input error", err)
	}
}
