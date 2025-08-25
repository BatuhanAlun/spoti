-- +goose Up
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS t_roles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4()
);

CREATE TABLE IF NOT EXISTS t_users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    role_id UUID REFERENCES t_roles(id),
    hesap_turu VARCHAR(50) DEFAULT 'Free',
    cash DECIMAL(10, 2) DEFAULT 100.00
);

-- +goose Down
DROP TABLE IF EXISTS t_users;
DROP TABLE IF EXISTS t_roles;