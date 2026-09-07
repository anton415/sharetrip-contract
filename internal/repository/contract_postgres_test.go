package repository

import (
	"context"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/anton415/sharetrip-contract/internal/domain"
	"github.com/anton415/sharetrip-contract/internal/repository/entity"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestRepositoryPostgresCreateAndSign(t *testing.T) {
	for _, withExpiration := range []bool{false, true} {
		name := "without expiration"
		if withExpiration {
			name = "with expiration"
		}
		t.Run(name, func(t *testing.T) {
			conn := newPostgresTestConn(t)
			ctx := t.Context()
			row := contractRowForTest()
			if !withExpiration {
				row.ExpiredAt = nil
			}
			contract, err := toDomainContract(row)
			if err != nil {
				t.Fatal(err)
			}
			tx := beginTestTx(t, conn)
			repo := NewContractRepository(tx)
			if err := repo.Create(ctx, contract); err != nil {
				t.Fatalf("Create(): %v", err)
			}
			loaded, err := repo.GetByIDForUpdate(ctx, contract.ID())
			if err != nil {
				t.Fatalf("GetByIDForUpdate(): %v", err)
			}
			assertStoredContract(t, toEntityContract(loaded), row)
			if err := tx.Commit(ctx); err != nil {
				t.Fatalf("commit creation: %v", err)
			}

			// Signing is a separate transaction, as it will be in service.
			tx = beginTestTx(t, conn)
			repo = NewContractRepository(tx)
			loaded, err = repo.GetByIDForUpdate(ctx, contract.ID())
			if err != nil {
				t.Fatalf("load committed contract: %v", err)
			}
			signedAt := row.UpdatedAt.Add(time.Hour)
			if err := loaded.Sign(signedAt); err != nil {
				t.Fatalf("Sign(): %v", err)
			}
			if err := repo.Update(ctx, loaded); err != nil {
				t.Fatalf("Update(): %v", err)
			}
			if err := tx.Commit(ctx); err != nil {
				t.Fatalf("commit signing: %v", err)
			}

			repo = NewContractRepository(beginTestTx(t, conn))
			loaded, err = repo.GetByIDForUpdate(ctx, contract.ID())
			if err != nil {
				t.Fatalf("load committed update: %v", err)
			}
			row.Status = "active"
			row.UpdatedAt = signedAt
			assertStoredContract(t, toEntityContract(loaded), row)
		})
	}
}

func TestRepositoryPostgresNotFound(t *testing.T) {
	conn := newPostgresTestConn(t)
	repo := NewContractRepository(beginTestTx(t, conn))
	contract := postgresTestContract(t)
	got, err := repo.GetByIDForUpdate(t.Context(), contract.ID())
	if !errors.Is(err, ErrContractNotFound) {
		t.Fatalf("GetByIDForUpdate() error = %v, want %v", err, ErrContractNotFound)
	}
	if !reflect.DeepEqual(got, domain.Contract{}) {
		t.Error("missing contract must return an empty result")
	}
	if err := repo.Update(t.Context(), contract); !errors.Is(err, ErrContractNotFound) {
		t.Errorf("Update() error = %v, want %v", err, ErrContractNotFound)
	}
}

func TestRepositoryPostgresRollback(t *testing.T) {
	t.Run("creation", func(t *testing.T) {
		conn := newPostgresTestConn(t)
		contract := postgresTestContract(t)
		tx := beginTestTx(t, conn)
		if err := NewContractRepository(tx).Create(t.Context(), contract); err != nil {
			t.Fatal(err)
		}
		if err := tx.Rollback(t.Context()); err != nil {
			t.Fatalf("rollback creation: %v", err)
		}
		repo := NewContractRepository(beginTestTx(t, conn))
		if _, err := repo.GetByIDForUpdate(t.Context(), contract.ID()); !errors.Is(err, ErrContractNotFound) {
			t.Errorf("read after rollback error = %v, want %v", err, ErrContractNotFound)
		}
	})

	t.Run("update", func(t *testing.T) {
		conn := newPostgresTestConn(t)
		contract := postgresTestContract(t)
		commitTestContract(t, conn, contract)
		tx := beginTestTx(t, conn)
		repo := NewContractRepository(tx)
		loaded, err := repo.GetByIDForUpdate(t.Context(), contract.ID())
		if err != nil {
			t.Fatal(err)
		}
		if err := loaded.Sign(loaded.UpdatedAt().Add(time.Hour)); err != nil {
			t.Fatal(err)
		}
		if err := repo.Update(t.Context(), loaded); err != nil {
			t.Fatal(err)
		}
		if err := tx.Rollback(t.Context()); err != nil {
			t.Fatalf("rollback update: %v", err)
		}
		repo = NewContractRepository(beginTestTx(t, conn))
		loaded, err = repo.GetByIDForUpdate(t.Context(), contract.ID())
		if err != nil {
			t.Fatal(err)
		}
		assertStoredContract(t, toEntityContract(loaded), toEntityContract(contract))
	})
}

func TestRepositoryPostgresDuplicateID(t *testing.T) {
	conn := newPostgresTestConn(t)
	repo := NewContractRepository(beginTestTx(t, conn))
	contract := postgresTestContract(t)
	if err := repo.Create(t.Context(), contract); err != nil {
		t.Fatal(err)
	}
	err := repo.Create(t.Context(), contract)
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23505" {
		t.Errorf("duplicate Create() error = %v, want wrapped unique violation", err)
	}
}

func TestRepositoryPostgresClosedTransaction(t *testing.T) {
	conn := newPostgresTestConn(t)
	tx := beginTestTx(t, conn)
	repo := NewContractRepository(tx)
	if err := tx.Rollback(t.Context()); err != nil {
		t.Fatal(err)
	}
	contract := postgresTestContract(t)
	if err := repo.Create(t.Context(), contract); !errors.Is(err, pgx.ErrTxClosed) {
		t.Errorf("Create() error = %v, want wrapped ErrTxClosed", err)
	}
	if _, err := repo.GetByIDForUpdate(t.Context(), contract.ID()); !errors.Is(err, pgx.ErrTxClosed) {
		t.Errorf("GetByIDForUpdate() error = %v, want wrapped ErrTxClosed", err)
	}
	if err := repo.Update(t.Context(), contract); !errors.Is(err, pgx.ErrTxClosed) {
		t.Errorf("Update() error = %v, want wrapped ErrTxClosed", err)
	}
}

func TestRepositoryPostgresLocksContract(t *testing.T) {
	conn := newPostgresTestConn(t)
	contract := postgresTestContract(t)
	commitTestContract(t, conn, contract)
	tx := beginTestTx(t, conn)
	repo := NewContractRepository(tx)
	loaded, err := repo.GetByIDForUpdate(t.Context(), contract.ID())
	if err != nil {
		t.Fatal(err)
	}

	other, err := pgx.ConnectConfig(t.Context(), conn.Config().Copy())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = other.Close(context.Background()) })
	if _, err := other.Exec(t.Context(), "SET lock_timeout = '100ms'"); err != nil {
		t.Fatal(err)
	}
	otherTx := beginTestTx(t, other)
	_, err = NewContractRepository(otherTx).GetByIDForUpdate(t.Context(), contract.ID())
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "55P03" {
		t.Fatalf("concurrent read error = %v, want lock timeout", err)
	}
	if err := otherTx.Rollback(t.Context()); err != nil {
		t.Fatal(err)
	}
	if err := loaded.Sign(loaded.UpdatedAt().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := repo.Update(t.Context(), loaded); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(t.Context()); err != nil {
		t.Fatal(err)
	}
	otherRepo := NewContractRepository(beginTestTx(t, other))
	loaded, err = otherRepo.GetByIDForUpdate(t.Context(), contract.ID())
	if err != nil {
		t.Fatalf("read after lock release: %v", err)
	}
	if err := loaded.Sign(loaded.UpdatedAt().Add(time.Hour)); !errors.Is(err, domain.ErrInvalidContractTransition) {
		t.Errorf("second signing error = %v, want invalid transition", err)
	}
}

func newPostgresTestConn(t *testing.T) *pgx.Conn {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	config, err := pgx.ParseConfig(dsn)
	if err != nil {
		t.Fatal("invalid TEST_DATABASE_URL")
	}
	schema := "contract_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	config.RuntimeParams["search_path"] = schema
	config.RuntimeParams["statement_timeout"] = "5s"
	conn, err := pgx.ConnectConfig(t.Context(), config)
	if err != nil {
		t.Fatalf("connect test PostgreSQL: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, err := conn.Exec(ctx, "DROP SCHEMA "+pgx.Identifier{schema}.Sanitize()+" CASCADE"); err != nil {
			t.Errorf("drop test schema: %v", err)
		}
		_ = conn.Close(ctx)
	})
	if _, err := conn.Exec(t.Context(), "CREATE SCHEMA "+pgx.Identifier{schema}.Sanitize()); err != nil {
		t.Fatalf("create test schema: %v", err)
	}
	migration, err := os.ReadFile("../../migrations/00001_create_contracts.sql")
	if err != nil {
		t.Fatal(err)
	}
	up, _, found := strings.Cut(string(migration), "-- +goose Down")
	if !found {
		t.Fatal("migration is missing the Down boundary")
	}
	if _, err := conn.Exec(t.Context(), up); err != nil {
		t.Fatalf("apply migration to test schema: %v", err)
	}
	return conn
}

func beginTestTx(t *testing.T, conn *pgx.Conn) pgx.Tx {
	t.Helper()
	tx, err := conn.Begin(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			t.Errorf("rollback test transaction: %v", err)
		}
	})
	return tx
}

func postgresTestContract(t *testing.T) domain.Contract {
	t.Helper()
	contract, err := toDomainContract(contractRowForTest())
	if err != nil {
		t.Fatal(err)
	}
	return contract
}

func commitTestContract(t *testing.T, conn *pgx.Conn, contract domain.Contract) {
	t.Helper()
	tx := beginTestTx(t, conn)
	if err := NewContractRepository(tx).Create(t.Context(), contract); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(t.Context()); err != nil {
		t.Fatal(err)
	}
}

func assertStoredContract(t *testing.T, got, want entity.Contract) {
	t.Helper()
	// TIMESTAMPTZ preserves the instant, not the original Go time.Location.
	if got.ID != want.ID || got.ClientID != want.ClientID || got.Status != want.Status ||
		!got.CreatedAt.Equal(want.CreatedAt) || !got.UpdatedAt.Equal(want.UpdatedAt) {
		t.Fatalf("stored contract = %+v, want %+v", got, want)
	}
	if (got.ExpiredAt == nil) != (want.ExpiredAt == nil) {
		t.Fatalf("ExpiredAt = %v, want %v", got.ExpiredAt, want.ExpiredAt)
	}
	if got.ExpiredAt != nil && !got.ExpiredAt.Equal(*want.ExpiredAt) {
		t.Fatalf("ExpiredAt = %v, want %v", got.ExpiredAt, want.ExpiredAt)
	}
}
