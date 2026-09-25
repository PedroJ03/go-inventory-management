package domain

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type StockLevel struct {
	ProductID  uuid.UUID
	LocationID int64
	Units      int64
}

func (s StockLevel) Validate() error {
	if s.ProductID == uuid.Nil || s.LocationID <= 0 {
		return fmt.Errorf("stock level requires product and location identities")
	}
	if s.Units < 0 {
		return fmt.Errorf("stock units must not be negative")
	}
	return nil
}

type MovementType string

const (
	MovementReceiving    MovementType = "receiving"
	MovementSelling      MovementType = "selling"
	MovementTransferring MovementType = "transferring"
)

type StockMovement struct {
	ID        int64
	Type      MovementType
	Units     int64
	ProductID uuid.UUID
	FromID    *int64
	ToID      *int64
	CreatedAt time.Time
}

func (m StockMovement) Validate() error {
	if m.ID < 0 {
		return fmt.Errorf("stock movement id must not be negative")
	}
	if m.Type != MovementReceiving && m.Type != MovementSelling && m.Type != MovementTransferring {
		return fmt.Errorf("unsupported stock movement type %q", m.Type)
	}
	if m.Units <= 0 || m.ProductID == uuid.Nil {
		return fmt.Errorf("stock movement requires a positive unit count and product identity")
	}
	if m.CreatedAt.IsZero() {
		return fmt.Errorf("stock movement created-at timestamp must be set")
	}
	if m.Type == MovementReceiving && (m.FromID != nil || m.ToID == nil) {
		return fmt.Errorf("receiving requires only a destination location")
	}
	if m.Type == MovementSelling && (m.FromID == nil || m.ToID != nil) {
		return fmt.Errorf("selling requires only a source location")
	}
	if m.Type == MovementTransferring && (m.FromID == nil || m.ToID == nil) {
		return fmt.Errorf("transferring requires source and destination locations")
	}
	if (m.FromID != nil && *m.FromID <= 0) || (m.ToID != nil && *m.ToID <= 0) {
		return fmt.Errorf("stock movement locations must have positive identities")
	}
	if m.FromID != nil && m.ToID != nil && *m.FromID == *m.ToID {
		return fmt.Errorf("source and destination locations must differ")
	}
	return nil
}
