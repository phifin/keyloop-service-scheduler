package handlers_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apphttp "keyloop-service-scheduler/internal/http"
	"keyloop-service-scheduler/internal/repository"
	"keyloop-service-scheduler/internal/scheduling"
)

const testUUID = "11111111-1111-1111-1111-111111111111"

type fakeReferenceRepository struct {
	err error
}

func (r fakeReferenceRepository) ListDealerships(ctx context.Context) ([]repository.Dealership, error) {
	if r.err != nil {
		return nil, r.err
	}

	return []repository.Dealership{{
		ID:       "44444444-4444-4444-4444-444444444444",
		Name:     "Downtown Keyloop Motors",
		Address:  "100 Main Street",
		Timezone: "America/Detroit",
	}}, nil
}

func (r fakeReferenceRepository) ListCustomers(ctx context.Context) ([]repository.Customer, error) {
	if r.err != nil {
		return nil, r.err
	}

	phone := "+1-555-0101"
	return []repository.Customer{{
		ID:    testUUID,
		Name:  "John Smith",
		Email: "john.smith@example.com",
		Phone: &phone,
	}}, nil
}

func (r fakeReferenceRepository) ListCustomerVehicles(ctx context.Context, customerID string) ([]repository.Vehicle, error) {
	if r.err != nil {
		return nil, r.err
	}

	return []repository.Vehicle{{
		ID:         "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa1",
		CustomerID: customerID,
		VIN:        "KLSSVIN000000001",
		Make:       "Toyota",
		Model:      "Corolla",
		Year:       2021,
	}}, nil
}

func (r fakeReferenceRepository) ListServiceTypes(ctx context.Context) ([]repository.ServiceType, error) {
	if r.err != nil {
		return nil, r.err
	}

	return []repository.ServiceType{{
		ID:                "55555555-5555-5555-5555-555555555551",
		Name:              "Oil Change",
		DurationMinutes:   30,
		RequiredSkillCode: "OIL_CHANGE",
	}}, nil
}

func (r fakeReferenceRepository) ListTechnicians(ctx context.Context, dealershipID string) ([]repository.Technician, error) {
	if r.err != nil {
		return nil, r.err
	}

	return []repository.Technician{{
		ID:           "66666666-6666-6666-6666-666666666661",
		DealershipID: dealershipID,
		Name:         "Alex Morgan",
		Skills:       []string{"GENERAL_MAINTENANCE", "OIL_CHANGE"},
	}}, nil
}

func (r fakeReferenceRepository) ListServiceBays(ctx context.Context, dealershipID string) ([]repository.ServiceBay, error) {
	if r.err != nil {
		return nil, r.err
	}

	return []repository.ServiceBay{{
		ID:           "88888888-8888-8888-8888-888888888881",
		DealershipID: dealershipID,
		Name:         "Bay 1",
	}}, nil
}

func TestReferenceEndpointsSuccess(t *testing.T) {
	router := testRouter(fakeReferenceRepository{}, nil)

	tests := []struct {
		name           string
		path           string
		expectedStatus int
		expectedBody   string
	}{
		{name: "dealerships", path: "/dealerships", expectedStatus: http.StatusOK, expectedBody: "Downtown Keyloop Motors"},
		{name: "customers", path: "/customers", expectedStatus: http.StatusOK, expectedBody: "John Smith"},
		{name: "vehicles", path: "/customers/" + testUUID + "/vehicles", expectedStatus: http.StatusOK, expectedBody: "KLSSVIN000000001"},
		{name: "service types", path: "/service-types", expectedStatus: http.StatusOK, expectedBody: "OIL_CHANGE"},
		{name: "technicians", path: "/technicians?dealershipId=44444444-4444-4444-4444-444444444444", expectedStatus: http.StatusOK, expectedBody: "Alex Morgan"},
		{name: "service bays", path: "/service-bays?dealershipId=44444444-4444-4444-4444-444444444444", expectedStatus: http.StatusOK, expectedBody: "Bay 1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, tt.path, nil)

			router.ServeHTTP(response, request)

			if response.Code != tt.expectedStatus {
				t.Fatalf("expected status %d, got %d", tt.expectedStatus, response.Code)
			}
			if !strings.Contains(response.Body.String(), tt.expectedBody) {
				t.Fatalf("expected body to contain %q, got %q", tt.expectedBody, response.Body.String())
			}
		})
	}
}

func TestReferenceEndpointValidation(t *testing.T) {
	router := testRouter(fakeReferenceRepository{}, nil)

	tests := []struct {
		name           string
		path           string
		expectedStatus int
		expectedBody   string
	}{
		{name: "invalid customer id", path: "/customers/not-a-uuid/vehicles", expectedStatus: http.StatusBadRequest, expectedBody: "customerId must be a valid UUID"},
		{name: "missing technicians dealership id", path: "/technicians", expectedStatus: http.StatusBadRequest, expectedBody: "dealershipId is required"},
		{name: "invalid technicians dealership id", path: "/technicians?dealershipId=not-a-uuid", expectedStatus: http.StatusBadRequest, expectedBody: "dealershipId must be a valid UUID"},
		{name: "missing service bays dealership id", path: "/service-bays", expectedStatus: http.StatusBadRequest, expectedBody: "dealershipId is required"},
		{name: "invalid service bays dealership id", path: "/service-bays?dealershipId=not-a-uuid", expectedStatus: http.StatusBadRequest, expectedBody: "dealershipId must be a valid UUID"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, tt.path, nil)

			router.ServeHTTP(response, request)

			if response.Code != tt.expectedStatus {
				t.Fatalf("expected status %d, got %d", tt.expectedStatus, response.Code)
			}
			if !strings.Contains(response.Body.String(), tt.expectedBody) {
				t.Fatalf("expected body to contain %q, got %q", tt.expectedBody, response.Body.String())
			}
		})
	}
}

func TestCustomerVehiclesUnknownCustomer(t *testing.T) {
	router := testRouter(fakeReferenceRepository{err: repository.ErrNotFound}, nil)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/customers/"+testUUID+"/vehicles", nil)

	router.ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, response.Code)
	}
	if !strings.Contains(response.Body.String(), "customer not found") {
		t.Fatalf("expected not found message, got %q", response.Body.String())
	}
}

func TestRepositoryErrorIsGeneric(t *testing.T) {
	router := testRouter(fakeReferenceRepository{err: errors.New("database exploded")}, nil)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/dealerships", nil)

	router.ServeHTTP(response, request)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, response.Code)
	}
	if strings.Contains(response.Body.String(), "database exploded") {
		t.Fatalf("expected internal database error to be hidden, got %q", response.Body.String())
	}
}

func TestReferenceEndpointWithoutRepository(t *testing.T) {
	router := testRouter(nil, nil)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/dealerships", nil)

	router.ServeHTTP(response, request)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, response.Code)
	}
}

func testRouter(referenceRepo repository.ReferenceRepository, availabilityService *scheduling.AvailabilityService) http.Handler {
	return apphttp.NewRouter(referenceRepo, availabilityService, nil, testLogger())
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
