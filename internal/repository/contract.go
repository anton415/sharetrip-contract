package repository

import "github.com/jackc/pgx/v5"

type ContractRepository struct {
	tx pgx.Tx
}

func NewContractRepository(tx pgx.Tx) *ContractRepository {
	return &ContractRepository{tx: tx}
}
