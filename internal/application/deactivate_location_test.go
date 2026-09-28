package application

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/PedroJ03/go-inventory-management/internal/domain"
)

type deactivateLocationRepositoryFake struct {
	findResult       domain.Location
	findErr          error
	deactivateResult domain.Location
	deactivateErr    error
	findIDs          []int64
	deactivateIDs    []int64
}

func (f *deactivateLocationRepositoryFake) Create(context.Context, domain.Location) (domain.Location, error) {
	return domain.Location{}, nil
}

func (f *deactivateLocationRepositoryFake) FindByID(_ context.Context, id int64) (domain.Location, error) {
	f.findIDs = append(f.findIDs, id)
	if f.findErr != nil {
		return domain.Location{}, f.findErr
	}
	return f.findResult, nil
}

func (f *deactivateLocationRepositoryFake) Activate(context.Context, int64) (domain.Location, error) {
	return domain.Location{}, nil
}

func (f *deactivateLocationRepositoryFake) Deactivate(_ context.Context, id int64) (domain.Location, error) {
	f.deactivateIDs = append(f.deactivateIDs, id)
	if f.deactivateErr != nil {
		return domain.Location{}, f.deactivateErr
	}
	return f.deactivateResult, nil
}

type deactivateStockLevelRepositoryFake struct {
	hasPositiveStock bool
	checkErr         error
	checkedIDs       []int64
	upserted         []domain.StockLevel
}

func (f *deactivateStockLevelRepositoryFake) FindByProductAndLocation(context.Context, uuid.UUID, int64) (domain.StockLevel, error) {
	return domain.StockLevel{}, nil
}

func (f *deactivateStockLevelRepositoryFake) Upsert(_ context.Context, stockLevel domain.StockLevel) error {
	f.upserted = append(f.upserted, stockLevel)
	return nil
}

func (f *deactivateStockLevelRepositoryFake) HasPositiveStockAtLocation(_ context.Context, id int64) (bool, error) {
	f.checkedIDs = append(f.checkedIDs, id)
	if f.checkErr != nil {
		return false, f.checkErr
	}
	return f.hasPositiveStock, nil
}

func TestDeactivateLocationRejectsInvalidIDWithoutRepositoryCalls(t *testing.T) {
	for _, id := range []int64{0, -1} {
		t.Run("invalid id", func(t *testing.T) {
			locations := &deactivateLocationRepositoryFake{}
			stockLevels := &deactivateStockLevelRepositoryFake{}

			_, err := DeactivateLocation(context.Background(), locations, stockLevels, id)
			if !errors.Is(err, ErrInvalidLocation) {
				t.Fatalf("DeactivateLocation(%d) error = %v, want invalid location error", id, err)
			}
			assertNoDeactivateLocationCalls(t, locations, stockLevels)
		})
	}
}

func TestDeactivateLocationRejectsMissingLocationBeforeStockCheck(t *testing.T) {
	locations := &deactivateLocationRepositoryFake{findErr: ErrLocationNotFound}
	stockLevels := &deactivateStockLevelRepositoryFake{}

	_, err := DeactivateLocation(context.Background(), locations, stockLevels, 7)
	if !errors.Is(err, ErrLocationNotFound) {
		t.Fatalf("DeactivateLocation() error = %v, want location not found error", err)
	}
	if len(locations.findIDs) != 1 || locations.findIDs[0] != 7 {
		t.Fatalf("location find IDs = %v, want [7]", locations.findIDs)
	}
	if len(locations.deactivateIDs) != 0 || len(stockLevels.checkedIDs) != 0 {
		t.Fatalf("unexpected repository calls: deactivations=%v stock checks=%v", locations.deactivateIDs, stockLevels.checkedIDs)
	}
}

func TestDeactivateLocationRejectsPositiveStockWithoutDeactivation(t *testing.T) {
	locations := &deactivateLocationRepositoryFake{findResult: validLocation(7)}
	stockLevels := &deactivateStockLevelRepositoryFake{hasPositiveStock: true}

	_, err := DeactivateLocation(context.Background(), locations, stockLevels, 7)
	if !errors.Is(err, ErrLocationHasStock) {
		t.Fatalf("DeactivateLocation() error = %v, want positive-stock error", err)
	}
	if len(stockLevels.checkedIDs) != 1 || stockLevels.checkedIDs[0] != 7 {
		t.Fatalf("stock check IDs = %v, want [7]", stockLevels.checkedIDs)
	}
	if len(locations.deactivateIDs) != 0 {
		t.Fatalf("location deactivation IDs = %v, want none", locations.deactivateIDs)
	}
}

func TestDeactivateLocationDelegatesZeroStockAndReturnsPersistedResult(t *testing.T) {
	persisted := validLocation(7)
	persisted.Active = false
	locations := &deactivateLocationRepositoryFake{
		findResult:       validLocation(7),
		deactivateResult: persisted,
	}
	stockLevels := &deactivateStockLevelRepositoryFake{}

	got, err := DeactivateLocation(context.Background(), locations, stockLevels, 7)
	if err != nil {
		t.Fatalf("DeactivateLocation() error = %v", err)
	}
	if got != persisted {
		t.Fatalf("DeactivateLocation() = %+v, want persisted location %+v", got, persisted)
	}
	if len(locations.deactivateIDs) != 1 || locations.deactivateIDs[0] != 7 {
		t.Fatalf("location deactivation IDs = %v, want [7]", locations.deactivateIDs)
	}
}

func TestDeactivateLocationPropagatesStockCheckErrorWithoutDeactivation(t *testing.T) {
	wantErr := errors.New("stock repository unavailable")
	locations := &deactivateLocationRepositoryFake{findResult: validLocation(7)}
	stockLevels := &deactivateStockLevelRepositoryFake{checkErr: wantErr}

	_, err := DeactivateLocation(context.Background(), locations, stockLevels, 7)
	if !errors.Is(err, wantErr) {
		t.Fatalf("DeactivateLocation() error = %v, want stock repository error", err)
	}
	if len(locations.deactivateIDs) != 0 {
		t.Fatalf("location deactivation IDs = %v, want none", locations.deactivateIDs)
	}
}

func TestDeactivateLocationPropagatesDeactivationError(t *testing.T) {
	wantErr := errors.New("location repository unavailable")
	locations := &deactivateLocationRepositoryFake{findResult: validLocation(7), deactivateErr: wantErr}
	stockLevels := &deactivateStockLevelRepositoryFake{}

	_, err := DeactivateLocation(context.Background(), locations, stockLevels, 7)
	if !errors.Is(err, wantErr) {
		t.Fatalf("DeactivateLocation() error = %v, want location repository error", err)
	}
}

func TestDeactivateLocationDoesNotMutateStockAssignments(t *testing.T) {
	locations := &deactivateLocationRepositoryFake{findResult: validLocation(7), deactivateResult: validLocation(7)}
	stockLevels := &deactivateStockLevelRepositoryFake{}

	if _, err := DeactivateLocation(context.Background(), locations, stockLevels, 7); err != nil {
		t.Fatalf("DeactivateLocation() error = %v", err)
	}
	if len(stockLevels.checkedIDs) != 1 || len(stockLevels.upserted) != 0 {
		t.Fatalf("stock repository calls: checks=%v upserts=%v, want one check and no upserts", stockLevels.checkedIDs, stockLevels.upserted)
	}
}

func assertNoDeactivateLocationCalls(t *testing.T, locations *deactivateLocationRepositoryFake, stockLevels *deactivateStockLevelRepositoryFake) {
	t.Helper()
	if len(locations.findIDs) != 0 || len(locations.deactivateIDs) != 0 || len(stockLevels.checkedIDs) != 0 {
		t.Fatalf("unexpected repository calls: finds=%v deactivations=%v stock checks=%v", locations.findIDs, locations.deactivateIDs, stockLevels.checkedIDs)
	}
}

func validLocation(id int64) domain.Location {
	return domain.Location{ID: id, Name: "Warehouse", Street: "Main Street 1", City: "Springfield", Active: true}
}
