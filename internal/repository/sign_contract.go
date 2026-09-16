package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/anton415/sharetrip-contract/internal/domain"
)

func (r *ContractRepository) SignContract(
	ctx context.Context,
	id domain.ContractID,
	signedAt time.Time,
) error {
	const query = `
		UPDATE contracts
		SET status = 'active',
		    updated_at = $2
		WHERE id = $1
	`

	cmdTag, err := r.tx.Exec(ctx, query, id.Value(), signedAt)
	if err != nil {
		return fmt.Errorf("sign contract: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return ErrContractNotFound
	}

	return nil
}
