package handlers_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"keyloop-service-scheduler/internal/scheduling"
)

func TestAvailabilityValidation(t *testing.T) {
	service := scheduling.NewAvailabilityService(fakeAvailabilityRepository{})
	router := testRouter(nil, service)

	tests := []struct {
		name           string
		path           string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "missing dealershipId",
			path:           "/availability?serviceTypeId=55555555-5555-5555-5555-555555555551&startTime=2026-05-04T10:00:00Z",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "dealershipId is required",
		},
		{
			name:           "invalid serviceTypeId",
			path:           "/availability?dealershipId=44444444-4444-4444-4444-444444444444&serviceTypeId=not-a-uuid&startTime=2026-05-04T10:00:00Z",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "serviceTypeId must be a valid UUID",
		},
		{
			name:           "invalid startTime",
			path:           "/availability?dealershipId=44444444-4444-4444-4444-444444444444&serviceTypeId=55555555-5555-5555-5555-555555555551&startTime=2026-05-04%2010:00:00",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "startTime must be a valid RFC3339 timestamp",
		},
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

func TestAvailabilityNotFound(t *testing.T) {
	tests := []struct {
		name string
		repo fakeAvailabilityRepository
	}{
		{name: "unknown dealership", repo: fakeAvailabilityRepository{dealershipErr: scheduling.ErrNotFound}},
		{name: "unknown service type", repo: fakeAvailabilityRepository{serviceTypeErr: scheduling.ErrNotFound}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := scheduling.NewAvailabilityService(tt.repo)
			router := testRouter(nil, service)
			response := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/availability?dealershipId=44444444-4444-4444-4444-444444444444&serviceTypeId=55555555-5555-5555-5555-555555555551&startTime=2026-05-04T10:00:00Z", nil)

			router.ServeHTTP(response, request)

			if response.Code != http.StatusNotFound {
				t.Fatalf("expected status %d, got %d", http.StatusNotFound, response.Code)
			}
			if !strings.Contains(response.Body.String(), "dealership or service type not found") {
				t.Fatalf("expected not found message, got %q", response.Body.String())
			}
		})
	}
}

type fakeAvailabilityRepository struct {
	dealershipErr     error
	serviceTypeErr    error
	busyTechnicianIDs []string
	busyServiceBayIDs []string
}

func (r fakeAvailabilityRepository) VerifyDealershipExists(ctx context.Context, dealershipID string) error {
	return r.dealershipErr
}

func (r fakeAvailabilityRepository) GetServiceType(ctx context.Context, serviceTypeID string) (scheduling.ServiceType, error) {
	if r.serviceTypeErr != nil {
		return scheduling.ServiceType{}, r.serviceTypeErr
	}

	return scheduling.ServiceType{
		ID:                serviceTypeID,
		DurationMinutes:   30,
		RequiredSkillCode: "OIL_CHANGE",
	}, nil
}

func (r fakeAvailabilityRepository) ListQualifiedTechnicians(ctx context.Context, dealershipID, requiredSkillCode string) ([]scheduling.Resource, error) {
	return []scheduling.Resource{{ID: "tech-1", Name: "Alex Morgan"}}, nil
}

func (r fakeAvailabilityRepository) ListBusyTechnicianIDs(ctx context.Context, dealershipID string, startTime, endTime time.Time) ([]string, error) {
	return r.busyTechnicianIDs, nil
}

func (r fakeAvailabilityRepository) ListAvailableServiceBays(ctx context.Context, dealershipID string) ([]scheduling.Resource, error) {
	return []scheduling.Resource{{ID: "bay-1", Name: "Bay 1"}}, nil
}

func (r fakeAvailabilityRepository) ListBusyServiceBayIDs(ctx context.Context, dealershipID string, startTime, endTime time.Time) ([]string, error) {
	return r.busyServiceBayIDs, nil
}
