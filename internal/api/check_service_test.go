package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/anton415/sharetrip-contract/gen"
	"github.com/anton415/sharetrip-contract/internal/service"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type checkServiceLogStub struct {
	stubContractService
	allowed bool
	err     error
}

func (s checkServiceLogStub) CheckService(context.Context, service.CheckServiceRequest) (service.CheckServiceResponse, error) {
	return service.CheckServiceResponse{Allowed: s.allowed}, s.err
}

func TestCheckServiceLogsPermission(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		result  string
		allowed bool
		err     error
		status  int
	}{
		{result: "allowed", allowed: true, status: http.StatusOK},
		{result: "denied", status: http.StatusOK},
		{result: "error", err: errors.New("database unavailable"), status: http.StatusInternalServerError},
	} {
		t.Run(tt.result, func(t *testing.T) {
			t.Parallel()
			var logs bytes.Buffer
			server := NewServer(checkServiceLogStub{allowed: tt.allowed, err: tt.err})
			server.logger = slog.New(slog.NewJSONHandler(&logs, nil))
			app := fiber.New()
			gen.RegisterHandlers(app, server)
			userID, tripID := uuid.NewString(), uuid.NewString()
			body := fmt.Sprintf(`{"client_id":%q,"service_code":"trip_creation"}`, userID)
			request := httptest.NewRequest(http.MethodPost, "/contracts/check-service", strings.NewReader(body))
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("X-Request-ID", "req-123")
			request.Header.Set("X-Correlation-ID", "pub-777")
			request.Header.Set("X-Trip-ID", tripID)
			response, err := app.Test(request)
			if err != nil {
				t.Fatal(err)
			}
			defer closeResponseBody(t, response)
			if response.StatusCode != tt.status {
				t.Fatalf("status = %d, want %d", response.StatusCode, tt.status)
			}
			var entry map[string]any
			if err := json.NewDecoder(&logs).Decode(&entry); err != nil {
				t.Fatal(err)
			}
			for key, want := range map[string]string{
				"service": "contract", "operation": "CheckPermission",
				"request_id": "req-123", "correlation_id": "pub-777",
				"trip_id": tripID, "user_id": userID, "result": tt.result,
			} {
				if entry[key] != want {
					t.Errorf("log %s = %v, want %s", key, entry[key], want)
				}
			}
		})
	}
}
