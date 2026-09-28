package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/PedroJ03/go-inventory-management/internal/domain"
)

type transferProductRepositoryFake struct {
	product domain.Product
	err     error
}

func (f transferProductRepositoryFake) FindByID(context.Context, uuid.UUID) (domain.Product, error) {
	return f.product, f.err
}
func (f transferProductRepositoryFake) Create(context.Context, domain.Product) error { return nil }
func (f transferProductRepositoryFake) FindByIdentity(context.Context, string, string, float64, domain.Unit) (domain.Product, error) {
	return domain.Product{}, ErrProductNotFound
}

type transferLocationRepositoryFake struct {
	locations map[int64]domain.Location
	err       error
}

func (f transferLocationRepositoryFake) FindByID(_ context.Context, id int64) (domain.Location, error) {
	if f.err != nil {
		return domain.Location{}, f.err
	}
	return f.locations[id], nil
}
func (f transferLocationRepositoryFake) Create(context.Context, domain.Location) (domain.Location, error) {
	return domain.Location{}, nil
}
func (f transferLocationRepositoryFake) Activate(context.Context, int64) (domain.Location, error) {
	return domain.Location{}, nil
}
func (f transferLocationRepositoryFake) Deactivate(context.Context, int64) (domain.Location, error) {
	return domain.Location{}, nil
}

type transferPortFake struct {
	source, destination domain.StockLevel
	movement            domain.StockMovement
	err                 error
}

func (f *transferPortFake) Transfer(_ context.Context, movement domain.StockMovement) (domain.StockLevel, domain.StockLevel, domain.StockMovement, error) {
	f.movement = movement
	return f.source, f.destination, movement, f.err
}

func transferFixtures() (uuid.UUID, domain.Product, domain.Location, domain.Location) {
	productID := uuid.New()
	return productID,
		domain.Product{Identity: domain.ProductIdentity{ID: productID, Name: "Milk", Brand: "Acme", Amount: 1, Unit: domain.UnitEach}},
		domain.Location{ID: 7, Name: "Source", Street: "Main", City: "Springfield", Active: true},
		domain.Location{ID: 8, Name: "Destination", Street: "Second", City: "Springfield", Active: true}
}

func TestTransferStockCreatesValidatedTransferMovement(t *testing.T) {
	productID, product, source, destination := transferFixtures()
	port := &transferPortFake{source: domain.StockLevel{ProductID: productID, LocationID: source.ID, Units: 3}, destination: domain.StockLevel{ProductID: productID, LocationID: destination.ID, Units: 5}}

	result, err := TransferStock(context.Background(), transferProductRepositoryFake{product: product}, transferLocationRepositoryFake{locations: map[int64]domain.Location{source.ID: source, destination.ID: destination}}, port, TransferStockInput{ProductID: productID, SourceLocationID: source.ID, DestinationLocationID: destination.ID, Units: 2})
	if err != nil {
		t.Fatalf("TransferStock() error = %v", err)
	}
	if result.Movement.Type != domain.MovementTransferring || result.Movement.Units != 2 || result.Movement.FromID == nil || result.Movement.ToID == nil {
		t.Fatalf("TransferStock() movement = %+v, want transfer movement", result.Movement)
	}
	if *result.Movement.FromID != source.ID || *result.Movement.ToID != destination.ID || result.Movement.CreatedAt.Location() != time.UTC {
		t.Fatalf("TransferStock() movement = %+v, want UTC source and destination", result.Movement)
	}
}

func TestTransferStockRejectsSameLocationWithoutAtomicPort(t *testing.T) {
	productID, product, source, _ := transferFixtures()
	port := &transferPortFake{}

	_, err := TransferStock(context.Background(), transferProductRepositoryFake{product: product}, transferLocationRepositoryFake{locations: map[int64]domain.Location{source.ID: source}}, port, TransferStockInput{ProductID: productID, SourceLocationID: source.ID, DestinationLocationID: source.ID, Units: 1})
	if !errors.Is(err, ErrInvalidTransferInput) {
		t.Fatalf("TransferStock() error = %v, want invalid input", err)
	}
	if port.movement.Units != 0 {
		t.Fatal("TransferStock() called atomic port for same location")
	}
}

func TestTransferStockRejectsInactiveLocationBeforeAtomicPort(t *testing.T) {
	productID, product, source, destination := transferFixtures()
	destination.Active = false
	port := &transferPortFake{}

	_, err := TransferStock(context.Background(), transferProductRepositoryFake{product: product}, transferLocationRepositoryFake{locations: map[int64]domain.Location{source.ID: source, destination.ID: destination}}, port, TransferStockInput{ProductID: productID, SourceLocationID: source.ID, DestinationLocationID: destination.ID, Units: 1})
	if !errors.Is(err, ErrInactiveLocation) {
		t.Fatalf("TransferStock() error = %v, want inactive location", err)
	}
	if port.movement.Units != 0 {
		t.Fatal("TransferStock() called atomic port for inactive location")
	}
}

func TestTransferStockPropagatesInsufficientSourceStock(t *testing.T) {
	productID, product, source, destination := transferFixtures()
	port := &transferPortFake{err: ErrInsufficientStock}

	_, err := TransferStock(context.Background(), transferProductRepositoryFake{product: product}, transferLocationRepositoryFake{locations: map[int64]domain.Location{source.ID: source, destination.ID: destination}}, port, TransferStockInput{ProductID: productID, SourceLocationID: source.ID, DestinationLocationID: destination.ID, Units: 1})
	if !errors.Is(err, ErrInsufficientStock) {
		t.Fatalf("TransferStock() error = %v, want insufficient stock", err)
	}
}
