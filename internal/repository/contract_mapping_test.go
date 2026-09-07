package repository

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/anton415/sharetrip-contract/internal/domain"
	"github.com/anton415/sharetrip-contract/internal/repository/entity"
	"github.com/google/uuid"
)

func TestContractMappingRoundTrip(t *testing.T) {
	t.Parallel()
	for _, status := range []string{"draft", "active", "suspended", "terminated"} {
		t.Run(status, func(t *testing.T) {
			t.Parallel()
			for _, withExpiration := range []bool{false, true} {
				name := "without expiration"
				if withExpiration {
					name = "with expiration"
				}
				t.Run(name, func(t *testing.T) {
					t.Parallel()
					want := contractRowForTest()
					want.Status = status
					if !withExpiration {
						want.ExpiredAt = nil
					}
					contract, err := toDomainContract(want)
					if err != nil {
						t.Fatalf("toDomainContract(): %v", err)
					}
					if contract.ID().Value() != want.ID || contract.ClientID().Value() != want.ClientID ||
						string(contract.Status()) != want.Status {
						t.Error("mapping to domain changed identifiers or status")
					}
					if !contract.CreatedAt().Equal(want.CreatedAt) || !contract.UpdatedAt().Equal(want.UpdatedAt) {
						t.Error("mapping to domain changed timestamps")
					}
					got := toEntityContract(contract)
					if !reflect.DeepEqual(got, want) {
						t.Errorf("round trip = %+v, want %+v", got, want)
					}
				})
			}
		})
	}
}

func TestToDomainContractRejectsInvalidRow(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		change  func(*entity.Contract)
		wantErr error
	}{
		{"empty contract id", func(row *entity.Contract) { row.ID = uuid.Nil }, domain.ErrInvalidContractID},
		{"empty client id", func(row *entity.Contract) { row.ClientID = uuid.Nil }, domain.ErrInvalidClientID},
		{"empty status", func(row *entity.Contract) { row.Status = "" }, domain.ErrInvalidContractStatus},
		{"unknown status", func(row *entity.Contract) { row.Status = "unknown" }, domain.ErrInvalidContractStatus},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			row := contractRowForTest()
			tt.change(&row)
			got, err := toDomainContract(row)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("toDomainContract() error = %v, want wrapped %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(got, domain.Contract{}) {
				t.Errorf("toDomainContract() = %+v, want empty contract", got)
			}
		})
	}
}

func TestContractMappingCopiesExpiration(t *testing.T) {
	t.Parallel()
	t.Run("input entity cannot change domain", func(t *testing.T) {
		t.Parallel()
		row := contractRowForTest()
		want := *row.ExpiredAt
		contract, err := toDomainContract(row)
		if err != nil {
			t.Fatalf("toDomainContract(): %v", err)
		}
		*row.ExpiredAt = row.ExpiredAt.Add(time.Hour)
		if contract.ExpiredAt() == nil || !contract.ExpiredAt().Equal(want) {
			t.Error("changing input entity expiration changed the domain contract")
		}
	})

	t.Run("output entity cannot change domain", func(t *testing.T) {
		t.Parallel()
		row := contractRowForTest()
		want := *row.ExpiredAt
		contract, err := toDomainContract(row)
		if err != nil {
			t.Fatalf("toDomainContract(): %v", err)
		}
		mapped := toEntityContract(contract)
		if mapped.ExpiredAt == nil {
			t.Fatal("toEntityContract() lost expiration")
		}
		*mapped.ExpiredAt = mapped.ExpiredAt.Add(time.Hour)
		if contract.ExpiredAt() == nil || !contract.ExpiredAt().Equal(want) {
			t.Error("changing output entity expiration changed the domain contract")
		}
	})
}

func contractRowForTest() entity.Contract {
	createdAt := time.Date(2026, 9, 7, 10, 0, 0, 0, time.UTC)
	expiredAt := createdAt.Add(24 * time.Hour)
	return entity.Contract{
		ID:        uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
		ClientID:  uuid.MustParse("550e8400-e29b-41d4-a716-446655440001"),
		Status:    "draft",
		CreatedAt: createdAt,
		UpdatedAt: createdAt.Add(time.Hour),
		ExpiredAt: &expiredAt,
	}
}
