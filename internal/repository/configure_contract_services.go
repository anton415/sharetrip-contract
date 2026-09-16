package repository

import (
	"context"
	"fmt"

	"github.com/anton415/sharetrip-contract/internal/domain"
)

func (r *ContractRepository) ConfigureServices(
	ctx context.Context,
	id domain.ContractID,
	services []domain.ContractService,
) error {
	const deleteQuery = `
		DELETE FROM contract_services WHERE contract_id = $1
	`
	if _, err := r.tx.Exec(ctx, deleteQuery, id.Value()); err != nil {
		return fmt.Errorf("clear contract services: %w", err)
	}

	const insertQuery = `
		INSERT INTO contract_services (contract_id, service_code, enabled)
		VALUES ($1, $2, $3)
	`

	for _, service := range services {
		_, err := r.tx.Exec(ctx, insertQuery, id.Value(), service.ServiceCode, service.Enabled)
		if err != nil {
			return fmt.Errorf("configure contract services: %w", err)
		}
	}

	return nil
}
