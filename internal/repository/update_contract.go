package repository

import (
	"context"
	"fmt"

	"github.com/anton415/sharetrip-contract/internal/domain"
)

func (r *ContractRepository) Update(
	ctx context.Context,
	contract domain.Contract,
) error {
	row := toEntityContract(contract)

	const query = `
		UPDATE contracts
		SET status = $2,
		    updated_at = $3
		WHERE id = $1
	`

	cmdTag, err := r.tx.Exec(ctx, query,
		row.ID,
		row.Status,
		row.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("update contract: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return ErrContractNotFound
	}

	return nil
}
