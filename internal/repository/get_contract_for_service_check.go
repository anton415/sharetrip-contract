package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/anton415/sharetrip-contract/internal/domain"
	"github.com/anton415/sharetrip-contract/internal/repository/entity"
	"github.com/jackc/pgx/v5"
)

func (r *ContractRepository) GetForServiceCheck(
	ctx context.Context,
	clientID domain.ClientID,
	serviceCode string,
) (domain.Contract, bool, error) {
	const query = `
		SELECT c.id, c.client_id, c.status,
		       c.created_at, c.updated_at, c.expired_at,
		       COALESCE(s.enabled, false)
		FROM contracts c
		LEFT JOIN contract_services s ON s.contract_id = c.id AND s.service_code = $2
		WHERE c.client_id = $1
		ORDER BY (c.status = 'active') DESC
		LIMIT 1
	`

	var row entity.Contract
	var enabled bool
	err := r.tx.QueryRow(ctx, query, clientID.Value(), serviceCode).Scan(
		&row.ID,
		&row.ClientID,
		&row.Status,
		&row.CreatedAt,
		&row.UpdatedAt,
		&row.ExpiredAt,
		&enabled,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Contract{}, false, ErrContractNotFound
	}
	if err != nil {
		return domain.Contract{}, false, fmt.Errorf("select contract for service check: %w", err)
	}

	contract, err := toDomainContract(row)
	if err != nil {
		return domain.Contract{}, false, fmt.Errorf("restore contract for service check: %w", err)
	}

	return contract, enabled, nil
}
