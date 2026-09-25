package domain

import (
	"fmt"
	"math"
	"strings"

	"github.com/google/uuid"
)

type Unit string

const (
	UnitEach        Unit = "each"
	UnitMilliliters Unit = "milliliters"
	UnitLiters      Unit = "liters"
	UnitGrams       Unit = "grams"
	UnitKilograms   Unit = "kilograms"
)

// ProductIdentity contains the immutable identity and business attributes of a product.
type ProductIdentity struct {
	ID     uuid.UUID
	Name   string
	Brand  string
	Amount float64
	Unit   Unit
}

type Product struct {
	Identity ProductIdentity
}

type InventoryRecord = Product

func (p Product) Validate() error {
	if p.Identity.ID == uuid.Nil {
		return fmt.Errorf("product id must not be nil")
	}
	if strings.TrimSpace(p.Identity.Name) == "" || strings.TrimSpace(p.Identity.Brand) == "" {
		return fmt.Errorf("product name and brand must not be empty")
	}
	if math.IsNaN(p.Identity.Amount) || math.IsInf(p.Identity.Amount, 0) || p.Identity.Amount <= 0 {
		return fmt.Errorf("product amount must be greater than zero")
	}
	switch p.Identity.Unit {
	case UnitEach:
		if p.Identity.Amount != 1 {
			return fmt.Errorf("unit %q requires amount 1", p.Identity.Unit)
		}
	case UnitMilliliters, UnitLiters, UnitGrams, UnitKilograms:
	default:
		return fmt.Errorf("unsupported product unit %q", p.Identity.Unit)
	}
	return nil
}
