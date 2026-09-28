package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/PedroJ03/go-inventory-management/internal/domain"
)

type sellingPortFake struct {
	stock    domain.StockLevel
	movement domain.StockMovement
	err      error
}

func (f *sellingPortFake) Sell(_ context.Context, movement domain.StockMovement) (domain.StockLevel, domain.StockMovement, error) {
	f.movement = movement
	return f.stock, f.movement, f.err
}

func TestSellStockCreatesValidatedSellingMovement(t *testing.T) {
	productID := uuid.New()
	product := domain.Product{Identity: domain.ProductIdentity{ID: productID, Name: "Milk", Brand: "Acme", Amount: 1, Unit: domain.UnitEach}}
	location := domain.Location{ID: 7, Name: "Warehouse", Street: "Main", City: "Springfield", Active: true}
	port := &sellingPortFake{stock: domain.StockLevel{ProductID: productID, LocationID: location.ID, Units: 3}}

	result, err := SellStock(context.Background(), receivingProductRepositoryFake{product: product}, receivingLocationRepositoryFake{location: location}, port, SellStockInput{ProductID: productID, LocationID: location.ID, Units: 2})
	if err != nil {
		t.Fatalf("SellStock() error = %v", err)
	}
	if result.StockLevel.Units != 3 || result.Movement.Type != domain.MovementSelling || result.Movement.Units != 2 {
		t.Fatalf("SellStock() result = %+v, want selling result", result)
	}
	if result.Movement.CreatedAt.Location() != time.UTC || result.Movement.FromID == nil || *result.Movement.FromID != location.ID || result.Movement.ToID != nil {
		t.Fatalf("SellStock() movement = %+v, want UTC source movement", result.Movement)
	}
}

func TestSellStockRejectsInactiveLocationBeforeAtomicPort(t *testing.T) {
	productID := uuid.New()
	product := domain.Product{Identity: domain.ProductIdentity{ID: productID, Name: "Milk", Brand: "Acme", Amount: 1, Unit: domain.UnitEach}}
	location := domain.Location{ID: 7, Name: "Warehouse", Street: "Main", City: "Springfield", Active: false}
	port := &sellingPortFake{}

	_, err := SellStock(context.Background(), receivingProductRepositoryFake{product: product}, receivingLocationRepositoryFake{location: location}, port, SellStockInput{ProductID: productID, LocationID: location.ID, Units: 1})
	if !errors.Is(err, ErrInactiveLocation) {
		t.Fatalf("SellStock() error = %v, want inactive location error", err)
	}
	if port.movement.Units != 0 {
		t.Fatal("SellStock() called atomic port for inactive location")
	}
}

func TestSellStockPropagatesInsufficientStock(t *testing.T) {
	productID := uuid.New()
	product := domain.Product{Identity: domain.ProductIdentity{ID: productID, Name: "Milk", Brand: "Acme", Amount: 1, Unit: domain.UnitEach}}
	location := domain.Location{ID: 7, Name: "Warehouse", Street: "Main", City: "Springfield", Active: true}
	port := &sellingPortFake{err: ErrInsufficientStock}

	_, err := SellStock(context.Background(), receivingProductRepositoryFake{product: product}, receivingLocationRepositoryFake{location: location}, port, SellStockInput{ProductID: productID, LocationID: location.ID, Units: 1})
	if !errors.Is(err, ErrInsufficientStock) {
		t.Fatalf("SellStock() error = %v, want insufficient stock error", err)
	}
}
