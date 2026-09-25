package postgres

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"

	"github.com/pedroj/go-inventory-management/internal/application"
	"github.com/pedroj/go-inventory-management/internal/domain"
)

func newStockSellingRepositoryForTest(t *testing.T) (*StockSellingRepository, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	repository, err := NewStockSellingRepository(db)
	if err != nil {
		db.Close()
		t.Fatal(err)
	}
	return repository, mock, func() { db.Close() }
}

func sellingMovementFixture() (domain.StockMovement, time.Time) {
	createdAt := time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC)
	fromID := int64(7)
	return domain.StockMovement{Type: domain.MovementSelling, Units: 5, ProductID: uuid.New(), FromID: &fromID, CreatedAt: createdAt}, createdAt
}

func expectSellingStockUpdate(mock sqlmock.Sqlmock, movement domain.StockMovement, units int64) {
	mock.ExpectQuery(regexp.QuoteMeta(sellStockLevelSQL)).WithArgs(movement.ProductID, int64(7), movement.Units).WillReturnRows(
		sqlmock.NewRows([]string{"product_id", "location_id", "units"}).AddRow(movement.ProductID.String(), 7, units),
	)
}

func TestStockSellingRepositoryCommitsStockAndMovementTogether(t *testing.T) {
	repository, mock, closeDB := newStockSellingRepositoryForTest(t)
	defer closeDB()
	movement, createdAt := sellingMovementFixture()
	mock.ExpectBegin()
	expectSellingStockUpdate(mock, movement, 10)
	mock.ExpectQuery(regexp.QuoteMeta(insertStockMovementSQL)).WithArgs(movement.Type, movement.Units, movement.ProductID, int64(7), nil, createdAt).WillReturnRows(
		sqlmock.NewRows([]string{"id", "type", "units", "product_id", "from_location_id", "to_location_id", "created_at"}).AddRow(42, movement.Type, movement.Units, movement.ProductID.String(), 7, nil, createdAt),
	)
	mock.ExpectCommit()

	stock, persisted, err := repository.Sell(context.Background(), movement)
	if err != nil {
		t.Fatalf("Sell() error = %v", err)
	}
	if stock.Units != 10 || persisted.ID != 42 || !persisted.CreatedAt.Equal(createdAt) {
		t.Fatalf("Sell() = stock %+v, movement %+v; want persisted values", stock, persisted)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestStockSellingRepositoryRejectsNoStockOrInsufficientStock(t *testing.T) {
	for _, name := range []string{"no stock row", "insufficient units"} {
		t.Run(name, func(t *testing.T) {
			repository, mock, closeDB := newStockSellingRepositoryForTest(t)
			defer closeDB()
			movement, _ := sellingMovementFixture()
			mock.ExpectBegin()
			mock.ExpectQuery(regexp.QuoteMeta(sellStockLevelSQL)).WithArgs(movement.ProductID, int64(7), movement.Units).WillReturnError(sql.ErrNoRows)
			mock.ExpectRollback()

			_, _, err := repository.Sell(context.Background(), movement)
			if !errors.Is(err, application.ErrInsufficientStock) {
				t.Fatalf("Sell() error = %v, want insufficient stock error", err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestStockSellingRepositoryRollsBackWhenMovementInsertFails(t *testing.T) {
	repository, mock, closeDB := newStockSellingRepositoryForTest(t)
	defer closeDB()
	movement, createdAt := sellingMovementFixture()
	mock.ExpectBegin()
	expectSellingStockUpdate(mock, movement, 10)
	mock.ExpectQuery(regexp.QuoteMeta(insertStockMovementSQL)).WithArgs(movement.Type, movement.Units, movement.ProductID, int64(7), nil, createdAt).WillReturnError(errors.New("movement insert failed"))
	mock.ExpectRollback()

	_, _, err := repository.Sell(context.Background(), movement)
	if !errors.Is(err, application.ErrStockMovementPersistence) {
		t.Fatalf("Sell() error = %v, want movement persistence error", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
