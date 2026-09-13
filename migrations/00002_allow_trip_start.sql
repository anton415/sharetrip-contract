-- +goose Up
ALTER TABLE contract_services DROP CONSTRAINT contract_services_code_check,
    ADD CONSTRAINT contract_services_code_check CHECK (
        service_code IN ('trip_creation', 'trip_participants', 'notifications', 'trip_start'));

-- +goose Down
ALTER TABLE contract_services DROP CONSTRAINT contract_services_code_check,
    ADD CONSTRAINT contract_services_code_check CHECK (
        service_code IN ('trip_creation', 'trip_participants', 'notifications'));
