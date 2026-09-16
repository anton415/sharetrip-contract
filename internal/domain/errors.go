package domain

import "errors"

var (
	ErrInvalidClientID           = errors.New("invalid client id")
	ErrInvalidContractID         = errors.New("invalid contract id")
	ErrInvalidContractTransition = errors.New("invalid contract state transition")
	ErrContractNotDraft          = errors.New("contract is not a draft")
	ErrInvalidContractStatus     = errors.New("invalid contract status")
	ErrInvalidContractServices   = errors.New("invalid contract services")
)
