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

type sellProductRepositoryFake struct {
	product domain.Product
	err     error
}

func (f sellProductRepositoryFake) Create(context.Context, domain.Product) error { return nil }
func (f sellProductRepositoryFake) FindByID(context.Context, uuid.UUID) (domain.Product, error) {
	return f.product, f.err
}
func (f sellProductRepositoryFake) FindByIdentity(context.Context, string, string, float64, domain.Unit) (domain.Product, error) {
	return domain.Product{}, application.ErrProductNotFound
}

type sellLocationRepositoryFake struct {
	location domain.Location
	err      error
}

func (f sellLocationRepositoryFake) Create(context.Context, domain.Location) (domain.Location, error) {
	return domain.Location{}, nil
}
func (f sellLocationRepositoryFake) FindByID(context.Context, int64) (domain.Location, error) {
	return f.location, f.err
}
func (f sellLocationRepositoryFake) Activate(context.Context, int64) (domain.Location, error) {
	return f.location, nil
}
func (f sellLocationRepositoryFake) Deactivate(context.Context, int64) (domain.Location, error) {
	return f.location, nil
}

type sellStockPortFake struct {
	stock    domain.StockLevel
	movement domain.StockMovement
	err      error
}

func (f *sellStockPortFake) Sell(_ context.Context, movement domain.StockMovement) (domain.StockLevel, domain.StockMovement, error) {
	f.movement = movement
	movement.ID = 42
	return f.stock, movement, f.err
}

func sellHandlerFixtures() (uuid.UUID, sellProductRepositoryFake, sellLocationRepositoryFake) {
	productID := uuid.New()
	return productID,
		sellProductRepositoryFake{product: domain.Product{Identity: domain.ProductIdentity{ID: productID, Name: "Milk", Brand: "Acme", Amount: 1, Unit: domain.UnitEach}}},
		sellLocationRepositoryFake{location: domain.Location{ID: 7, Name: "Warehouse", Street: "Main", City: "Springfield", Active: true}}
}

func newSellStockHandler(t *testing.T, products application.ProductRepository, locations application.LocationRepository, selling application.StockSellingPort) *SellStockHandler {
	t.Helper()
	handler, err := NewSellStockHandler(products, locations, selling)
	if err != nil {
		t.Fatal(err)
	}
	return handler
}

func sellRequest(body string) *http.Request {
	request := httptest.NewRequest(http.MethodPost, "/stock/sell", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return request
}

func TestSellStockHandlerServesForm(t *testing.T) {
	_, products, locations := sellHandlerFixtures()
	handler := newSellStockHandler(t, products, locations, &sellStockPortFake{})

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/stock/sell", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("GET status = %d, want %d", response.Code, http.StatusOK)
	}
	for _, expected := range []string{"Sell stock", `name="product_id"`, `name="location_id"`, `name="units"`} {
		if !strings.Contains(response.Body.String(), expected) {
			t.Errorf("GET body does not contain %q", expected)
		}
	}
}

func TestSellStockHandlerSellsStock(t *testing.T) {
	productID, products, locations := sellHandlerFixtures()
	port := &sellStockPortFake{stock: domain.StockLevel{ProductID: productID, LocationID: 7, Units: 3}}
	handler := newSellStockHandler(t, products, locations, port)

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, sellRequest("product_id="+productID.String()+"&location_id=7&units=2"))

	if response.Code != http.StatusCreated {
		t.Fatalf("POST status = %d, want %d", response.Code, http.StatusCreated)
	}
	if port.movement.Units != 2 || port.movement.FromID == nil || *port.movement.FromID != 7 || port.movement.ToID != nil {
		t.Fatalf("selling port got movement %+v", port.movement)
	}
	for _, expected := range []string{"Stock sold successfully", "3", "42"} {
		if !strings.Contains(response.Body.String(), expected) {
			t.Errorf("success response does not contain %q", expected)
		}
	}
}

func TestSellStockHandlerRejectsInvalidFormWithoutPersistence(t *testing.T) {
	_, products, locations := sellHandlerFixtures()
	port := &sellStockPortFake{}
	handler := newSellStockHandler(t, products, locations, port)

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, sellRequest("product_id=not-a-uuid&location_id=0&units=1.5"))

	if response.Code != http.StatusBadRequest || port.movement.Units != 0 {
		t.Fatalf("invalid form response = status %d, movement %+v", response.Code, port.movement)
	}
	if !strings.Contains(response.Body.String(), "valid product UUID") {
		t.Fatal("invalid form response did not explain the product UUID error")
	}
}

func TestSellStockHandlerRendersInactiveLocationError(t *testing.T) {
	productID, products, locations := sellHandlerFixtures()
	locations.location.Active = false
	port := &sellStockPortFake{}
	handler := newSellStockHandler(t, products, locations, port)

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, sellRequest("product_id="+productID.String()+"&location_id=7&units=1"))

	if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), "inactive") || port.movement.Units != 0 {
		t.Fatalf("inactive location response = status %d, body %q, movement %+v", response.Code, response.Body.String(), port.movement)
	}
}

func TestSellStockHandlerRendersNotFoundError(t *testing.T) {
	productID, products, locations := sellHandlerFixtures()
	products.err = application.ErrProductNotFound
	handler := newSellStockHandler(t, products, locations, &sellStockPortFake{})

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, sellRequest("product_id="+productID.String()+"&location_id=7&units=1"))

	if response.Code != http.StatusNotFound || !strings.Contains(response.Body.String(), "could not be found") {
		t.Fatalf("not-found response = status %d, body %q", response.Code, response.Body.String())
	}
}

func TestSellStockHandlerRendersInsufficientStockError(t *testing.T) {
	productID, products, locations := sellHandlerFixtures()
	port := &sellStockPortFake{err: application.ErrInsufficientStock}
	handler := newSellStockHandler(t, products, locations, port)

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, sellRequest("product_id="+productID.String()+"&location_id=7&units=1"))

	if response.Code != http.StatusConflict || !strings.Contains(response.Body.String(), "not enough stock") {
		t.Fatalf("insufficient-stock response = status %d, body %q", response.Code, response.Body.String())
	}
}

func TestSellStockHandlerRendersRepositoryErrorSafely(t *testing.T) {
	productID, products, locations := sellHandlerFixtures()
	port := &sellStockPortFake{err: errors.New("database unavailable")}
	handler := newSellStockHandler(t, products, locations, port)

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, sellRequest("product_id="+productID.String()+"&location_id=7&units=1"))

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("repository-error status = %d, want %d", response.Code, http.StatusInternalServerError)
	}
	if !strings.Contains(response.Body.String(), "could not be completed") || strings.Contains(response.Body.String(), "database unavailable") {
		t.Fatalf("repository error was not rendered safely: %q", response.Body.String())
	}
}
