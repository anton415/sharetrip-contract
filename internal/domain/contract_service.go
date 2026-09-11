package domain

type ContractService struct {
	ServiceCode string
	Enabled     bool
}

func ValidateContractServices(services []ContractService) error {
	if len(services) == 0 {
		return ErrInvalidContractServices
	}
	seen := make(map[string]bool, len(services))
	for _, item := range services {
		switch item.ServiceCode {
		case "trip_creation", "trip_participants", "notifications":
		default:
			return ErrInvalidContractServices
		}
		if seen[item.ServiceCode] {
			return ErrInvalidContractServices
		}
		seen[item.ServiceCode] = true
	}
	return nil
}
