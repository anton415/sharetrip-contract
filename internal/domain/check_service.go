package domain

import "time"

func (c Contract) CheckService(serviceEnabled bool, now time.Time) (bool, string) {
	if c.status != ContractStatusActive {
		return false, "contract_not_active"
	}
	if c.expiredAt != nil && !now.Before(*c.expiredAt) {
		return false, "contract_expired"
	}
	if !serviceEnabled {
		return false, "service_not_allowed"
	}
	return true, "service_allowed"
}
