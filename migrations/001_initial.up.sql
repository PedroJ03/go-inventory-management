CREATE TABLE products (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL CHECK (btrim(name) <> ''),
    brand TEXT NOT NULL CHECK (btrim(brand) <> ''),
    amount NUMERIC NOT NULL CHECK (amount > 0),
    unit TEXT NOT NULL CHECK (unit IN ('each', 'milliliters', 'liters', 'grams', 'kilograms')),
    CHECK ((unit = 'each' AND amount = 1) OR unit <> 'each')
);

CREATE UNIQUE INDEX products_business_identity_idx
    ON products (lower(name), lower(brand), amount, unit);

CREATE TABLE locations (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    street TEXT NOT NULL,
    city TEXT NOT NULL,
    active BOOLEAN NOT NULL DEFAULT TRUE
);

CREATE TABLE stock_levels (
    product_id UUID NOT NULL REFERENCES products(id),
    location_id BIGINT NOT NULL REFERENCES locations(id),
    units BIGINT NOT NULL CHECK (units >= 0),
    PRIMARY KEY (product_id, location_id)
);

CREATE TABLE stock_movements (
    id BIGSERIAL PRIMARY KEY,
    type TEXT NOT NULL CHECK (type IN ('receiving', 'selling', 'transferring')),
    units BIGINT NOT NULL CHECK (units > 0),
    product_id UUID NOT NULL REFERENCES products(id),
    from_location_id BIGINT REFERENCES locations(id),
    to_location_id BIGINT REFERENCES locations(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CHECK (from_location_id IS NULL OR to_location_id IS NULL OR from_location_id <> to_location_id),
    CHECK (
        (type = 'receiving' AND from_location_id IS NULL AND to_location_id IS NOT NULL) OR
        (type = 'selling' AND from_location_id IS NOT NULL AND to_location_id IS NULL) OR
        (type = 'transferring' AND from_location_id IS NOT NULL AND to_location_id IS NOT NULL)
    )
);
