package postgres

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/PedroJ03/go-inventory-management/internal/application"
	"github.com/PedroJ03/go-inventory-management/internal/domain"
)

func TestProductRepositoryCreate(t *testing.T) {
	id := uuid.New()
	product := domain.Product{Identity: domain.ProductIdentity{ID: id, Name: "Milk", Brand: "Acme", Amount: 1, Unit: domain.UnitEach}}

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repository, err := NewProductRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectExec(regexp.QuoteMeta(insertProductSQL)).WithArgs(id, "Milk", "Acme", 1.0, domain.UnitEach).WillReturnResult(sqlmock.NewResult(0, 1))

	if err := repository.Create(context.Background(), product); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestProductRepositoryCreateMapsDuplicate(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repository, err := NewProductRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	product := domain.Product{Identity: domain.ProductIdentity{ID: uuid.New(), Name: "Milk", Brand: "Acme", Amount: 1, Unit: domain.UnitEach}}
	mock.ExpectExec(regexp.QuoteMeta(insertProductSQL)).WillReturnError(&pq.Error{Code: "23505", Constraint: "products_business_identity_idx"})

	err = repository.Create(context.Background(), product)
	if !errors.Is(err, application.ErrProductAlreadyExists) {
		t.Fatalf("Create() error = %v, want duplicate error", err)
	}
}

func TestProductRepositoryFindByIDMapsRow(t *testing.T) {
	id := uuid.New()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repository, err := NewProductRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectQuery(regexp.QuoteMeta(findProductByIDSQL)).WithArgs(id).WillReturnRows(
		sqlmock.NewRows([]string{"id", "name", "brand", "amount", "unit"}).AddRow(id.String(), "Milk", "Acme", "0.5", "liters"),
	)

	product, err := repository.FindByID(context.Background(), id)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	want := domain.ProductIdentity{ID: id, Name: "Milk", Brand: "Acme", Amount: 0.5, Unit: domain.UnitLiters}
	if product.Identity != want {
		t.Fatalf("FindByID() = %+v, want %+v", product.Identity, want)
	}
}

func TestProductRepositoryFindByIdentityUsesBusinessIdentity(t *testing.T) {
	id := uuid.New()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repository, err := NewProductRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectQuery(regexp.QuoteMeta(findProductByIdentitySQL)).WithArgs("Milk", "Acme", 0.5, domain.UnitLiters).WillReturnRows(
		sqlmock.NewRows([]string{"id", "name", "brand", "amount", "unit"}).AddRow(id.String(), "milk", "acme", 0.5, "liters"),
	)

	product, err := repository.FindByIdentity(context.Background(), "Milk", "Acme", 0.5, domain.UnitLiters)
	if err != nil {
		t.Fatalf("FindByIdentity() error = %v", err)
	}
	if product.Identity.ID != id {
		t.Fatalf("FindByIdentity() ID = %v, want %v", product.Identity.ID, id)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestProductRepositoryFindMapsNotFound(t *testing.T) {
	id := uuid.New()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repository, err := NewProductRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectQuery(regexp.QuoteMeta(findProductByIDSQL)).WithArgs(id).WillReturnError(sql.ErrNoRows)

	_, err = repository.FindByID(context.Background(), id)
	if !errors.Is(err, application.ErrProductNotFound) {
		t.Fatalf("FindByID() error = %v, want not found error", err)
	}
}

func TestScanProductRejectsInvalidDatabaseRow(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repository, err := NewProductRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	id := uuid.New()
	mock.ExpectQuery(regexp.QuoteMeta(findProductByIDSQL)).WithArgs(id).WillReturnRows(
		sqlmock.NewRows([]string{"id", "name", "brand", "amount", "unit"}).AddRow(id.String(), "Milk", "Acme", 2, "each"),
	)

	_, err = repository.FindByID(context.Background(), id)
	if err == nil {
		t.Fatal("FindByID() error = nil, want invalid database row error")
	}
}
