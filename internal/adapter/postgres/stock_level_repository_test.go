package postgres

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"

	"github.com/pedroj/go-inventory-management/internal/application"
	"github.com/pedroj/go-inventory-management/internal/domain"
)

func newStockLevelRepositoryForTest(t *testing.T) (*StockLevelRepository, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	repository, err := NewStockLevelRepository(db)
	if err != nil {
		db.Close()
		t.Fatal(err)
	}
	return repository, mock, func() { db.Close() }
}

func TestStockLevelRepositoryFindMapsRow(t *testing.T) {
	repository, mock, closeDB := newStockLevelRepositoryForTest(t)
	defer closeDB()
	productID := uuid.New()
	mock.ExpectQuery(regexp.QuoteMeta(findStockLevelSQL)).WithArgs(productID, int64(7)).WillReturnRows(
		sqlmock.NewRows([]string{"product_id", "location_id", "units"}).AddRow(productID.String(), 7, 12),
	)

	got, err := repository.FindByProductAndLocation(context.Background(), productID, 7)
	if err != nil {
		t.Fatalf("FindByProductAndLocation() error = %v", err)
	}
	want := domain.StockLevel{ProductID: productID, LocationID: 7, Units: 12}
	if got != want {
		t.Fatalf("FindByProductAndLocation() = %+v, want %+v", got, want)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestStockLevelRepositoryUpsert(t *testing.T) {
	repository, mock, closeDB := newStockLevelRepositoryForTest(t)
	defer closeDB()
	productID := uuid.New()
	stockLevel := domain.StockLevel{ProductID: productID, LocationID: 7, Units: 12}
	mock.ExpectExec(regexp.QuoteMeta(upsertStockLevelSQL)).WithArgs(productID, int64(7), int64(12)).WillReturnResult(sqlmock.NewResult(0, 1))

	if err := repository.Upsert(context.Background(), stockLevel); err != nil {
		t.Fatalf("Upsert() error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestStockLevelRepositoryUpsertPreservesZeroUnits(t *testing.T) {
	repository, mock, closeDB := newStockLevelRepositoryForTest(t)
	defer closeDB()
	productID := uuid.New()
	mock.ExpectExec(regexp.QuoteMeta(upsertStockLevelSQL)).WithArgs(productID, int64(7), int64(0)).WillReturnResult(sqlmock.NewResult(0, 1))

	if err := repository.Upsert(context.Background(), domain.StockLevel{ProductID: productID, LocationID: 7, Units: 0}); err != nil {
		t.Fatalf("Upsert() zero units error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestStockLevelRepositoryHasPositiveStock(t *testing.T) {
	tests := []struct {
		name  string
		value bool
	}{
		{name: "positive stock exists", value: true},
		{name: "only zero stock exists", value: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository, mock, closeDB := newStockLevelRepositoryForTest(t)
			defer closeDB()
			mock.ExpectQuery(regexp.QuoteMeta(hasPositiveStockAtLocationSQL)).WithArgs(int64(7)).WillReturnRows(
				sqlmock.NewRows([]string{"exists"}).AddRow(tt.value),
			)

			got, err := repository.HasPositiveStockAtLocation(context.Background(), 7)
			if err != nil || got != tt.value {
				t.Fatalf("HasPositiveStockAtLocation() = %v, error = %v; want %v", got, err, tt.value)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestStockLevelRepositoryMapsNotFound(t *testing.T) {
	repository, mock, closeDB := newStockLevelRepositoryForTest(t)
	defer closeDB()
	productID := uuid.New()
	mock.ExpectQuery(regexp.QuoteMeta(findStockLevelSQL)).WithArgs(productID, int64(7)).WillReturnError(sql.ErrNoRows)

	_, err := repository.FindByProductAndLocation(context.Background(), productID, 7)
	if !errors.Is(err, application.ErrStockLevelNotFound) {
		t.Fatalf("FindByProductAndLocation() error = %v, want not found error", err)
	}
}

func TestStockLevelRepositoryRejectsInvalidInputWithoutQueries(t *testing.T) {
	repository, mock, closeDB := newStockLevelRepositoryForTest(t)
	defer closeDB()

	if _, err := repository.FindByProductAndLocation(context.Background(), uuid.Nil, 7); !errors.Is(err, application.ErrInvalidStockLevel) {
		t.Fatalf("FindByProductAndLocation() error = %v, want invalid stock level error", err)
	}
	if err := repository.Upsert(context.Background(), domain.StockLevel{ProductID: uuid.New(), LocationID: 7, Units: -1}); !errors.Is(err, application.ErrInvalidStockLevel) {
		t.Fatalf("Upsert() error = %v, want invalid stock level error", err)
	}
	if _, err := repository.HasPositiveStockAtLocation(context.Background(), 0); !errors.Is(err, application.ErrInvalidStockLevel) {
		t.Fatalf("HasPositiveStockAtLocation() error = %v, want invalid stock level error", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
