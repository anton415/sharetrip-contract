package domain

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestCreateContract(t *testing.T) {
	clientID, err := NewClientID(uuid.New())
	if err != nil {
		t.Fatalf("NewClientID(): %v", err)
	}
	expiredAt := time.Now().UTC().Add(24 * time.Hour)
	tests := []struct {
		name      string
		expiredAt *time.Time
	}{
		{name: "without expiration"},
		{name: "with expiration", expiredAt: &expiredAt},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := time.Now().UTC()
			response, err := CreateContract(CreateContractRequest{
				ClientID:  clientID,
				ExpiredAt: tt.expiredAt,
			})
			after := time.Now().UTC()
			if err != nil {
				t.Fatalf("CreateContract(): %v", err)
			}
			if response.ID.Value() == uuid.Nil {
				t.Error("created contract has an empty ID")
			}
			if response.ClientID != clientID {
				t.Errorf("ClientID = %v, want %v", response.ClientID, clientID)
			}
			if response.Status != ContractStatusDraft {
				t.Errorf("Status = %q, want %q", response.Status, ContractStatusDraft)
			}
			if response.CreatedAt.Before(before) || response.CreatedAt.After(after) {
				t.Errorf("CreatedAt = %v, want between %v and %v", response.CreatedAt, before, after)
			}
			if !response.UpdatedAt.Equal(response.CreatedAt) {
				t.Errorf("UpdatedAt = %v, want CreatedAt %v", response.UpdatedAt, response.CreatedAt)
			}
			if response.CreatedAt.Location() != time.UTC || response.UpdatedAt.Location() != time.UTC {
				t.Error("creation and update timestamps must use UTC")
			}
			if tt.expiredAt == nil {
				if response.ExpiredAt != nil {
					t.Errorf("ExpiredAt = %v, want nil", response.ExpiredAt)
				}
				return
			}
			if response.ExpiredAt == nil || !response.ExpiredAt.Equal(*tt.expiredAt) {
				t.Fatalf("ExpiredAt = %v, want %v", response.ExpiredAt, tt.expiredAt)
			}
			wantExpiration := *tt.expiredAt
			*response.ExpiredAt = response.ExpiredAt.Add(time.Hour)
			if !tt.expiredAt.Equal(wantExpiration) {
				t.Error("changing response expiration changed the request")
			}
		})
	}
}

func TestCreateContractRejectsEmptyClientID(t *testing.T) {
	response, err := CreateContract(CreateContractRequest{})
	if !errors.Is(err, ErrInvalidClientID) {
		t.Fatalf("CreateContract() error = %v, want %v", err, ErrInvalidClientID)
	}
	if !reflect.DeepEqual(response, CreateContractResponse{}) {
		t.Errorf("CreateContract() response = %+v, want empty response", response)
	}
}
