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

func newStockTransferringRepositoryForTest(t *testing.T) (*StockTransferringRepository, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	repository, err := NewStockTransferringRepository(db)
	if err != nil {
		db.Close()
		t.Fatal(err)
	}
	return repository, mock, func() { db.Close() }
}

func transferringMovementFixture() (domain.StockMovement, time.Time) {
	createdAt := time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC)
	fromID, toID := int64(7), int64(8)
	return domain.StockMovement{Type: domain.MovementTransferring, Units: 5, ProductID: uuid.New(), FromID: &fromID, ToID: &toID, CreatedAt: createdAt}, createdAt
}

func expectTransferSourceUpdate(mock sqlmock.Sqlmock, movement domain.StockMovement, units int64) {
	mock.ExpectQuery(regexp.QuoteMeta(transferStockLevelSQL)).WithArgs(movement.ProductID, int64(7), movement.Units).WillReturnRows(sqlmock.NewRows([]string{"product_id", "location_id", "units"}).AddRow(movement.ProductID.String(), 7, units))
}

func expectTransferDestinationUpsert(mock sqlmock.Sqlmock, movement domain.StockMovement, units int64) {
	mock.ExpectQuery(regexp.QuoteMeta(transferDestinationStockLevelSQL)).WithArgs(movement.ProductID, int64(8), movement.Units).WillReturnRows(sqlmock.NewRows([]string{"product_id", "location_id", "units"}).AddRow(movement.ProductID.String(), 8, units))
}

func expectTransferMovementInsert(mock sqlmock.Sqlmock, movement domain.StockMovement, createdAt time.Time) {
	mock.ExpectQuery(regexp.QuoteMeta(insertStockMovementSQL)).WithArgs(movement.Type, movement.Units, movement.ProductID, int64(7), int64(8), createdAt).WillReturnRows(sqlmock.NewRows([]string{"id", "type", "units", "product_id", "from_location_id", "to_location_id", "created_at"}).AddRow(42, movement.Type, movement.Units, movement.ProductID.String(), 7, 8, createdAt))
}

func TestStockTransferringRepositoryAutoCreatesDestinationAndCommitsAtomically(t *testing.T) {
	repository, mock, closeDB := newStockTransferringRepositoryForTest(t)
	defer closeDB()
	movement, createdAt := transferringMovementFixture()
	mock.ExpectBegin()
	expectTransferSourceUpdate(mock, movement, 5)
	expectTransferDestinationUpsert(mock, movement, 5)
	expectTransferMovementInsert(mock, movement, createdAt)
	mock.ExpectCommit()

	source, destination, persisted, err := repository.Transfer(context.Background(), movement)
	if err != nil {
		t.Fatalf("Transfer() error = %v", err)
	}
	if source.Units != 5 || destination.LocationID != 8 || persisted.ID != 42 {
		t.Fatalf("Transfer() = source %+v, destination %+v, movement %+v", source, destination, persisted)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestStockTransferringRepositoryRejectsInsufficientSourceStock(t *testing.T) {
	repository, mock, closeDB := newStockTransferringRepositoryForTest(t)
	defer closeDB()
	movement, _ := transferringMovementFixture()
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(transferStockLevelSQL)).WithArgs(movement.ProductID, int64(7), movement.Units).WillReturnError(sql.ErrNoRows)
	mock.ExpectRollback()

	_, _, _, err := repository.Transfer(context.Background(), movement)
	if !errors.Is(err, application.ErrInsufficientStock) {
		t.Fatalf("Transfer() error = %v, want insufficient stock", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestStockTransferringRepositoryRollsBackWhenMovementInsertFails(t *testing.T) {
	repository, mock, closeDB := newStockTransferringRepositoryForTest(t)
	defer closeDB()
	movement, createdAt := transferringMovementFixture()
	mock.ExpectBegin()
	expectTransferSourceUpdate(mock, movement, 5)
	expectTransferDestinationUpsert(mock, movement, 5)
	mock.ExpectQuery(regexp.QuoteMeta(insertStockMovementSQL)).WithArgs(movement.Type, movement.Units, movement.ProductID, int64(7), int64(8), createdAt).WillReturnError(errors.New("movement insert failed"))
	mock.ExpectRollback()

	_, _, _, err := repository.Transfer(context.Background(), movement)
	if !errors.Is(err, application.ErrStockMovementPersistence) {
		t.Fatalf("Transfer() error = %v, want movement persistence", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
