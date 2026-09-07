package domain

type ContractStatus string

const (
	ContractStatusDraft      ContractStatus = "draft"
	ContractStatusActive     ContractStatus = "active"
	ContractStatusSuspended  ContractStatus = "suspended"
	ContractStatusTerminated ContractStatus = "terminated"
)

func NewContractStatus(value string) (ContractStatus, error) {
	status := ContractStatus(value)

	switch status {
	case ContractStatusDraft,
		ContractStatusActive,
		ContractStatusSuspended,
		ContractStatusTerminated:
		return status, nil
	default:
		return "", ErrInvalidContractStatus
	}
}
