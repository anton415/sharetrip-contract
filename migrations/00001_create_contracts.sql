-- +goose Up
-- +goose StatementBegin

CREATE TABLE contracts (
    id UUID PRIMARY KEY,
    company_id UUID NOT NULL,
    status TEXT NOT NULL DEFAULT 'draft',
    valid_from TIMESTAMPTZ NOT NULL,
    valid_until TIMESTAMPTZ,
    CONSTRAINT contracts_status_check CHECK (
        status IN ('draft', 'active', 'suspended', 'terminated')
    ),
    CONSTRAINT contracts_validity_period_check CHECK (
        valid_until IS NULL OR valid_until > valid_from
    )
);

CREATE INDEX contracts_company_id_idx
    ON contracts (company_id);

CREATE UNIQUE INDEX contracts_one_active_per_company_idx
    ON contracts (company_id)
    WHERE status = 'active';

CREATE TABLE contract_services (
    contract_id UUID NOT NULL,
    service_code TEXT NOT NULL,
    enabled BOOLEAN NOT NULL,
    CONSTRAINT contract_services_pkey PRIMARY KEY (contract_id, service_code),
    CONSTRAINT contract_services_contract_id_fkey FOREIGN KEY (contract_id)
        REFERENCES contracts (id) ON DELETE CASCADE,
    CONSTRAINT contract_services_code_check CHECK (
        service_code IN ('trip_creation', 'trip_participants', 'notifications')
    )
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS contract_services;
DROP TABLE IF EXISTS contracts;

-- +goose StatementEnd
