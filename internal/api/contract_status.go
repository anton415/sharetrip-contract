package api

import (
	"fmt"

	"github.com/anton415/sharetrip-contract/gen"
)

func toAPIContractStatus(value string) (gen.ContractStatus, error) {
	status := gen.ContractStatus(value)
	if !status.Valid() {
		return "", fmt.Errorf("unsupported API contract status: %q", value)
	}

	return status, nil
}
