package domain

import (
	"math"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestProductValidate(t *testing.T) {
	id := uuid.New()
	tests := []struct {
		name    string
		product Product
		valid   bool
	}{
		{"valid count product", Product{Identity: ProductIdentity{ID: id, Name: "milk", Brand: "acme", Amount: 1, Unit: UnitEach}}, true},
		{"valid fractional volume product", Product{Identity: ProductIdentity{ID: id, Name: "milk", Brand: "acme", Amount: 0.5, Unit: UnitLiters}}, true},
		{"fractional count rejected", Product{Identity: ProductIdentity{ID: id, Name: "milk", Brand: "acme", Amount: 0.5, Unit: UnitEach}}, false},
		{"zero package amount rejected", Product{Identity: ProductIdentity{ID: id, Name: "milk", Brand: "acme", Amount: 0, Unit: UnitLiters}}, false},
		{"nan package amount rejected", Product{Identity: ProductIdentity{ID: id, Name: "milk", Brand: "acme", Amount: math.NaN(), Unit: UnitLiters}}, false},
		{"unknown unit rejected", Product{Identity: ProductIdentity{ID: id, Name: "milk", Brand: "acme", Amount: 1, Unit: "packs"}}, false},
		{"blank name rejected", Product{Identity: ProductIdentity{ID: id, Name: " ", Brand: "acme", Amount: 1, Unit: UnitEach}}, false},
		{"blank brand rejected", Product{Identity: ProductIdentity{ID: id, Name: "milk", Brand: "\t", Amount: 1, Unit: UnitEach}}, false},
		{"nil identity rejected", Product{Identity: ProductIdentity{Name: "milk", Brand: "acme", Amount: 1, Unit: UnitEach}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.product.Validate() == nil; got != tt.valid {
				t.Fatalf("Validate() success = %v, want %v", got, tt.valid)
			}
		})
	}
}

func TestLocationValidate(t *testing.T) {
	tests := []struct {
		name     string
		location Location
		valid    bool
	}{
		{"database-assigned id", Location{Name: "Main", Street: "1 Main St", City: "Springfield", Active: true}, true},
		{"persisted id", Location{ID: 7, Name: "Main", Street: "1 Main St", City: "Springfield"}, true},
		{"negative id rejected", Location{ID: -1, Name: "Main", Street: "1 Main St", City: "Springfield"}, false},
		{"blank name rejected", Location{Street: "1 Main St", City: "Springfield"}, false},
		{"blank street rejected", Location{Name: "Main", City: "Springfield"}, false},
		{"blank city rejected", Location{Name: "Main", Street: "1 Main St"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.location.Validate() == nil; got != tt.valid {
				t.Fatalf("Validate() success = %v, want %v", got, tt.valid)
			}
		})
	}
}

func TestStockLevelValidate(t *testing.T) {
	valid := StockLevel{ProductID: uuid.New(), LocationID: 1, Units: 0}
	if err := valid.Validate(); err != nil {
		t.Fatalf("Validate() error = %v, want nil for zero stock", err)
	}

	tests := []StockLevel{
		{LocationID: 1, Units: 1},
		{ProductID: uuid.New(), Units: 1},
		{ProductID: uuid.New(), LocationID: 0, Units: 1},
		{ProductID: uuid.New(), LocationID: 1, Units: -1},
	}
	for _, level := range tests {
		if err := level.Validate(); err == nil {
			t.Fatalf("Validate() = nil for invalid stock level %+v", level)
		}
	}
}

func TestStockLevelRejectsNegativeUnits(t *testing.T) {
	level := StockLevel{ProductID: uuid.New(), LocationID: 1, Units: -1}
	if err := level.Validate(); err == nil {
		t.Fatal("Validate() error = nil, want negative-units error")
	}
}

func TestStockMovementRequiresTypeSpecificLocations(t *testing.T) {
	from, to := int64(1), int64(2)
	movement := StockMovement{Type: MovementReceiving, Units: 1, ProductID: uuid.New(), FromID: &from, ToID: &to}
	if err := movement.Validate(); err == nil {
		t.Fatal("Validate() error = nil, want receiving location error")
	}
}

func TestStockMovementValidate(t *testing.T) {
	createdAt := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	from, to := int64(1), int64(2)
	tests := []struct {
		name     string
		movement StockMovement
		valid    bool
	}{
		{"receiving requires destination only", StockMovement{Type: MovementReceiving, Units: 1, ProductID: uuid.New(), ToID: &to, CreatedAt: createdAt}, true},
		{"selling requires source only", StockMovement{Type: MovementSelling, Units: 1, ProductID: uuid.New(), FromID: &from, CreatedAt: createdAt}, true},
		{"transferring requires both locations", StockMovement{Type: MovementTransferring, Units: 1, ProductID: uuid.New(), FromID: &from, ToID: &to, CreatedAt: createdAt}, true},
		{"unknown type rejected", StockMovement{Type: "adjusting", Units: 1, ProductID: uuid.New(), ToID: &to, CreatedAt: createdAt}, false},
		{"zero units rejected", StockMovement{Type: MovementReceiving, ProductID: uuid.New(), ToID: &to, CreatedAt: createdAt}, false},
		{"nil timestamp rejected", StockMovement{Type: MovementReceiving, Units: 1, ProductID: uuid.New(), ToID: &to}, false},
		{"same locations rejected", StockMovement{Type: MovementTransferring, Units: 1, ProductID: uuid.New(), FromID: &from, ToID: &from, CreatedAt: createdAt}, false},
		{"non-positive location rejected", StockMovement{Type: MovementReceiving, Units: 1, ProductID: uuid.New(), ToID: new(int64), CreatedAt: createdAt}, false},
		{"negative movement id rejected", StockMovement{ID: -1, Type: MovementReceiving, Units: 1, ProductID: uuid.New(), ToID: &to, CreatedAt: createdAt}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.movement.Validate() == nil; got != tt.valid {
				t.Fatalf("Validate() success = %v, want %v", got, tt.valid)
			}
		})
	}
}
