package domain

import (
	"time"

	"github.com/google/uuid"
)

type Contract struct {
	id        ContractID
	clientID  ClientID
	status    ContractStatus
	createdAt time.Time
	updatedAt time.Time
	expiredAt *time.Time
}

func (c Contract) ID() ContractID {
	return c.id
}

func (c Contract) Status() ContractStatus {
	return c.status
}

func (c Contract) ClientID() ClientID {
	return c.clientID
}

func (c Contract) CreatedAt() time.Time {
	return c.createdAt
}

func (c Contract) UpdatedAt() time.Time {
	return c.updatedAt
}

func (c Contract) ExpiredAt() *time.Time {
	if c.expiredAt == nil {
		return nil
	}

	value := *c.expiredAt
	return &value
}

func (c *Contract) Sign(now time.Time) error {
	if c.status != ContractStatusDraft {
		return ErrInvalidContractTransition
	}

	c.status = ContractStatusActive
	c.updatedAt = now.UTC()

	return nil
}

func RestoreContract(
	id ContractID,
	clientID ClientID,
	status ContractStatus,
	createdAt time.Time,
	updatedAt time.Time,
	expiredAt *time.Time,
) (Contract, error) {
	if id.Value() == uuid.Nil {
		return Contract{}, ErrInvalidContractID
	}
	if clientID.Value() == uuid.Nil {
		return Contract{}, ErrInvalidClientID
	}
	if _, err := NewContractStatus(string(status)); err != nil {
		return Contract{}, err
	}
	var expiration *time.Time
	if expiredAt != nil {
		value := *expiredAt
		expiration = &value
	}
	return Contract{
		id:        id,
		clientID:  clientID,
		status:    status,
		createdAt: createdAt,
		updatedAt: updatedAt,
		expiredAt: expiration,
	}, nil
}
