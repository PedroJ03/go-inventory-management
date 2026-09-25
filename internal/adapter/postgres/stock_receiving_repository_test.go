package postgres

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"

	"github.com/pedroj/go-inventory-management/internal/application"
	"github.com/pedroj/go-inventory-management/internal/domain"
)

func newStockReceivingRepositoryForTest(t *testing.T) (*StockReceivingRepository, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	repository, err := NewStockReceivingRepository(db)
	if err != nil {
		db.Close()
		t.Fatal(err)
	}
	return repository, mock, func() { db.Close() }
}

func receivingMovementFixture() (domain.StockMovement, time.Time) {
	createdAt := time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC)
	toID := int64(7)
	return domain.StockMovement{Type: domain.MovementReceiving, Units: 5, ProductID: uuid.New(), ToID: &toID, CreatedAt: createdAt}, createdAt
}

func TestStockReceivingRepositoryCommitsStockAndMovementTogether(t *testing.T) {
	repository, mock, closeDB := newStockReceivingRepositoryForTest(t)
	defer closeDB()
	movement, createdAt := receivingMovementFixture()
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(receiveStockLevelSQL)).WithArgs(movement.ProductID, int64(7), int64(5)).WillReturnRows(
		sqlmock.NewRows([]string{"product_id", "location_id", "units"}).AddRow(movement.ProductID.String(), 7, 15),
	)
	mock.ExpectQuery(regexp.QuoteMeta(insertStockMovementSQL)).WithArgs(movement.Type, movement.Units, movement.ProductID, nil, int64(7), createdAt).WillReturnRows(
		sqlmock.NewRows([]string{"id", "type", "units", "product_id", "from_location_id", "to_location_id", "created_at"}).AddRow(42, movement.Type, movement.Units, movement.ProductID.String(), nil, 7, createdAt),
	)
	mock.ExpectCommit()

	stock, persisted, err := repository.Receive(context.Background(), movement)
	if err != nil {
		t.Fatalf("Receive() error = %v", err)
	}
	if stock.Units != 15 || persisted.ID != 42 || !persisted.CreatedAt.Equal(createdAt) {
		t.Fatalf("Receive() = stock %+v, movement %+v; want persisted values", stock, persisted)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestStockReceivingRepositoryRollsBackWhenMovementInsertFails(t *testing.T) {
	repository, mock, closeDB := newStockReceivingRepositoryForTest(t)
	defer closeDB()
	movement, createdAt := receivingMovementFixture()
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(receiveStockLevelSQL)).WithArgs(movement.ProductID, int64(7), int64(5)).WillReturnRows(
		sqlmock.NewRows([]string{"product_id", "location_id", "units"}).AddRow(movement.ProductID.String(), 7, 15),
	)
	mock.ExpectQuery(regexp.QuoteMeta(insertStockMovementSQL)).WithArgs(movement.Type, movement.Units, movement.ProductID, nil, int64(7), createdAt).WillReturnError(errors.New("movement insert failed"))
	mock.ExpectRollback()

	_, _, err := repository.Receive(context.Background(), movement)
	if !errors.Is(err, application.ErrStockMovementPersistence) {
		t.Fatalf("Receive() error = %v, want movement persistence error", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
