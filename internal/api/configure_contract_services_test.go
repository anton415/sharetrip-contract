package api

import (
	"reflect"
	"testing"

	"github.com/anton415/sharetrip-contract/internal/service"
	"github.com/google/uuid"
)

func TestHTTPPostgresConfigureContractServices(t *testing.T) {
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
		response := postJSON(t, app, "/contracts/"+step.id+"/configure-services", step.body)
		if step.status == 204 {
			closeResponseBody(t, response)
			if response.StatusCode != step.status {
				t.Fatalf("configure services: status=%d, want %d", response.StatusCode, step.status)
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
	want := map[string]bool{"trip_creation": false, "trip_participants": true}
	if !reflect.DeepEqual(stored, want) {
		t.Fatalf("stored services = %v, want %v", stored, want)
	}
	if _, err := svc.SignContract(t.Context(), service.SignContractRequest{ContractID: created.ID}); err != nil {
		t.Fatal(err)
	}
	assertJSONResponse(t, postJSON(t, app, "/contracts/"+id+"/configure-services",
		`{"services":[{"service_code":"trip_creation","enabled":true}]}`), 409,
		map[string]string{"error": "contract services can be configured only for a draft contract"})
}
