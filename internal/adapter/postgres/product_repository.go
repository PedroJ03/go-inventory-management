package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/pedroj/go-inventory-management/internal/application"
	"github.com/pedroj/go-inventory-management/internal/domain"
)

const (
	insertProductSQL         = `INSERT INTO products (id, name, brand, amount, unit) VALUES ($1, $2, $3, $4, $5)`
	findProductByIDSQL       = `SELECT id, name, brand, amount, unit FROM products WHERE id = $1`
	findProductByIdentitySQL = `SELECT id, name, brand, amount, unit FROM products WHERE lower(name) = lower($1) AND lower(brand) = lower($2) AND amount = $3 AND unit = $4`
)

// ProductRepository persists products using explicit PostgreSQL SQL.
type ProductRepository struct {
	db *sql.DB
}

var _ application.ProductRepository = (*ProductRepository)(nil)

func NewProductRepository(db *sql.DB) (*ProductRepository, error) {
	if db == nil {
		return nil, errors.New("product repository database must not be nil")
	}
	return &ProductRepository{db: db}, nil
}

func (r *ProductRepository) Create(ctx context.Context, product domain.Product) error {
	if err := product.Validate(); err != nil {
		return fmt.Errorf("validate product: %w", err)
	}

	identity := product.Identity
	_, err := r.db.ExecContext(ctx, insertProductSQL, identity.ID, identity.Name, identity.Brand, identity.Amount, identity.Unit)
	if err != nil {
		return mapProductWriteError(err)
	}
	return nil
}

func (r *ProductRepository) FindByID(ctx context.Context, id uuid.UUID) (domain.Product, error) {
	if id == uuid.Nil {
		return domain.Product{}, fmt.Errorf("find product by id: %w: product id must not be nil", application.ErrInvalidProduct)
	}
	return r.findOne(ctx, findProductByIDSQL, id)
}

func (r *ProductRepository) FindByIdentity(ctx context.Context, name, brand string, amount float64, unit domain.Unit) (domain.Product, error) {
	return r.findOne(ctx, findProductByIdentitySQL, name, brand, amount, unit)
}

type rowScanner interface {
	Scan(dest ...any) error
}

func (r *ProductRepository) findOne(ctx context.Context, query string, args ...any) (domain.Product, error) {
	row := r.db.QueryRowContext(ctx, query, args...)
	product, err := scanProduct(row)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Product{}, application.ErrProductNotFound
	}
	if err != nil {
		return domain.Product{}, fmt.Errorf("scan product: %w", err)
	}
	return product, nil
}

func scanProduct(row rowScanner) (domain.Product, error) {
	var idText, name, brand, unitText string
	var amount float64
	if err := row.Scan(&idText, &name, &brand, &amount, &unitText); err != nil {
		return domain.Product{}, err
	}

	id, err := uuid.Parse(idText)
	if err != nil {
		return domain.Product{}, fmt.Errorf("parse product id: %w", err)
	}
	product := domain.Product{Identity: domain.ProductIdentity{
		ID: id, Name: name, Brand: brand, Amount: amount, Unit: domain.Unit(unitText),
	}}
	if err := product.Validate(); err != nil {
		return domain.Product{}, fmt.Errorf("validate database product: %w", err)
	}
	return product, nil
}

func mapProductWriteError(err error) error {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		switch pqErr.Code {
		case "23505":
			return fmt.Errorf("%w: %v", application.ErrProductAlreadyExists, err)
		case "23502", "23514", "22P02":
			return fmt.Errorf("%w: %v", application.ErrProductConstraint, err)
		}
	}
	return fmt.Errorf("persist product: %w", err)
}
