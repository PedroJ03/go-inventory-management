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

type productRepositoryFake struct {
	created  []domain.Product
	createFn func(context.Context, domain.Product) error
}

func (f *productRepositoryFake) Create(ctx context.Context, product domain.Product) error {
	f.created = append(f.created, product)
	if f.createFn != nil {
		return f.createFn(ctx, product)
	}
	return nil
}

func (f *productRepositoryFake) FindByID(context.Context, uuid.UUID) (domain.Product, error) {
	return domain.Product{}, application.ErrProductNotFound
}

func (f *productRepositoryFake) FindByIdentity(context.Context, string, string, float64, domain.Unit) (domain.Product, error) {
	return domain.Product{}, application.ErrProductNotFound
}

func TestProductHandlerServesCreationForm(t *testing.T) {
	handler, err := NewProductHandler(&productRepositoryFake{})
	if err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodGet, "/products/new", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("GET status = %d, want %d", response.Code, http.StatusOK)
	}
	body := response.Body.String()
	for _, expected := range []string{"Create product", `name="name"`, `name="amount"`, "milliliters"} {
		if !strings.Contains(body, expected) {
			t.Errorf("GET body does not contain %q", expected)
		}
	}
}

func TestProductHandlerCreatesProduct(t *testing.T) {
	repository := &productRepositoryFake{}
	handler, err := NewProductHandler(repository)
	if err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodPost, "/products", strings.NewReader("name=Milk&brand=Acme&amount=0.5&unit=liters"))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("POST status = %d, want %d", response.Code, http.StatusCreated)
	}
	if len(repository.created) != 1 || repository.created[0].Identity.Amount != 0.5 {
		t.Fatalf("repository received %+v, want one 0.5-liter product", repository.created)
	}
	if !strings.Contains(response.Body.String(), "Product created") {
		t.Fatal("success response does not show created-product result")
	}
}

func TestProductHandlerRejectsInvalidFormWithoutPersistence(t *testing.T) {
	repository := &productRepositoryFake{}
	handler, err := NewProductHandler(repository)
	if err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodPost, "/products", strings.NewReader("name=Milk&brand=Acme&amount=not-a-number&unit=packs"))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("invalid POST status = %d, want %d", response.Code, http.StatusBadRequest)
	}
	if len(repository.created) != 0 {
		t.Fatalf("repository received %d products, want none", len(repository.created))
	}
	for _, expected := range []string{"valid number", "supported unit"} {
		if !strings.Contains(response.Body.String(), expected) {
			t.Errorf("invalid response does not contain %q", expected)
		}
	}
}

func TestProductHandlerRendersApplicationError(t *testing.T) {
	repository := &productRepositoryFake{createFn: func(context.Context, domain.Product) error {
		return application.ErrProductAlreadyExists
	}}
	handler, err := NewProductHandler(repository)
	if err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodPost, "/products", strings.NewReader("name=Milk&brand=Acme&amount=1&unit=each"))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("application-error status = %d, want %d", response.Code, http.StatusBadRequest)
	}
	if !strings.Contains(response.Body.String(), "already exists") {
		t.Fatal("application error was not rendered as a user-facing form error")
	}
}

func TestProductHandlerRendersRepositoryError(t *testing.T) {
	repository := &productRepositoryFake{createFn: func(context.Context, domain.Product) error {
		return errors.New("database unavailable")
	}}
	handler, err := NewProductHandler(repository)
	if err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodPost, "/products", strings.NewReader("name=Milk&brand=Acme&amount=1&unit=each"))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("repository-error status = %d, want %d", response.Code, http.StatusBadRequest)
	}
	if !strings.Contains(response.Body.String(), "could not be created") {
		t.Fatal("repository error was not rendered as a user-facing form error")
	}
}
