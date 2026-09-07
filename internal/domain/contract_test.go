package domain

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestContractSign(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		status  ContractStatus
		wantErr error
	}{
		{name: "draft", status: ContractStatusDraft},
		{name: "active", status: ContractStatusActive, wantErr: ErrInvalidContractTransition},
		{name: "suspended", status: ContractStatusSuspended, wantErr: ErrInvalidContractTransition},
		{name: "terminated", status: ContractStatusTerminated, wantErr: ErrInvalidContractTransition},
		{name: "empty status", wantErr: ErrInvalidContractTransition},
		{name: "unknown status", status: ContractStatus("unknown"), wantErr: ErrInvalidContractTransition},
	}
	signedAt := time.Date(2026, 9, 7, 14, 0, 0, 0, time.FixedZone("UTC+3", 3*60*60))

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			contract := draftContractForTest()
			contract.status = tt.status
			want := contract
			if tt.wantErr == nil {
				want.status = ContractStatusActive
				want.updatedAt = signedAt.UTC()
			}

			err := contract.Sign(signedAt)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Sign() error = %v, want %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(contract, want) {
				t.Errorf("Sign() contract = %+v, want %+v", contract, want)
			}
		})
	}
}

func TestContractSignTwice(t *testing.T) {
	t.Parallel()
	contract := draftContractForTest()
	signedAt := contract.CreatedAt().Add(time.Hour)
	if err := contract.Sign(signedAt); err != nil {
		t.Fatalf("first Sign(): %v", err)
	}
	want := contract

	if err := contract.Sign(signedAt.Add(time.Hour)); !errors.Is(err, ErrInvalidContractTransition) {
		t.Fatalf("second Sign() error = %v, want %v", err, ErrInvalidContractTransition)
	}
	if !reflect.DeepEqual(contract, want) {
		t.Errorf("second Sign() changed contract: got %+v, want %+v", contract, want)
	}
}

func TestContractExpiredAt(t *testing.T) {
	t.Parallel()
	t.Run("no expiration", func(t *testing.T) {
		t.Parallel()
		contract := Contract{}
		if got := contract.ExpiredAt(); got != nil {
			t.Errorf("ExpiredAt() = %v, want nil", got)
		}
	})

	t.Run("returns independent copy", func(t *testing.T) {
		t.Parallel()
		contract := draftContractForTest()
		want := *contract.expiredAt
		got := contract.ExpiredAt()
		if got == nil || !got.Equal(want) {
			t.Fatalf("ExpiredAt() = %v, want %v", got, want)
		}
		*got = got.Add(time.Hour)
		if !contract.expiredAt.Equal(want) {
			t.Error("changing returned expiration changed the contract")
		}
	})
}

func draftContractForTest() Contract {
	createdAt := time.Date(2026, 9, 7, 10, 0, 0, 0, time.UTC)
	expiredAt := createdAt.Add(24 * time.Hour)
	return Contract{
		id:        ContractID{value: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")},
		clientID:  ClientID{value: uuid.MustParse("550e8400-e29b-41d4-a716-446655440001")},
		status:    ContractStatusDraft,
		createdAt: createdAt,
		updatedAt: createdAt,
		expiredAt: &expiredAt,
	}
}
