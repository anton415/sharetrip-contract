package repository

import (
	"context"
	"fmt"

	"github.com/anton415/sharetrip-contract/internal/domain"
)

func (r *ContractRepository) UpsertServices(
	ctx context.Context,
	id domain.ContractID,
	services []domain.ContractService,
) error {
	const query = `
		INSERT INTO contract_services (contract_id, service_code, enabled)
		VALUES ($1, $2, $3)
		ON CONFLICT (contract_id, service_code)
		DO UPDATE SET enabled = EXCLUDED.enabled
	`

	for _, service := range services {
		_, err := r.tx.Exec(ctx, query, id.Value(), service.ServiceCode, service.Enabled)
		if err != nil {
			return fmt.Errorf("upsert contract service: %w", err)
		}
	}

	return nil
}
