package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/anton415/sharetrip-contract/internal/domain"
	"github.com/anton415/sharetrip-contract/internal/repository"
	"github.com/anton415/sharetrip-contract/internal/service"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type stubContractService struct {
	create    func(context.Context, service.CreateContractRequest) (service.CreateContractResponse, error)
	sign      func(context.Context, service.SignContractRequest) (service.SignContractResponse, error)
	get       func(context.Context, service.GetContractRequest) (service.GetContractResponse, error)
	getActive func(context.Context, service.GetActiveContractRequest) (service.GetContractResponse, error)
}

func (s stubContractService) UpsertContractServices(context.Context, service.UpsertContractServicesRequest) error {
	return errors.New("unexpected UpsertContractServices call")
}

func (s stubContractService) CheckService(context.Context, service.CheckServiceRequest) (service.CheckServiceResponse, error) {
	return service.CheckServiceResponse{}, errors.New("unexpected CheckService call")
}

func (s stubContractService) CreateContract(ctx context.Context, request service.CreateContractRequest) (service.CreateContractResponse, error) {
	if s.create == nil {
		return service.CreateContractResponse{}, errors.New("unexpected CreateContract call")
	}
	return s.create(ctx, request)
}

func (s stubContractService) SignContract(ctx context.Context, request service.SignContractRequest) (service.SignContractResponse, error) {
	if s.sign == nil {
		return service.SignContractResponse{}, errors.New("unexpected SignContract call")
	}
	return s.sign(ctx, request)
}

func (s stubContractService) GetContract(ctx context.Context, request service.GetContractRequest) (service.GetContractResponse, error) {
	if s.get == nil {
		return service.GetContractResponse{}, errors.New("unexpected GetContract call")
	}
	return s.get(ctx, request)
}

func (s stubContractService) GetActiveContract(ctx context.Context, request service.GetActiveContractRequest) (service.GetContractResponse, error) {
	if s.getActive == nil {
		return service.GetContractResponse{}, errors.New("unexpected GetActiveContract call")
	}
	return s.getActive(ctx, request)
}

func TestGetContractHTTPMapping(t *testing.T) {
	t.Parallel()
	for _, status := range []string{"draft", "active", "suspended", "terminated"} {
		t.Run(status, func(t *testing.T) {
			t.Parallel()
			id, clientID := uuid.New(), uuid.New()
			createdAt := time.Date(2026, 9, 7, 10, 0, 0, 0, time.UTC)
			updatedAt, expires := createdAt.Add(time.Hour), createdAt.AddDate(1, 0, 0)
			contract := service.GetContractResponse{
				ID: id, ClientID: clientID, Status: status, CreatedAt: createdAt, UpdatedAt: updatedAt,
			}
			if status == "active" {
				contract.ExpiredAt = &expires
			}
			getCalls, activeCalls := 0, 0
			stub := stubContractService{
				get: func(_ context.Context, request service.GetContractRequest) (service.GetContractResponse, error) {
					getCalls++
					if request.ContractID != id {
						t.Errorf("ContractID = %v, want %v", request.ContractID, id)
					}
					return contract, nil
				},
				getActive: func(_ context.Context, request service.GetActiveContractRequest) (service.GetContractResponse, error) {
					activeCalls++
					if request.ClientID != clientID {
						t.Errorf("ClientID = %v, want %v", request.ClientID, clientID)
					}
					return contract, nil
				},
			}
			want := map[string]any{"contract": map[string]any{
				"id": id, "client_id": clientID, "status": status, "created_at": createdAt,
				"updated_at": updatedAt, "expired_at": contract.ExpiredAt,
			}}
			app := testApp(stub)
			assertJSONResponse(t, getHTTP(t, app, "/contracts/"+id.String()), 200, want)
			wantActiveCalls := 0
			if status == "active" {
				assertJSONResponse(t, getHTTP(t, app, "/clients/"+clientID.String()+"/contracts/active"), 200, want)
				wantActiveCalls = 1
			}
			if getCalls != 1 || activeCalls != wantActiveCalls {
				t.Errorf("service calls = (%d, %d), want (1, %d)", getCalls, activeCalls, wantActiveCalls)
			}
		})
	}
}

func TestGetContractHTTPErrors(t *testing.T) {
	t.Parallel()
	for _, operation := range []struct{ path, field string }{
		{"/contracts/%s", "contract"}, {"/clients/%s/contracts/active", "client"},
	} {
		t.Run(operation.field, func(t *testing.T) {
			t.Parallel()
			for _, tt := range []struct {
				name, id, message string
				err               error
				status, calls     int
			}{
				{"malformed", "invalid", "invalid " + operation.field + " id", nil, 400, 0},
				{"zero", uuid.Nil.String(), "invalid " + operation.field + " id", nil, 400, 0},
				{"missing", uuid.NewString(), "contract not found", repository.ErrContractNotFound, 404, 1},
				{"internal", uuid.NewString(), "internal server error", errors.New("private database detail"), 500, 1},
			} {
				t.Run(tt.name, func(t *testing.T) {
					t.Parallel()
					calls := 0
					result := func() (service.GetContractResponse, error) {
						calls++
						return service.GetContractResponse{}, fmt.Errorf("operation failed: %w", tt.err)
					}
					stub := stubContractService{
						get: func(context.Context, service.GetContractRequest) (service.GetContractResponse, error) {
							return result()
						},
						getActive: func(context.Context, service.GetActiveContractRequest) (service.GetContractResponse, error) {
							return result()
						},
					}
					response := getHTTP(t, testApp(stub), fmt.Sprintf(operation.path, tt.id))
					assertJSONResponse(t, response, tt.status, map[string]string{"error": tt.message})
					if calls != tt.calls {
						t.Errorf("service calls = %d, want %d", calls, tt.calls)
					}
				})
			}
		})
	}
}

func TestCreateContractHTTPMapping(t *testing.T) {
	t.Parallel()
	for _, expiry := range []string{"null", `"2027-01-01T10:00:00Z"`} {
		t.Run(expiry, func(t *testing.T) {
			t.Parallel()
			id, clientID := uuid.New(), uuid.New()
			createdAt := time.Date(2026, 9, 7, 10, 0, 0, 0, time.UTC)
			var expires *time.Time
			if err := json.Unmarshal([]byte(expiry), &expires); err != nil {
				t.Fatal(err)
			}
			calls := 0
			stub := stubContractService{create: func(_ context.Context, request service.CreateContractRequest) (service.CreateContractResponse, error) {
				calls++
				if request.ClientID != clientID || !reflect.DeepEqual(request.ExpiredAt, expires) {
					t.Errorf("unexpected service request: %+v", request)
				}
				return service.CreateContractResponse{
					ID: id, ClientID: clientID, Status: "draft", CreatedAt: createdAt,
					UpdatedAt: createdAt, ExpiredAt: expires,
				}, nil
			}}
			response := postJSON(t, testApp(stub), "/create_contract",
				fmt.Sprintf(`{"client_id":%q,"expired_at":%s}`, clientID, expiry))
			assertJSONResponse(t, response, 201, map[string]any{"contract": map[string]any{
				"id": id, "client_id": clientID, "status": "draft", "created_at": createdAt,
				"updated_at": createdAt, "expired_at": expires,
			}})
			if calls != 1 {
				t.Errorf("service calls = %d, want 1", calls)
			}
		})
	}
}

func TestSignContractHTTPMapping(t *testing.T) {
	t.Parallel()
	requestID, responseID := uuid.New(), uuid.New()
	calls := 0
	stub := stubContractService{sign: func(_ context.Context, request service.SignContractRequest) (service.SignContractResponse, error) {
		calls++
		if request.ContractID != requestID {
			t.Errorf("ContractID = %v, want %v", request.ContractID, requestID)
		}
		return service.SignContractResponse{ID: responseID, Status: "active", UpdatedAt: time.Now()}, nil
	}}
	response := postJSON(t, testApp(stub), "/sign_contract", fmt.Sprintf(`{"contract_id":%q}`, requestID))
	assertJSONResponse(t, response, 200, map[string]any{"contract": map[string]any{
		"id": responseID, "status": "active",
	}})
	if calls != 1 {
		t.Errorf("service calls = %d, want 1", calls)
	}
}

func TestHTTPRejectsInvalidRequests(t *testing.T) {
	t.Parallel()
	for _, operation := range []struct{ path, field string }{
		{"/create_contract", "client_id"}, {"/sign_contract", "contract_id"},
	} {
		t.Run(operation.path, func(t *testing.T) {
			t.Parallel()
			valid := fmt.Sprintf(`{%q:%q}`, operation.field, uuid.New())
			bodies := []string{"", "{", "null", "[]", "{}",
				fmt.Sprintf(`{%q:null}`, operation.field),
				fmt.Sprintf(`{%q:"invalid"}`, operation.field),
				fmt.Sprintf(`{%q:%q}`, operation.field, uuid.Nil),
				strings.TrimSuffix(valid, "}") + `,"unknown":true}`,
				valid + " {}",
			}
			if operation.path == "/create_contract" {
				bodies = append(bodies, strings.TrimSuffix(valid, "}")+`,"expired_at":"invalid"}`)
			}
			for _, body := range bodies {
				t.Run(body, func(t *testing.T) {
					t.Parallel()
					response := postJSON(t, testApp(stubContractService{}), operation.path, body)
					assertJSONResponse(t, response, 400, map[string]string{"error": "invalid request body"})
				})
			}
			request := httptest.NewRequest(http.MethodPost, operation.path, strings.NewReader(valid))
			request.Header.Set("Content-Type", "text/plain")
			response, err := testApp(stubContractService{}).Test(request)
			if err != nil {
				t.Fatal(err)
			}
			assertJSONResponse(t, response, 400, map[string]string{"error": "invalid request body"})
		})
	}
}

func TestHTTPServiceErrors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name, path string
		err        error
		status     int
		message    string
	}{
		{"client id", "/create_contract", domain.ErrInvalidClientID, 400, "invalid client id"},
		{"contract id", "/sign_contract", domain.ErrInvalidContractID, 400, "invalid contract id"},
		{"missing", "/sign_contract", repository.ErrContractNotFound, 404, "contract not found"},
		{"transition", "/sign_contract", domain.ErrInvalidContractTransition, 409, "contract cannot be signed in its current state"},
		{"create internal", "/create_contract", errors.New("private database detail"), 500, "internal server error"},
		{"sign internal", "/sign_contract", errors.New("private database detail"), 500, "internal server error"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			wrapped := fmt.Errorf("operation failed: %w", tt.err)
			stub := stubContractService{
				create: func(context.Context, service.CreateContractRequest) (service.CreateContractResponse, error) {
					return service.CreateContractResponse{}, wrapped
				},
				sign: func(context.Context, service.SignContractRequest) (service.SignContractResponse, error) {
					return service.SignContractResponse{}, wrapped
				},
			}
			field := "client_id"
			if tt.path == "/sign_contract" {
				field = "contract_id"
			}
			response := postJSON(t, testApp(stub), tt.path, fmt.Sprintf(`{%q:%q}`, field, uuid.New()))
			assertJSONResponse(t, response, tt.status, map[string]string{"error": tt.message})
		})
	}
}

func testApp(contractService ContractService) *fiber.App {
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	RegisterRoutes(app, contractService)
	return app
}

func postJSON(t *testing.T, app *fiber.App, path, body string) *http.Response {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response, err := app.Test(request, 10000)
	if err != nil {
		t.Fatal(err)
	}
	return response
}

func getHTTP(t *testing.T, app *fiber.App, path string) *http.Response {
	t.Helper()
	response, err := app.Test(httptest.NewRequest(http.MethodGet, path, nil), 10000)
	if err != nil {
		t.Fatal(err)
	}
	return response
}

func assertJSONResponse(t *testing.T, response *http.Response, status int, want any) {
	t.Helper()
	defer closeResponseBody(t, response)
	if response.StatusCode != status {
		t.Errorf("HTTP status = %d, want %d", response.StatusCode, status)
	}
	if !strings.HasPrefix(response.Header.Get("Content-Type"), "application/json") {
		t.Error("response must have JSON content type")
	}
	var got, expected any
	if err := json.NewDecoder(response.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(encoded, &expected); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("JSON = %#v, want %#v", got, expected)
	}
}

func closeResponseBody(t *testing.T, response *http.Response) {
	t.Helper()
	if err := response.Body.Close(); err != nil {
		t.Errorf("close response body: %v", err)
	}
}
