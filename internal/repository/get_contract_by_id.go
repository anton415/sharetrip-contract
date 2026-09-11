package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/anton415/sharetrip-contract/internal/domain"
	"github.com/anton415/sharetrip-contract/internal/repository/entity"
	"github.com/jackc/pgx/v5"
)

func (r *ContractRepository) GetByID(
	ctx context.Context,
	id domain.ContractID,
) (domain.Contract, error) {
	const query = `
		SELECT id, client_id, status,
		       created_at, updated_at, expired_at
		FROM contracts
		WHERE id = $1
	`

	return scanContract(r.tx.QueryRow(ctx, query, id.Value()))
}

func (r *ContractRepository) GetActiveByClientID(
	ctx context.Context,
	id domain.ClientID,
) (domain.Contract, error) {
	const query = `
		SELECT id, client_id, status,
		       created_at, updated_at, expired_at
		FROM contracts
		WHERE client_id = $1 AND status = 'active'
	`

	return scanContract(r.tx.QueryRow(ctx, query, id.Value()))
}

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

	return scanContract(r.tx.QueryRow(ctx, query, id.Value()))
}

func scanContract(result pgx.Row) (domain.Contract, error) {
	var row entity.Contract
	err := result.Scan(
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
