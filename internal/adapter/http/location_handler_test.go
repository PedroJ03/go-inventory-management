package httpadapter

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/PedroJ03/go-inventory-management/internal/domain"
)

type locationRepositoryFake struct {
	created          []domain.Location
	findResult       domain.Location
	findErr          error
	activatedIDs     []int64
	activateResult   domain.Location
	deactivatedIDs   []int64
	deactivateResult domain.Location
	createErr        error
}

func (f *locationRepositoryFake) Create(_ context.Context, location domain.Location) (domain.Location, error) {
	f.created = append(f.created, location)
	if f.createErr != nil {
		return domain.Location{}, f.createErr
	}
	return f.createdLocation(), nil
}

func (f *locationRepositoryFake) FindByID(_ context.Context, _ int64) (domain.Location, error) {
	if f.findErr != nil {
		return domain.Location{}, f.findErr
	}
	return f.findResult, nil
}

func (f *locationRepositoryFake) Activate(_ context.Context, id int64) (domain.Location, error) {
	f.activatedIDs = append(f.activatedIDs, id)
	return f.activateResult, nil
}

func (f *locationRepositoryFake) Deactivate(_ context.Context, id int64) (domain.Location, error) {
	f.deactivatedIDs = append(f.deactivatedIDs, id)
	return f.deactivateResult, nil
}

func (f *locationRepositoryFake) createdLocation() domain.Location {
	location := f.created[len(f.created)-1]
	location.ID = 7
	return location
}

type stockLevelRepositoryFake struct {
	hasPositiveStock bool
}

func (f *stockLevelRepositoryFake) FindByProductAndLocation(context.Context, uuid.UUID, int64) (domain.StockLevel, error) {
	return domain.StockLevel{}, nil
}

func (f *stockLevelRepositoryFake) Upsert(context.Context, domain.StockLevel) error { return nil }

func (f *stockLevelRepositoryFake) HasPositiveStockAtLocation(context.Context, int64) (bool, error) {
	return f.hasPositiveStock, nil
}

func newLocationHandlerForTest(t *testing.T, locations *locationRepositoryFake, stockLevels *stockLevelRepositoryFake) *LocationHandler {
	t.Helper()
	handler, err := NewLocationHandler(locations, stockLevels)
	if err != nil {
		t.Fatal(err)
	}
	return handler
}

func TestLocationHandlerServesCreationForm(t *testing.T) {
	handler := newLocationHandlerForTest(t, &locationRepositoryFake{}, &stockLevelRepositoryFake{})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/locations/new", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("GET status = %d, want %d", response.Code, http.StatusOK)
	}
	for _, expected := range []string{"Create location", `name="name"`, `name="street"`, `name="city"`} {
		if !strings.Contains(response.Body.String(), expected) {
			t.Errorf("GET body does not contain %q", expected)
		}
	}
}

func TestLocationHandlerCreatesLocation(t *testing.T) {
	repository := &locationRepositoryFake{}
	handler := newLocationHandlerForTest(t, repository, &stockLevelRepositoryFake{})
	request := httptest.NewRequest(http.MethodPost, "/locations", strings.NewReader("name=Warehouse&street=Main+Street+1&city=Springfield"))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("POST status = %d, want %d", response.Code, http.StatusCreated)
	}
	if len(repository.created) != 1 || repository.created[0].Name != "Warehouse" || !repository.created[0].Active {
		t.Fatalf("repository received %+v, want one active location", repository.created)
	}
	if !strings.Contains(response.Body.String(), "Location created successfully") {
		t.Fatal("creation response does not show success message")
	}
}

func TestLocationHandlerRejectsInvalidCreationWithoutPersistence(t *testing.T) {
	repository := &locationRepositoryFake{}
	handler := newLocationHandlerForTest(t, repository, &stockLevelRepositoryFake{})
	request := httptest.NewRequest(http.MethodPost, "/locations", strings.NewReader("name=Warehouse&street=&city=Springfield"))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("invalid POST status = %d, want %d", response.Code, http.StatusBadRequest)
	}
	if len(repository.created) != 0 {
		t.Fatalf("repository received %d locations, want none", len(repository.created))
	}
	if !strings.Contains(response.Body.String(), "name, street, and city") {
		t.Fatal("invalid response does not explain required location fields")
	}
}

func TestLocationHandlerActivatesLocation(t *testing.T) {
	repository := &locationRepositoryFake{activateResult: testLocation(7, true)}
	handler := newLocationHandlerForTest(t, repository, &stockLevelRepositoryFake{})
	request := httptest.NewRequest(http.MethodPost, "/locations/7/activate", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK || len(repository.activatedIDs) != 1 || repository.activatedIDs[0] != 7 {
		t.Fatalf("activation response = %d, IDs = %v", response.Code, repository.activatedIDs)
	}
	if !strings.Contains(response.Body.String(), "Active") {
		t.Fatal("activation response does not show active status")
	}
}

func TestLocationHandlerRendersDeactivationRejection(t *testing.T) {
	repository := &locationRepositoryFake{findResult: testLocation(7, true)}
	handler := newLocationHandlerForTest(t, repository, &stockLevelRepositoryFake{hasPositiveStock: true})
	request := httptest.NewRequest(http.MethodPost, "/locations/7/deactivate", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("deactivation status = %d, want %d", response.Code, http.StatusBadRequest)
	}
	if len(repository.deactivatedIDs) != 0 {
		t.Fatal("handler bypassed the use-case stock guard")
	}
	if !strings.Contains(response.Body.String(), "positive stock") {
		t.Fatal("deactivation rejection was not rendered as a user-facing message")
	}
}

func TestLocationHandlerDeactivatesLocationWithZeroStock(t *testing.T) {
	repository := &locationRepositoryFake{findResult: testLocation(7, true), deactivateResult: testLocation(7, false)}
	handler := newLocationHandlerForTest(t, repository, &stockLevelRepositoryFake{})
	request := httptest.NewRequest(http.MethodPost, "/locations/7/deactivate", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK || len(repository.deactivatedIDs) != 1 || repository.deactivatedIDs[0] != 7 {
		t.Fatalf("deactivation response = %d, IDs = %v", response.Code, repository.deactivatedIDs)
	}
	if !strings.Contains(response.Body.String(), "Inactive") {
		t.Fatal("successful deactivation does not show inactive status")
	}
}

func TestLocationHandlerMapsUnknownErrorsToSafeMessage(t *testing.T) {
	repository := &locationRepositoryFake{createErr: errors.New("database unavailable")}
	handler := newLocationHandlerForTest(t, repository, &stockLevelRepositoryFake{})
	request := httptest.NewRequest(http.MethodPost, "/locations", strings.NewReader("name=Warehouse&street=Main&city=Springfield"))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if !strings.Contains(response.Body.String(), "could not be completed") {
		t.Fatal("unknown error was not mapped to a user-facing message")
	}
}

func testLocation(id int64, active bool) domain.Location {
	return domain.Location{ID: id, Name: "Warehouse", Street: "Main Street 1", City: "Springfield", Active: active}
}
