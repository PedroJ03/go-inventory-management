package application

import (
	"context"
	"errors"
	"testing"

	"github.com/PedroJ03/go-inventory-management/internal/domain"
)

type locationRepositoryFake struct {
	created        []domain.Location
	activatedIDs   []int64
	createResult   domain.Location
	activateResult domain.Location
	createErr      error
	activateErr    error
}

func (f *locationRepositoryFake) Create(_ context.Context, location domain.Location) (domain.Location, error) {
	f.created = append(f.created, location)
	if f.createErr != nil {
		return domain.Location{}, f.createErr
	}
	return f.createResult, nil
}

func (f *locationRepositoryFake) FindByID(context.Context, int64) (domain.Location, error) {
	return domain.Location{}, ErrLocationNotFound
}

func (f *locationRepositoryFake) Activate(_ context.Context, id int64) (domain.Location, error) {
	f.activatedIDs = append(f.activatedIDs, id)
	if f.activateErr != nil {
		return domain.Location{}, f.activateErr
	}
	return f.activateResult, nil
}

func (f *locationRepositoryFake) Deactivate(context.Context, int64) (domain.Location, error) {
	return domain.Location{}, ErrLocationNotFound
}

func TestCreateLocationPersistsDatabaseAssignedLocation(t *testing.T) {
	persisted := domain.Location{ID: 7, Name: "Warehouse", Street: "Main Street 1", City: "Springfield", Active: true}
	repository := &locationRepositoryFake{createResult: persisted}
	input := CreateLocationInput{Name: "Warehouse", Street: "Main Street 1", City: "Springfield"}

	got, err := CreateLocation(context.Background(), repository, input)
	if err != nil {
		t.Fatalf("CreateLocation() error = %v", err)
	}
	if got != persisted {
		t.Fatalf("CreateLocation() = %+v, want persisted location %+v", got, persisted)
	}
	if len(repository.created) != 1 {
		t.Fatalf("repository received %d locations, want 1", len(repository.created))
	}
	if repository.created[0].ID != 0 {
		t.Fatalf("repository received location ID %d, want zero for database assignment", repository.created[0].ID)
	}
	if !repository.created[0].Active {
		t.Fatal("repository received inactive location, want new locations to start active")
	}
}

func TestCreateLocationRejectsInvalidInputBeforePersistence(t *testing.T) {
	repository := &locationRepositoryFake{}

	_, err := CreateLocation(context.Background(), repository, CreateLocationInput{Name: "Warehouse", Street: "", City: "Springfield"})
	if !errors.Is(err, ErrInvalidLocation) {
		t.Fatalf("CreateLocation() error = %v, want invalid location error", err)
	}
	if len(repository.created) != 0 {
		t.Fatalf("repository received %d locations, want none", len(repository.created))
	}
}

func TestCreateLocationPropagatesRepositoryError(t *testing.T) {
	wantErr := ErrLocationConstraint
	repository := &locationRepositoryFake{createErr: wantErr}

	_, err := CreateLocation(context.Background(), repository, CreateLocationInput{Name: "Warehouse", Street: "Main Street 1", City: "Springfield"})
	if !errors.Is(err, wantErr) {
		t.Fatalf("CreateLocation() error = %v, want repository error %v", err, wantErr)
	}
}

func TestActivateLocationDelegatesPositiveID(t *testing.T) {
	persisted := domain.Location{ID: 7, Name: "Warehouse", Street: "Main Street 1", City: "Springfield", Active: true}
	repository := &locationRepositoryFake{activateResult: persisted}

	got, err := ActivateLocation(context.Background(), repository, 7)
	if err != nil {
		t.Fatalf("ActivateLocation() error = %v", err)
	}
	if got != persisted {
		t.Fatalf("ActivateLocation() = %+v, want persisted location %+v", got, persisted)
	}
	if len(repository.activatedIDs) != 1 || repository.activatedIDs[0] != 7 {
		t.Fatalf("repository activated IDs = %v, want [7]", repository.activatedIDs)
	}
}

func TestActivateLocationRejectsNonPositiveIDBeforePersistence(t *testing.T) {
	for _, id := range []int64{0, -1} {
		t.Run("invalid id", func(t *testing.T) {
			repository := &locationRepositoryFake{}

			_, err := ActivateLocation(context.Background(), repository, id)
			if !errors.Is(err, ErrInvalidLocation) {
				t.Fatalf("ActivateLocation(%d) error = %v, want invalid location error", id, err)
			}
			if len(repository.activatedIDs) != 0 {
				t.Fatalf("repository received activation for %d", id)
			}
		})
	}
}

func TestActivateLocationPropagatesRepositoryError(t *testing.T) {
	wantErr := ErrLocationNotFound
	repository := &locationRepositoryFake{activateErr: wantErr}

	_, err := ActivateLocation(context.Background(), repository, 7)
	if !errors.Is(err, wantErr) {
		t.Fatalf("ActivateLocation() error = %v, want repository error %v", err, wantErr)
	}
}
