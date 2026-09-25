package postgres

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/pedroj/go-inventory-management/internal/application"
	"github.com/pedroj/go-inventory-management/internal/domain"
)

func newStockMovementRepositoryForTest(t *testing.T) (*StockMovementRepository, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	repository, err := NewStockMovementRepository(db)
	if err != nil {
		db.Close()
		t.Fatal(err)
	}
	return repository, mock, func() { db.Close() }
}

func TestStockMovementRepositoryCreateMapsMovementTypesAndGeneratedID(t *testing.T) {
	productID := uuid.New()
	createdAt := time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC)
	fromID, toID := int64(7), int64(8)
	tests := []struct {
		name     string
		movement domain.StockMovement
		fromArg  any
		toArg    any
	}{
		{
			name:     "receiving",
			movement: domain.StockMovement{Type: domain.MovementReceiving, Units: 10, ProductID: productID, ToID: &toID, CreatedAt: createdAt},
			fromArg:  nil,
			toArg:    toID,
		},
		{
			name:     "selling",
			movement: domain.StockMovement{Type: domain.MovementSelling, Units: 2, ProductID: productID, FromID: &fromID, CreatedAt: createdAt},
			fromArg:  fromID,
			toArg:    nil,
		},
		{
			name:     "transferring",
			movement: domain.StockMovement{Type: domain.MovementTransferring, Units: 3, ProductID: productID, FromID: &fromID, ToID: &toID, CreatedAt: createdAt},
			fromArg:  fromID,
			toArg:    toID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository, mock, closeDB := newStockMovementRepositoryForTest(t)
			defer closeDB()
			mock.ExpectQuery(regexp.QuoteMeta(insertStockMovementSQL)).
				WithArgs(tt.movement.Type, tt.movement.Units, productID, tt.fromArg, tt.toArg, createdAt).
				WillReturnRows(sqlmock.NewRows([]string{"id", "type", "units", "product_id", "from_location_id", "to_location_id", "created_at"}).
					AddRow(42, tt.movement.Type, tt.movement.Units, productID.String(), tt.fromArg, tt.toArg, createdAt))

			got, err := repository.Create(context.Background(), tt.movement)
			if err != nil {
				t.Fatalf("Create() error = %v", err)
			}
			want := tt.movement
			want.ID = 42
			if !got.CreatedAt.Equal(want.CreatedAt) || got.ID != want.ID || got.Type != want.Type || got.Units != want.Units || got.ProductID != want.ProductID || !sameOptionalInt(got.FromID, want.FromID) || !sameOptionalInt(got.ToID, want.ToID) {
				t.Fatalf("Create() = %+v, want %+v", got, want)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestStockMovementRepositoryCreateRejectsInvalidMovement(t *testing.T) {
	repository, mock, closeDB := newStockMovementRepositoryForTest(t)
	defer closeDB()

	_, err := repository.Create(context.Background(), domain.StockMovement{Type: domain.MovementReceiving, Units: 1, ProductID: uuid.New()})
	if !errors.Is(err, application.ErrInvalidStockMovement) {
		t.Fatalf("Create() error = %v, want invalid movement error", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestStockMovementRepositoryCreateMapsConstraintError(t *testing.T) {
	repository, mock, closeDB := newStockMovementRepositoryForTest(t)
	defer closeDB()
	fromID := int64(7)
	movement := domain.StockMovement{Type: domain.MovementSelling, Units: 1, ProductID: uuid.New(), FromID: &fromID, CreatedAt: time.Now()}
	mock.ExpectQuery(regexp.QuoteMeta(insertStockMovementSQL)).WillReturnError(&pq.Error{Code: "23503", Constraint: "stock_movements_product_id_fkey"})

	_, err := repository.Create(context.Background(), movement)
	if !errors.Is(err, application.ErrStockMovementConstraint) {
		t.Fatalf("Create() error = %v, want constraint error", err)
	}
	if errors.As(err, new(*pq.Error)) {
		t.Fatal("Create() leaked PostgreSQL error")
	}
}

func TestStockMovementRepositoryCreateMapsPersistenceError(t *testing.T) {
	repository, mock, closeDB := newStockMovementRepositoryForTest(t)
	defer closeDB()
	toID := int64(8)
	movement := domain.StockMovement{Type: domain.MovementReceiving, Units: 1, ProductID: uuid.New(), ToID: &toID, CreatedAt: time.Now()}
	mock.ExpectQuery(regexp.QuoteMeta(insertStockMovementSQL)).WillReturnError(errors.New("connection reset"))

	_, err := repository.Create(context.Background(), movement)
	if !errors.Is(err, application.ErrStockMovementPersistence) {
		t.Fatalf("Create() error = %v, want persistence error", err)
	}
}

func sameOptionalInt(left, right *int64) bool {
	if left == nil || right == nil {
		return left == right
	}
	return *left == *right
}
