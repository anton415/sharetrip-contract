package domain

import "errors"

var (
	ErrInvalidClientID           = errors.New("invalid client id")
	ErrInvalidContractID         = errors.New("invalid contract id")
	ErrInvalidContractTransition = errors.New("invalid contract state transition")
	ErrInvalidContractStatus     = errors.New("invalid contract status")
	ErrInvalidContractServices   = errors.New("invalid contract services")
)
