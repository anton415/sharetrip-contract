package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/anton415/sharetrip-contract/gen"
	"github.com/anton415/sharetrip-contract/internal/service"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestHTTPPostgresCheckService(t *testing.T) {
	t.Parallel()
	pool, observer := newHTTPTestDatabase(t)
	svc := service.NewService(pool)
	app := testApp(svc)
	clientID := uuid.New()
	check := func(code string, allowed bool, reason string) {
		t.Helper()
		body := fmt.Sprintf(`{"client_id":%q,"service_code":%q}`, clientID, code)
		assertJSONResponse(t, postJSON(t, app, "/contracts/check-service", body), 200,
			map[string]any{"allowed": allowed, "reason": reason})
	}
	setService := func(id uuid.UUID, enabled bool) {
		t.Helper()
		if err := svc.UpsertContractServices(t.Context(), service.UpsertContractServicesRequest{
			ContractID: id,
			Services:   []service.ContractService{{ServiceCode: "trip_creation", Enabled: enabled}},
		}); err != nil {
			t.Fatal(err)
		}
	}
	check("trip_creation", false, "contract_not_found")
	created, err := svc.CreateContract(t.Context(), service.CreateContractRequest{ClientID: clientID})
	if err != nil {
		t.Fatal(err)
	}
	setService(created.ID, true)
	check("trip_creation", false, "contract_not_active")
	if _, err := svc.SignContract(t.Context(), service.SignContractRequest{ContractID: created.ID}); err != nil {
		t.Fatal(err)
	}
	check("trip_creation", true, "service_allowed")
	check("notifications", false, "service_not_allowed")
	otherClientBody := fmt.Sprintf(`{"client_id":%q,"service_code":"trip_creation"}`, uuid.New())
	assertJSONResponse(t, postJSON(t, app, "/contracts/check-service", otherClientBody), 200,
		map[string]any{"allowed": false, "reason": "contract_not_found"})

	setService(created.ID, false)
	// A newer draft with the service enabled must not override the active contract.
	draft, err := svc.CreateContract(t.Context(), service.CreateContractRequest{ClientID: clientID})
	if err != nil {
		t.Fatal(err)
	}
	setService(draft.ID, true)
	check("trip_creation", false, "service_not_allowed")
	setService(created.ID, true)
	if _, err := observer.Exec(t.Context(), "UPDATE contracts SET expired_at = $2 WHERE id = $1",
		created.ID, time.Now().Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	check("trip_creation", false, "contract_expired")
	assertJSONResponse(t, postJSON(t, app, "/contracts/check-service",
		`{"client_id":"invalid","service_code":"trip_creation"}`), 400,
		map[string]string{"error": "invalid request body"})
	assertJSONResponse(t, postJSON(t, app, "/contracts/check-service",
		fmt.Sprintf(`{"client_id":%q}`, clientID)), 400,
		map[string]string{"error": "invalid contract services"})
	pool.Close()
	assertJSONResponse(t, postJSON(t, app, "/contracts/check-service", otherClientBody), 500,
		map[string]string{"error": "internal server error"})
}

func TestHTTPPostgresUpsertContractServices(t *testing.T) {
	t.Parallel()
	pool, observer := newHTTPTestDatabase(t)
	svc := service.NewService(pool)
	created, err := svc.CreateContract(t.Context(), service.CreateContractRequest{ClientID: uuid.New()})
	if err != nil {
		t.Fatal(err)
	}
	app := testApp(svc)
	id := created.ID.String()
	for _, step := range []struct {
		id, body, message string
		status            int
	}{
		{id, `{"services":[{"service_code":"trip_creation","enabled":true},{"service_code":"notifications","enabled":true}]}`, "", 204},
		{id, `{"services":[{"service_code":"trip_creation","enabled":false},{"service_code":"trip_participants","enabled":true}]}`, "", 204},
		{id, `{"services":[{"service_code":"notifications","enabled":false},{"service_code":"unknown","enabled":true}]}`, "invalid contract services", 400},
		{id, `{"services":[{"service_code":"notifications"}]}`, "invalid contract services", 400},
		{id, `{"services":[{"service_code":"notifications","enabled":false},{"service_code":"notifications","enabled":true}]}`, "invalid contract services", 400},
		{"invalid", `{"services":[{"service_code":"trip_creation","enabled":true}]}`, "invalid contract id", 400},
		{uuid.NewString(), `{"services":[{"service_code":"trip_creation","enabled":true}]}`, "contract not found", 404},
	} {
		request := httptest.NewRequest(http.MethodPatch, "/contracts/"+step.id+"/services", strings.NewReader(step.body))
		request.Header.Set("Content-Type", "application/json")
		response, err := app.Test(request, 10000)
		if err != nil {
			t.Fatal(err)
		}
		if step.status == 204 {
			closeResponseBody(t, response)
			if response.StatusCode != step.status {
				t.Fatalf("save services: status=%d, want %d", response.StatusCode, step.status)
			}
		} else {
			assertJSONResponse(t, response, step.status, map[string]string{"error": step.message})
		}
	}
	var stored map[string]bool
	if err := observer.QueryRow(t.Context(), `
		SELECT jsonb_object_agg(service_code, enabled) FROM contract_services WHERE contract_id = $1`, created.ID).
		Scan(&stored); err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{"trip_creation": false, "notifications": true, "trip_participants": true}
	if !reflect.DeepEqual(stored, want) {
		t.Fatalf("stored services = %v, want %v", stored, want)
	}
}

func TestHTTPPostgresContractLifecycle(t *testing.T) {
	t.Parallel()
	pool, observer := newHTTPTestDatabase(t)
	app := testApp(service.NewService(pool))
	clientID := uuid.New()
	expires := time.Date(2027, 1, 1, 10, 0, 0, 0, time.UTC)
	response := postJSON(t, app, "/create_contract",
		fmt.Sprintf(`{"client_id":%q,"expired_at":%q}`, clientID, expires.Format(time.RFC3339)))
	var created gen.CreateContractResponse
	err := json.NewDecoder(response.Body).Decode(&created)
	closeResponseBody(t, response)
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
	getPath := "/contracts/" + contract.Id.String()
	activePath := "/clients/" + clientID.String() + "/contracts/active"
	// Compare timestamps using PostgreSQL's precision and time zone.
	contract.CreatedAt, contract.UpdatedAt = storedCreated, storedUpdated
	contract.ExpiredAt = &storedExpires
	assertJSONResponse(t, getHTTP(t, app, getPath), 200, map[string]any{"contract": contract})
	assertJSONResponse(t, getHTTP(t, app, activePath), 404, map[string]string{"error": "contract not found"})

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
	contract.Status, contract.UpdatedAt = gen.Active, signedAt
	assertJSONResponse(t, getHTTP(t, app, getPath), 200, map[string]any{"contract": contract})
	assertJSONResponse(t, getHTTP(t, app, activePath), 200, map[string]any{"contract": contract})
	response = postJSON(t, app, "/sign_contract", signBody)
	assertJSONResponse(t, response, 409, map[string]string{"error": "contract cannot be signed in its current state"})
	if err := observer.QueryRow(t.Context(), "SELECT status, updated_at FROM contracts WHERE id = $1", contract.Id).
		Scan(&storedStatus, &storedUpdated); err != nil {
		t.Fatal(err)
	}
	if storedStatus != "active" || !storedUpdated.Equal(signedAt) {
		t.Error("HTTP reads or rejected signing changed stored data")
	}
	assertJSONResponse(t, getHTTP(t, app, "/contracts/"+uuid.NewString()), 404, map[string]string{"error": "contract not found"})
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
