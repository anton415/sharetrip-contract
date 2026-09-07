package domain

import (
	"errors"
	"reflect"
	"testing"
	"time"
)

func TestRestoreContractPreservesState(t *testing.T) {
	for _, status := range []ContractStatus{
		ContractStatusDraft,
		ContractStatusActive,
		ContractStatusSuspended,
		ContractStatusTerminated,
	} {
		t.Run(string(status), func(t *testing.T) {
			for _, withExpiration := range []bool{false, true} {
				name := "without expiration"
				if withExpiration {
					name = "with expiration"
				}
				t.Run(name, func(t *testing.T) {
					want := draftContractForTest()
					want.status = status
					want.updatedAt = want.createdAt.Add(time.Hour)
					if !withExpiration {
						want.expiredAt = nil
					}
					got, err := RestoreContract(want.id, want.clientID, want.status,
						want.createdAt, want.updatedAt, want.expiredAt)
					if err != nil {
						t.Fatalf("RestoreContract(): %v", err)
					}
					if !reflect.DeepEqual(got, want) {
						t.Errorf("RestoreContract() = %+v, want %+v", got, want)
					}
				})
			}
		})
	}
}

func TestRestoreContractRejectsInvalidState(t *testing.T) {
	tests := []struct {
		name    string
		change  func(*Contract)
		wantErr error
	}{
		{"empty contract id", func(c *Contract) { c.id = ContractID{} }, ErrInvalidContractID},
		{"empty client id", func(c *Contract) { c.clientID = ClientID{} }, ErrInvalidClientID},
		{"empty status", func(c *Contract) { c.status = "" }, ErrInvalidContractStatus},
		{"unknown status", func(c *Contract) { c.status = "unknown" }, ErrInvalidContractStatus},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := draftContractForTest()
			tt.change(&state)
			got, err := RestoreContract(state.id, state.clientID, state.status,
				state.createdAt, state.updatedAt, state.expiredAt)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("RestoreContract() error = %v, want %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(got, Contract{}) {
				t.Errorf("RestoreContract() = %+v, want empty contract", got)
			}
		})
	}
}

func TestRestoreContractCopiesExpiration(t *testing.T) {
	state := draftContractForTest()
	wantExpiration := *state.expiredAt
	got, err := RestoreContract(state.id, state.clientID, state.status,
		state.createdAt, state.updatedAt, state.expiredAt)
	if err != nil {
		t.Fatalf("RestoreContract(): %v", err)
	}
	*state.expiredAt = state.expiredAt.Add(time.Hour)
	if got.ExpiredAt() == nil || !got.ExpiredAt().Equal(wantExpiration) {
		t.Error("changing input expiration changed the restored contract")
	}
}

func TestRestoreContractAcceptsDatesFromCreateContract(t *testing.T) {
	// Creation currently accepts an expiration before creation time.
	// Restoring that same state must not introduce a new date restriction.
	expiredAt := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	created, err := CreateContract(CreateContractRequest{
		ClientID:  draftContractForTest().ClientID(),
		ExpiredAt: &expiredAt,
	})
	if err != nil {
		t.Fatalf("CreateContract(): %v", err)
	}
	restored, err := RestoreContract(created.ID, created.ClientID, created.Status,
		created.CreatedAt, created.UpdatedAt, created.ExpiredAt)
	if err != nil {
		t.Fatalf("RestoreContract() rejected a created contract: %v", err)
	}
	if restored.ExpiredAt() == nil || !restored.ExpiredAt().Equal(expiredAt) {
		t.Error("restoration changed expiration")
	}
}
