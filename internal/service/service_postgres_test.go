package service

import (
	"context"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/anton415/sharetrip-contract/internal/domain"
	"github.com/anton415/sharetrip-contract/internal/repository"
	"github.com/anton415/sharetrip-contract/internal/repository/entity"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestServicePostgresCreateAndSign(t *testing.T) {
	t.Parallel()
	for _, withExpiration := range []bool{false, true} {
		name := "without expiration"
		if withExpiration {
			name = "with expiration"
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			svc, observer := newPostgresTestService(t)
			request := CreateContractRequest{ClientID: uuid.New()}
			if withExpiration {
				expires := time.Now().UTC().Add(24 * time.Hour).Truncate(time.Microsecond)
				request.ExpiredAt = &expires
			}
			created, err := svc.CreateContract(t.Context(), request)
			if err != nil {
				t.Fatalf("CreateContract(): %v", err)
			}
			if created.ID == uuid.Nil || created.ClientID != request.ClientID || created.Status != "draft" ||
				created.CreatedAt.IsZero() || !created.CreatedAt.Equal(created.UpdatedAt) {
				t.Fatalf("unexpected creation response: %+v", created)
			}
			// A separate connection sees only committed changes.
			stored := readStoredContract(t, observer, created.ID)
			if stored.ClientID != created.ClientID || stored.Status != created.Status ||
				!stored.CreatedAt.Equal(created.CreatedAt.Truncate(time.Microsecond)) ||
				!stored.UpdatedAt.Equal(created.UpdatedAt.Truncate(time.Microsecond)) {
				t.Fatalf("stored contract = %+v, response = %+v", stored, created)
			}
			if (stored.ExpiredAt == nil) != (request.ExpiredAt == nil) ||
				(created.ExpiredAt == nil) != (request.ExpiredAt == nil) {
				t.Fatal("creation changed expiration nullability")
			}
			if request.ExpiredAt != nil && (!stored.ExpiredAt.Equal(*request.ExpiredAt) ||
				!created.ExpiredAt.Equal(*request.ExpiredAt)) {
				t.Fatal("creation changed expiration")
			}

			signed, err := svc.SignContract(t.Context(), SignContractRequest{ContractID: created.ID})
			if err != nil {
				t.Fatalf("SignContract(): %v", err)
			}
			if signed.ID != created.ID || signed.Status != "active" || signed.UpdatedAt.Before(created.CreatedAt) {
				t.Fatalf("unexpected signing response: %+v", signed)
			}
			updated := readStoredContract(t, observer, created.ID)
			want := stored
			want.Status = "active"
			want.UpdatedAt = updated.UpdatedAt
			if !updated.UpdatedAt.Equal(signed.UpdatedAt.Truncate(time.Microsecond)) || !reflect.DeepEqual(updated, want) {
				t.Fatalf("updated contract = %+v, want %+v", updated, want)
			}

			response, err := svc.SignContract(t.Context(), SignContractRequest{ContractID: created.ID})
			if !errors.Is(err, domain.ErrInvalidContractTransition) || response != (SignContractResponse{}) {
				t.Fatalf("second signing = %+v, %v; want empty response and invalid transition", response, err)
			}
			if got := readStoredContract(t, observer, created.ID); !reflect.DeepEqual(got, updated) {
				t.Error("rejected signing changed the contract")
			}
			assertPoolAvailable(t, svc.pool)
		})
	}
}

func TestServicePostgresSignErrors(t *testing.T) {
	t.Parallel()
	for _, state := range []string{"missing", "suspended", "terminated"} {
		t.Run(state, func(t *testing.T) {
			t.Parallel()
			svc, observer := newPostgresTestService(t)
			id := uuid.New()
			wantErr := repository.ErrContractNotFound
			var before entity.Contract
			if state != "missing" {
				created := createTestContract(t, svc, uuid.New())
				id = created.ID
				if _, err := observer.Exec(t.Context(), "UPDATE contracts SET status = $2 WHERE id = $1", id, state); err != nil {
					t.Fatal(err)
				}
				before = readStoredContract(t, observer, id)
				wantErr = domain.ErrInvalidContractTransition
			}
			response, err := svc.SignContract(t.Context(), SignContractRequest{ContractID: id})
			if !errors.Is(err, wantErr) || response != (SignContractResponse{}) {
				t.Fatalf("SignContract() = %+v, %v; want empty response and %v", response, err, wantErr)
			}
			if state != "missing" && !reflect.DeepEqual(readStoredContract(t, observer, id), before) {
				t.Error("rejected signing changed stored data")
			}
			assertPoolAvailable(t, svc.pool)
			createTestContract(t, svc, uuid.New())
		})
	}
}

func TestServicePostgresUpdateFailureRollsBack(t *testing.T) {
	t.Parallel()
	svc, observer := newPostgresTestService(t)
	clientID := uuid.New()
	first := createTestContract(t, svc, clientID)
	second := createTestContract(t, svc, clientID)
	if _, err := svc.SignContract(t.Context(), SignContractRequest{ContractID: first.ID}); err != nil {
		t.Fatal(err)
	}
	before := readStoredContract(t, observer, second.ID)
	response, err := svc.SignContract(t.Context(), SignContractRequest{ContractID: second.ID})
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23505" || response != (SignContractResponse{}) {
		t.Fatalf("signing second active contract = %+v, %v; want unique violation", response, err)
	}
	if got := readStoredContract(t, observer, second.ID); !reflect.DeepEqual(got, before) {
		t.Error("failed update changed the second contract")
	}
	assertPoolAvailable(t, svc.pool)
	createTestContract(t, svc, uuid.New())
}

func TestServicePostgresCommitFailure(t *testing.T) {
	t.Parallel()
	for _, operation := range []string{"create", "sign"} {
		t.Run(operation, func(t *testing.T) {
			t.Parallel()
			svc, observer := newPostgresTestService(t)
			var before entity.Contract
			if operation == "sign" {
				created := createTestContract(t, svc, uuid.New())
				before = readStoredContract(t, observer, created.ID)
			}
			// This trigger fails at COMMIT, after INSERT/UPDATE succeeded.
			_, err := observer.Exec(t.Context(), `
				CREATE FUNCTION reject_commit() RETURNS trigger LANGUAGE plpgsql AS $$
				BEGIN RAISE EXCEPTION 'test commit failure' USING ERRCODE = '23514'; END;
				$$;
				CREATE CONSTRAINT TRIGGER reject_commit
				AFTER INSERT OR UPDATE ON contracts DEFERRABLE INITIALLY DEFERRED
				FOR EACH ROW EXECUTE FUNCTION reject_commit();
			`)
			if err != nil {
				t.Fatal(err)
			}
			if operation == "create" {
				var response CreateContractResponse
				response, err = svc.CreateContract(t.Context(), CreateContractRequest{ClientID: uuid.New()})
				if !reflect.DeepEqual(response, CreateContractResponse{}) {
					t.Errorf("failed commit returned response: %+v", response)
				}
			} else {
				var response SignContractResponse
				response, err = svc.SignContract(t.Context(), SignContractRequest{ContractID: before.ID})
				if response != (SignContractResponse{}) {
					t.Errorf("failed commit returned response: %+v", response)
				}
			}
			var pgErr *pgconn.PgError
			if !errors.As(err, &pgErr) || pgErr.Code != "23514" || !strings.Contains(err.Error(), "commit "+operation+" contract") {
				t.Fatalf("error = %v, want wrapped commit failure", err)
			}
			var count int
			if err := observer.QueryRow(t.Context(), "SELECT count(*) FROM contracts").Scan(&count); err != nil {
				t.Fatal(err)
			}
			if operation == "create" && count != 0 {
				t.Errorf("failed creation left %d contracts", count)
			}
			if operation == "sign" && (count != 1 || !reflect.DeepEqual(readStoredContract(t, observer, before.ID), before)) {
				t.Error("failed signing commit changed stored data")
			}
			assertPoolAvailable(t, svc.pool)
		})
	}
}

func TestServicePostgresCancelledContext(t *testing.T) {
	t.Parallel()
	svc, _ := newPostgresTestService(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	created, err := svc.CreateContract(ctx, CreateContractRequest{ClientID: uuid.New()})
	if !errors.Is(err, context.Canceled) || !reflect.DeepEqual(created, CreateContractResponse{}) {
		t.Fatalf("cancelled creation = %+v, %v", created, err)
	}
	signed, err := svc.SignContract(ctx, SignContractRequest{ContractID: uuid.New()})
	if !errors.Is(err, context.Canceled) || signed != (SignContractResponse{}) {
		t.Fatalf("cancelled signing = %+v, %v", signed, err)
	}
	assertPoolAvailable(t, svc.pool)
}

func newPostgresTestService(t *testing.T) (*Service, *pgx.Conn) {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal("invalid TEST_DATABASE_URL")
	}
	schema := "service_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	config.MaxConns = 1
	config.MinConns = 0
	config.ConnConfig.RuntimeParams["search_path"] = schema
	config.ConnConfig.RuntimeParams["statement_timeout"] = "5s"
	observer, err := pgx.ConnectConfig(t.Context(), config.ConnConfig.Copy())
	if err != nil {
		t.Fatalf("connect test PostgreSQL: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, err := observer.Exec(ctx, "DROP SCHEMA "+pgx.Identifier{schema}.Sanitize()+" CASCADE"); err != nil {
			t.Errorf("drop test schema: %v", err)
		}
		_ = observer.Close(ctx)
	})
	if _, err := observer.Exec(t.Context(), "CREATE SCHEMA "+pgx.Identifier{schema}.Sanitize()); err != nil {
		t.Fatal(err)
	}
	migration, err := os.ReadFile("../../migrations/00001_create_contracts.sql")
	if err != nil {
		t.Fatal(err)
	}
	up, _, found := strings.Cut(string(migration), "-- +goose Down")
	if !found {
		t.Fatal("migration is missing the Down boundary")
	}
	if _, err := observer.Exec(t.Context(), up); err != nil {
		t.Fatalf("apply test migration: %v", err)
	}
	pool, err := pgxpool.NewWithConfig(t.Context(), config)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return NewService(pool), observer
}

func createTestContract(t *testing.T, svc *Service, clientID uuid.UUID) CreateContractResponse {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	created, err := svc.CreateContract(ctx, CreateContractRequest{ClientID: clientID})
	if err != nil {
		t.Fatalf("create test contract: %v", err)
	}
	return created
}

func readStoredContract(t *testing.T, conn *pgx.Conn, id uuid.UUID) entity.Contract {
	t.Helper()
	var row entity.Contract
	err := conn.QueryRow(t.Context(), `
		SELECT id, client_id, status, created_at, updated_at, expired_at
		FROM contracts WHERE id = $1`, id).Scan(
		&row.ID, &row.ClientID, &row.Status, &row.CreatedAt, &row.UpdatedAt, &row.ExpiredAt,
	)
	if err != nil {
		t.Fatalf("read stored contract: %v", err)
	}
	return row
}

func assertPoolAvailable(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	if got := pool.Stat().AcquiredConns(); got != 0 {
		t.Errorf("service still holds %d connections", got)
	}
	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()
	if err := pool.Ping(ctx); err != nil {
		t.Errorf("pool is unusable after operation: %v", err)
	}
}
