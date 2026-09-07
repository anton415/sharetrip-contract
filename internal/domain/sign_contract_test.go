package domain

import (
	"errors"
	"reflect"
	"testing"
	"time"
)

func TestSignContract(t *testing.T) {
	request := SignContractRequest{Contract: draftContractForTest()}
	original := request.Contract
	before := time.Now().UTC()
	response, err := SignContract(request)
	after := time.Now().UTC()
	if err != nil {
		t.Fatalf("SignContract(): %v", err)
	}
	if !reflect.DeepEqual(request.Contract, original) {
		t.Error("SignContract() changed the input contract")
	}
	if response.Contract.UpdatedAt().Before(before) || response.Contract.UpdatedAt().After(after) {
		t.Errorf("UpdatedAt = %v, want between %v and %v", response.Contract.UpdatedAt(), before, after)
	}
	want := original
	want.status = ContractStatusActive
	want.updatedAt = response.Contract.UpdatedAt()
	if !reflect.DeepEqual(response.Contract, want) {
		t.Errorf("SignContract() contract = %+v, want %+v", response.Contract, want)
	}
}

func TestSignContractRejectsInvalidTransition(t *testing.T) {
	for _, status := range []ContractStatus{
		ContractStatusActive,
		ContractStatusSuspended,
		ContractStatusTerminated,
	} {
		t.Run(string(status), func(t *testing.T) {
			contract := draftContractForTest()
			contract.status = status
			response, err := SignContract(SignContractRequest{Contract: contract})
			if !errors.Is(err, ErrInvalidContractTransition) {
				t.Fatalf("SignContract() error = %v, want %v", err, ErrInvalidContractTransition)
			}
			if !reflect.DeepEqual(response, SignContractResponse{}) {
				t.Errorf("SignContract() response = %+v, want empty response", response)
			}
		})
	}
}
