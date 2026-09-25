package httpadapter

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/pedroj/go-inventory-management/internal/application"
	"github.com/pedroj/go-inventory-management/internal/domain"
)

type receiveProductRepositoryFake struct {
	product domain.Product
	err     error
}

func (f receiveProductRepositoryFake) Create(context.Context, domain.Product) error { return nil }
func (f receiveProductRepositoryFake) FindByID(context.Context, uuid.UUID) (domain.Product, error) {
	return f.product, f.err
}
func (f receiveProductRepositoryFake) FindByIdentity(context.Context, string, string, float64, domain.Unit) (domain.Product, error) {
	return domain.Product{}, application.ErrProductNotFound
}

type receiveLocationRepositoryFake struct {
	location domain.Location
	err      error
}

func (f receiveLocationRepositoryFake) Create(context.Context, domain.Location) (domain.Location, error) {
	return domain.Location{}, nil
}
func (f receiveLocationRepositoryFake) FindByID(context.Context, int64) (domain.Location, error) {
	return f.location, f.err
}
func (f receiveLocationRepositoryFake) Activate(context.Context, int64) (domain.Location, error) {
	return f.location, nil
}
func (f receiveLocationRepositoryFake) Deactivate(context.Context, int64) (domain.Location, error) {
	return f.location, nil
}

type receiveStockPortFake struct {
	stock    domain.StockLevel
	movement domain.StockMovement
	err      error
}

func (f *receiveStockPortFake) Receive(_ context.Context, movement domain.StockMovement) (domain.StockLevel, domain.StockMovement, error) {
	f.movement = movement
	return f.stock, domain.StockMovement{ID: 42, Type: movement.Type, Units: movement.Units, ProductID: movement.ProductID, ToID: movement.ToID, CreatedAt: movement.CreatedAt}, f.err
}

func receiveHandlerFixtures() (uuid.UUID, receiveProductRepositoryFake, receiveLocationRepositoryFake) {
	productID := uuid.New()
	return productID,
		receiveProductRepositoryFake{product: domain.Product{Identity: domain.ProductIdentity{ID: productID, Name: "Milk", Brand: "Acme", Amount: 1, Unit: domain.UnitEach}}},
		receiveLocationRepositoryFake{location: domain.Location{ID: 7, Name: "Warehouse", Street: "Main", City: "Springfield", Active: true}}
}

func newReceiveStockHandler(t *testing.T, products application.ProductRepository, locations application.LocationRepository, receiving application.StockReceivingPort) *ReceiveStockHandler {
	t.Helper()
	handler, err := NewReceiveStockHandler(products, locations, receiving)
	if err != nil {
		t.Fatal(err)
	}
	return handler
}

func receiveRequest(method, body string) *http.Request {
	request := httptest.NewRequest(method, "/stock/receive", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return request
}

func TestReceiveStockHandlerServesForm(t *testing.T) {
	_, products, locations := receiveHandlerFixtures()
	handler := newReceiveStockHandler(t, products, locations, &receiveStockPortFake{})

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/stock/receive", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("GET status = %d, want %d", response.Code, http.StatusOK)
	}
	for _, expected := range []string{"Receive stock", `name="product_id"`, `name="location_id"`, `name="units"`} {
		if !strings.Contains(response.Body.String(), expected) {
			t.Errorf("GET body does not contain %q", expected)
		}
	}
}

func TestReceiveStockHandlerReceivesStock(t *testing.T) {
	productID, products, locations := receiveHandlerFixtures()
	port := &receiveStockPortFake{stock: domain.StockLevel{ProductID: productID, LocationID: 7, Units: 17}}
	handler := newReceiveStockHandler(t, products, locations, port)

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, receiveRequest(http.MethodPost, "product_id="+productID.String()+"&location_id=7&units=5"))

	if response.Code != http.StatusCreated {
		t.Fatalf("POST status = %d, want %d", response.Code, http.StatusCreated)
	}
	if port.movement.Units != 5 || port.movement.ToID == nil || *port.movement.ToID != 7 {
		t.Fatalf("receiving port got movement %+v", port.movement)
	}
	for _, expected := range []string{"Stock received successfully", "17", "42"} {
		if !strings.Contains(response.Body.String(), expected) {
			t.Errorf("success response does not contain %q", expected)
		}
	}
}

func TestReceiveStockHandlerRejectsInvalidFormWithoutPersistence(t *testing.T) {
	_, products, locations := receiveHandlerFixtures()
	port := &receiveStockPortFake{}
	handler := newReceiveStockHandler(t, products, locations, port)

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, receiveRequest(http.MethodPost, "product_id=not-a-uuid&location_id=0&units=1.5"))

	if response.Code != http.StatusBadRequest {
		t.Fatalf("invalid POST status = %d, want %d", response.Code, http.StatusBadRequest)
	}
	if port.movement.Units != 0 {
		t.Fatal("receiving port was called for invalid form data")
	}
	if !strings.Contains(response.Body.String(), "valid product UUID") {
		t.Fatal("invalid form response did not explain the product UUID error")
	}
}

func TestReceiveStockHandlerRendersInactiveLocationError(t *testing.T) {
	productID, products, locations := receiveHandlerFixtures()
	locations.location.Active = false
	port := &receiveStockPortFake{}
	handler := newReceiveStockHandler(t, products, locations, port)

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, receiveRequest(http.MethodPost, "product_id="+productID.String()+"&location_id=7&units=5"))

	if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), "inactive") {
		t.Fatalf("inactive location response = status %d, body %q", response.Code, response.Body.String())
	}
	if port.movement.Units != 0 {
		t.Fatal("receiving port was called for an inactive location")
	}
}

func TestReceiveStockHandlerRendersNotFoundError(t *testing.T) {
	productID, products, locations := receiveHandlerFixtures()
	products.err = application.ErrProductNotFound
	handler := newReceiveStockHandler(t, products, locations, &receiveStockPortFake{})

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, receiveRequest(http.MethodPost, "product_id="+productID.String()+"&location_id=7&units=5"))

	if response.Code != http.StatusNotFound || !strings.Contains(response.Body.String(), "could not be found") {
		t.Fatalf("not-found response = status %d, body %q", response.Code, response.Body.String())
	}
}

func TestReceiveStockHandlerRendersRepositoryError(t *testing.T) {
	productID, products, locations := receiveHandlerFixtures()
	port := &receiveStockPortFake{err: errors.New("database unavailable")}
	handler := newReceiveStockHandler(t, products, locations, port)

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, receiveRequest(http.MethodPost, "product_id="+productID.String()+"&location_id=7&units=5"))

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("repository-error status = %d, want %d", response.Code, http.StatusInternalServerError)
	}
	if !strings.Contains(response.Body.String(), "could not be completed") || strings.Contains(response.Body.String(), "database unavailable") {
		t.Fatalf("repository error was not rendered safely: %q", response.Body.String())
	}
}
