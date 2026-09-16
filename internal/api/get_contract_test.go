package api

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/anton415/sharetrip-contract/internal/repository"
	"github.com/anton415/sharetrip-contract/internal/service"
	"github.com/google/uuid"
)

func TestGetContractHTTPMapping(t *testing.T) {
	t.Parallel()
	for _, status := range []string{"draft", "active", "suspended", "terminated"} {
		t.Run(status, func(t *testing.T) {
			t.Parallel()
			id, clientID := uuid.New(), uuid.New()
			createdAt := time.Date(2026, 9, 7, 10, 0, 0, 0, time.UTC)
			expires := createdAt.AddDate(1, 0, 0)
			contract := service.GetContractResponse{
				ID: id, ClientID: clientID, Status: status, CreatedAt: createdAt, UpdatedAt: createdAt.Add(time.Hour),
			}
			if status == "active" {
				contract.ExpiredAt = &expires
			}
			calls := 0
			stub := stubContractService{get: func(_ context.Context, request service.GetContractRequest) (service.GetContractResponse, error) {
				calls++
				if request.ContractID != id {
					t.Errorf("ContractID = %v, want %v", request.ContractID, id)
				}
				return contract, nil
			}}
			assertJSONResponse(t, getHTTP(t, testApp(stub), "/contracts/"+id.String()), 200, map[string]any{"contract": map[string]any{
				"id": id, "client_id": clientID, "status": status, "created_at": createdAt,
				"updated_at": contract.UpdatedAt, "expired_at": contract.ExpiredAt,
			}})
			if calls != 1 {
				t.Errorf("service calls = %d, want 1", calls)
			}
		})
	}
}

func TestGetContractHTTPErrors(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name, id, message string
		err               error
		status, calls     int
	}{
		{"malformed", "invalid", "invalid contract id", nil, 400, 0},
		{"zero", uuid.Nil.String(), "invalid contract id", nil, 400, 0},
		{"missing", uuid.NewString(), "contract not found", repository.ErrContractNotFound, 404, 1},
		{"internal", uuid.NewString(), "internal server error", errors.New("private database detail"), 500, 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			calls := 0
			stub := stubContractService{get: func(context.Context, service.GetContractRequest) (service.GetContractResponse, error) {
				calls++
				return service.GetContractResponse{}, fmt.Errorf("operation failed: %w", tt.err)
			}}
			response := getHTTP(t, testApp(stub), "/contracts/"+tt.id)
			assertJSONResponse(t, response, tt.status, map[string]string{"error": tt.message})
			if calls != tt.calls {
				t.Errorf("service calls = %d, want %d", calls, tt.calls)
			}
		})
	}
}
