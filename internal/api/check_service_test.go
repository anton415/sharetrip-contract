package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/anton415/sharetrip-contract/gen"
	"github.com/anton415/sharetrip-contract/internal/domain"
	"github.com/anton415/sharetrip-contract/internal/service"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type checkServiceLogStub struct {
	stubContractService
	allowed bool
	reason  string
	err     error
	check   func(context.Context, service.CheckServiceRequest)
}

func (s checkServiceLogStub) CheckService(ctx context.Context, request service.CheckServiceRequest) (service.CheckServiceResponse, error) {
	if s.check != nil {
		s.check(ctx, request)
	}
	return service.CheckServiceResponse{Allowed: s.allowed, Reason: s.reason}, s.err
}

func TestCheckServiceLogging(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		result  string
		allowed bool
		reason  string
		err     error
		status  int
	}{
		{result: "allowed", allowed: true, reason: "service_allowed", status: http.StatusOK},
		{result: "denied", reason: "service_not_allowed", status: http.StatusOK},
		{result: "error", err: errors.New("database unavailable"), status: http.StatusInternalServerError},
	} {
		t.Run(tt.result, func(t *testing.T) {
			t.Parallel()
			var logs bytes.Buffer
			clientID, tripID := uuid.NewString(), uuid.NewString()
			server := NewServer(checkServiceLogStub{
				allowed: tt.allowed,
				reason:  tt.reason,
				err:     tt.err,
				check: func(ctx context.Context, request service.CheckServiceRequest) {
					if request.ClientID.String() != clientID || request.ServiceCode != "trip_creation" {
						t.Errorf("unexpected service request: %+v", request)
					}
					requestLogger(ctx).InfoContext(ctx, "service check")
				},
			})
			server.logger = slog.New(slog.NewJSONHandler(&logs, nil))
			app := fiber.New()
			gen.RegisterHandlersWithOptions(app, server, gen.FiberServerOptions{
				HandlerMiddlewares: []gen.HandlerMiddlewareFunc{server.checkServiceMiddleware},
			})
			body := fmt.Sprintf(`{"client_id":%q,"service_code":"trip_creation"}`, clientID)
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
			messages := []string{"service check"}
			if tt.err != nil {
				messages = append(messages, "contract request failed")
				var body gen.ErrorResponse
				if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
					t.Fatal(err)
				}
				if body.Error != "internal server error" {
					t.Errorf("response error = %q, want internal server error", body.Error)
				}
			} else {
				var body map[string]any
				if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
					t.Fatal(err)
				}
				if len(body) != 2 || body["allowed"] != tt.allowed || body["reason"] != tt.reason {
					t.Errorf("response = %v, want allowed=%t and reason=%q", body, tt.allowed, tt.reason)
				}
			}
			decoder := json.NewDecoder(&logs)
			for _, message := range messages {
				var entry map[string]any
				if err := decoder.Decode(&entry); err != nil {
					t.Fatal(err)
				}
				for key, want := range map[string]string{
					"service": "contract", "operation": "CheckPermission",
					"request_id": "req-123", "correlation_id": "pub-777",
					"trip_id": tripID, "client_id": clientID, "msg": message,
				} {
					if entry[key] != want {
						t.Errorf("log %s = %v, want %s", key, entry[key], want)
					}
				}
				if _, exists := entry["user_id"]; exists {
					t.Error("log must use client_id instead of user_id")
				}
				if message == "contract request failed" {
					if entry["result"] != tt.result {
						t.Errorf("log result = %v, want %s", entry["result"], tt.result)
					}
					if entry["level"] != "ERROR" {
						t.Errorf("log level = %v, want ERROR", entry["level"])
					}
					if want := "check service: " + tt.err.Error(); entry["error"] != want {
						t.Errorf("log error = %v, want %s", entry["error"], want)
					}
				}
			}
			if err := decoder.Decode(new(map[string]any)); err != io.EOF {
				t.Errorf("unexpected extra log entry: %v", err)
			}
		})
	}
}

func TestCheckServiceMapsServiceErrors(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		err     error
		message string
	}{
		{err: domain.ErrInvalidClientID, message: "invalid client id"},
		{err: domain.ErrInvalidContractServices, message: "invalid contract services"},
	} {
		t.Run(tt.message, func(t *testing.T) {
			t.Parallel()
			body := fmt.Sprintf(`{"client_id":%q,"service_code":"trip_creation"}`, uuid.NewString())
			response := postJSON(t, testApp(checkServiceLogStub{err: tt.err}), "/contracts/check-service", body)
			assertJSONResponse(t, response, http.StatusBadRequest, map[string]string{"error": tt.message})
		})
	}
}

func TestCheckServiceRejectsInvalidBody(t *testing.T) {
	t.Parallel()
	valid := fmt.Sprintf(`{"client_id":%q,"service_code":"trip_creation"}`, uuid.NewString())
	for _, tt := range []struct {
		name, body, contentType string
	}{
		{name: "empty", contentType: "application/json"},
		{name: "malformed", body: "{", contentType: "application/json"},
		{name: "array", body: "[]", contentType: "application/json"},
		{name: "invalid UUID", body: `{"client_id":"invalid","service_code":"trip_creation"}`, contentType: "application/json"},
		{name: "unknown field", body: strings.TrimSuffix(valid, "}") + `,"unknown":true}`, contentType: "application/json"},
		{name: "multiple values", body: valid + " {}", contentType: "application/json"},
		{name: "wrong content type", body: valid, contentType: "text/plain"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			app := testApp(checkServiceLogStub{check: func(context.Context, service.CheckServiceRequest) {
				t.Error("invalid body must not reach the service")
			}})
			request := httptest.NewRequest(http.MethodPost, "/contracts/check-service", strings.NewReader(tt.body))
			request.Header.Set("Content-Type", tt.contentType)
			response, err := app.Test(request)
			if err != nil {
				t.Fatal(err)
			}
			assertJSONResponse(t, response, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		})
	}
}
