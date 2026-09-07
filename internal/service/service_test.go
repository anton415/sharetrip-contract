package service

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/anton415/sharetrip-contract/internal/domain"
)

func TestCreateContractRejectsEmptyClientBeforeDatabase(t *testing.T) {
	t.Parallel()
	response, err := NewService(nil).CreateContract(context.Background(), CreateContractRequest{})
	if !errors.Is(err, domain.ErrInvalidClientID) {
		t.Fatalf("error = %v, want ErrInvalidClientID", err)
	}
	if !reflect.DeepEqual(response, CreateContractResponse{}) {
		t.Errorf("response = %+v, want empty response", response)
	}
}

func TestSignContractRejectsEmptyIDBeforeDatabase(t *testing.T) {
	t.Parallel()
	response, err := NewService(nil).SignContract(context.Background(), SignContractRequest{})
	if !errors.Is(err, domain.ErrInvalidContractID) {
		t.Fatalf("error = %v, want ErrInvalidContractID", err)
	}
	if !reflect.DeepEqual(response, SignContractResponse{}) {
		t.Errorf("response = %+v, want empty response", response)
	}
}
