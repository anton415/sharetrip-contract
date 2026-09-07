package repository

import (
	"context"
	"fmt"

	"github.com/anton415/sharetrip-contract/internal/domain"
)

func (r *ContractRepository) Create(
	ctx context.Context,
	contract domain.Contract,
) error {
	row := toEntityContract(contract)

	const query = `
		INSERT INTO contracts (
			id, client_id, status,
			created_at, updated_at, expired_at
		)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := r.tx.Exec(ctx, query,
		row.ID,
		row.ClientID,
		row.Status,
		row.CreatedAt,
		row.UpdatedAt,
		row.ExpiredAt,
	)
	if err != nil {
		return fmt.Errorf("insert contract: %w", err)
	}

	return nil
}
