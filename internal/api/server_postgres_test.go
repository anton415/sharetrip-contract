package api

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/anton415/sharetrip-contract/gen"
	"github.com/anton415/sharetrip-contract/internal/service"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestHTTPPostgresContractLifecycle(t *testing.T) {
	pool, observer := newHTTPTestDatabase(t)
	app := testApp(service.NewService(pool))
	clientID := uuid.New()
	expires := time.Date(2027, 1, 1, 10, 0, 0, 0, time.UTC)
	response := postJSON(t, app, "/create_contract",
		fmt.Sprintf(`{"client_id":%q,"expired_at":%q}`, clientID, expires.Format(time.RFC3339)))
	var created gen.CreateContractResponse
	err := json.NewDecoder(response.Body).Decode(&created)
	response.Body.Close()
	if err != nil || response.StatusCode != 201 {
		t.Fatalf("create response: status=%d, error=%v", response.StatusCode, err)
	}
	contract := created.Contract
	if contract.Id == uuid.Nil || contract.ClientId != clientID || contract.Status != gen.Draft ||
		contract.CreatedAt.IsZero() || !contract.UpdatedAt.Equal(contract.CreatedAt) ||
		contract.ExpiredAt == nil || !contract.ExpiredAt.Equal(expires) {
		t.Fatalf("unexpected contract: %+v", contract)
	}
	var storedClient uuid.UUID
	var storedStatus string
	var storedCreated, storedUpdated, storedExpires time.Time
	err = observer.QueryRow(t.Context(), `
		SELECT client_id, status, created_at, updated_at, expired_at FROM contracts WHERE id = $1`, contract.Id).
		Scan(&storedClient, &storedStatus, &storedCreated, &storedUpdated, &storedExpires)
	if err != nil {
		t.Fatalf("read committed creation: %v", err)
	}
	if storedClient != clientID || storedStatus != "draft" ||
		!storedCreated.Equal(contract.CreatedAt.Truncate(time.Microsecond)) ||
		!storedUpdated.Equal(contract.UpdatedAt.Truncate(time.Microsecond)) || !storedExpires.Equal(expires) {
		t.Fatal("HTTP response does not match committed creation")
	}

	signBody := fmt.Sprintf(`{"contract_id":%q}`, contract.Id)
	response = postJSON(t, app, "/sign_contract", signBody)
	assertJSONResponse(t, response, 200, map[string]any{"contract": map[string]any{
		"id": contract.Id, "status": "active",
	}})
	if err := observer.QueryRow(t.Context(), "SELECT status, updated_at FROM contracts WHERE id = $1", contract.Id).
		Scan(&storedStatus, &storedUpdated); err != nil {
		t.Fatal(err)
	}
	if storedStatus != "active" || storedUpdated.Before(storedCreated) {
		t.Fatal("HTTP signing was not committed")
	}
	signedAt := storedUpdated
	response = postJSON(t, app, "/sign_contract", signBody)
	assertJSONResponse(t, response, 409, map[string]string{"error": "contract cannot be signed in its current state"})
	if err := observer.QueryRow(t.Context(), "SELECT status, updated_at FROM contracts WHERE id = $1", contract.Id).
		Scan(&storedStatus, &storedUpdated); err != nil {
		t.Fatal(err)
	}
	if storedStatus != "active" || !storedUpdated.Equal(signedAt) {
		t.Error("rejected HTTP signing changed stored data")
	}
	response = postJSON(t, app, "/sign_contract", fmt.Sprintf(`{"contract_id":%q}`, uuid.New()))
	assertJSONResponse(t, response, 404, map[string]string{"error": "contract not found"})
	response = postJSON(t, app, "/create_contract", `{"client_id":"invalid"}`)
	assertJSONResponse(t, response, 400, map[string]string{"error": "invalid request body"})

	// The real service must translate a database access failure into a safe HTTP error.
	pool.Close()
	response = postJSON(t, app, "/create_contract", fmt.Sprintf(`{"client_id":%q}`, clientID))
	assertJSONResponse(t, response, 500, map[string]string{"error": "internal server error"})
	response = postJSON(t, app, "/sign_contract", signBody)
	assertJSONResponse(t, response, 500, map[string]string{"error": "internal server error"})
}

func newHTTPTestDatabase(t *testing.T) (*pgxpool.Pool, *pgx.Conn) {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal("invalid TEST_DATABASE_URL")
	}
	schema := "api_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	config.ConnConfig.RuntimeParams["search_path"] = schema
	config.ConnConfig.RuntimeParams["statement_timeout"] = "5s"
	observer, err := pgx.ConnectConfig(t.Context(), config.ConnConfig.Copy())
	if err != nil {
		t.Fatal(err)
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
		t.Fatal(err)
	}
	pool, err := pgxpool.NewWithConfig(t.Context(), config)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool, observer
}
