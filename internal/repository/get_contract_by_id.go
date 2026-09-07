package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/anton415/sharetrip-contract/internal/domain"
	"github.com/anton415/sharetrip-contract/internal/repository/entity"
	"github.com/jackc/pgx/v5"
)

func (r *ContractRepository) GetByIDForUpdate(
	ctx context.Context,
	id domain.ContractID,
) (domain.Contract, error) {
	const query = `
		SELECT id, client_id, status,
		       created_at, updated_at, expired_at
		FROM contracts
		WHERE id = $1
		FOR UPDATE
	`

	var row entity.Contract
	err := r.tx.QueryRow(ctx, query, id.Value()).Scan(
		&row.ID,
		&row.ClientID,
		&row.Status,
		&row.CreatedAt,
		&row.UpdatedAt,
		&row.ExpiredAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Contract{}, ErrContractNotFound
	}
	if err != nil {
		return domain.Contract{}, fmt.Errorf("select contract: %w", err)
	}

	contract, err := toDomainContract(row)
	if err != nil {
		return domain.Contract{}, fmt.Errorf("restore selected contract: %w", err)
	}

	return contract, nil
}
