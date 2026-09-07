package domain

import "time"

type SignContractRequest struct {
	Contract Contract
}

type SignContractResponse struct {
	Contract Contract
}

func SignContract(
	request SignContractRequest,
) (SignContractResponse, error) {
	contract := request.Contract

	if err := contract.Sign(time.Now().UTC()); err != nil {
		return SignContractResponse{}, err
	}

	return SignContractResponse{
		Contract: contract,
	}, nil
}
