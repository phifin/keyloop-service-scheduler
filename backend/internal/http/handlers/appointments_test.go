package handlers_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	apphttp "keyloop-service-scheduler/internal/http"
	"keyloop-service-scheduler/internal/scheduling"
)

type fakeAppointmentService struct {
	createErr error
	getErr    error
	cancelErr error
}

func (s fakeAppointmentService) Create(ctx context.Context, request scheduling.CreateAppointmentRequest) (scheduling.Appointment, error) {
	if s.createErr != nil {
		return scheduling.Appointment{}, s.createErr
	}

	return testAppointment(request), nil
}

func (s fakeAppointmentService) List(ctx context.Context, filters scheduling.AppointmentFilters) ([]scheduling.AppointmentListItem, error) {
	return []scheduling.AppointmentListItem{{
		ID:              "99999999-9999-9999-9999-999999999999",
		CustomerName:    "John Smith",
		VehicleVIN:      "KLSSVIN000000001",
		VehicleLabel:    "Toyota Corolla",
		DealershipName:  "Downtown Keyloop Motors",
		ServiceTypeName: "Oil Change",
		TechnicianName:  "Alex Morgan",
		ServiceBayName:  "Bay 1",
		StartTime:       mustParseTime("2026-05-04T14:00:00Z"),
		EndTime:         mustParseTime("2026-05-04T14:30:00Z"),
		Status:          scheduling.StatusConfirmed,
	}}, nil
}

func (s fakeAppointmentService) GetDetail(ctx context.Context, appointmentID string) (scheduling.AppointmentDetail, error) {
	if s.getErr != nil {
		return scheduling.AppointmentDetail{}, s.getErr
	}

	return scheduling.AppointmentDetail{
		ID:        appointmentID,
		Status:    scheduling.StatusConfirmed,
		StartTime: mustParseTime("2026-05-04T14:00:00Z"),
		EndTime:   mustParseTime("2026-05-04T14:30:00Z"),
		Customer:  scheduling.AppointmentRef{ID: testUUID, Name: "John Smith"},
	}, nil
}

func (s fakeAppointmentService) Cancel(ctx context.Context, appointmentID string) (scheduling.Appointment, error) {
	if s.cancelErr != nil {
		return scheduling.Appointment{}, s.cancelErr
	}

	appointment := testAppointment(scheduling.CreateAppointmentRequest{
		CustomerID:    testUUID,
		VehicleID:     "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa1",
		DealershipID:  "44444444-4444-4444-4444-444444444444",
		ServiceTypeID: "55555555-5555-5555-5555-555555555551",
		StartTime:     mustParseTime("2026-05-04T14:00:00Z"),
	})
	appointment.ID = appointmentID
	appointment.Status = scheduling.StatusCancelled
	return appointment, nil
}

func TestCreateAppointmentValidation(t *testing.T) {
	router := appointmentTestRouter(fakeAppointmentService{})

	tests := []struct {
		name           string
		body           string
		expectedStatus int
		expectedBody   string
	}{
		{name: "invalid uuid", body: `{"customerId":"not-a-uuid","vehicleId":"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa1","dealershipId":"44444444-4444-4444-4444-444444444444","serviceTypeId":"55555555-5555-5555-5555-555555555551","startTime":"2026-05-04T14:00:00Z"}`, expectedStatus: http.StatusBadRequest, expectedBody: "customerId must be a valid UUID"},
		{name: "missing field", body: `{"customerId":"11111111-1111-1111-1111-111111111111"}`, expectedStatus: http.StatusBadRequest, expectedBody: "vehicleId is required"},
		{name: "invalid time", body: `{"customerId":"11111111-1111-1111-1111-111111111111","vehicleId":"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa1","dealershipId":"44444444-4444-4444-4444-444444444444","serviceTypeId":"55555555-5555-5555-5555-555555555551","startTime":"tomorrow"}`, expectedStatus: http.StatusBadRequest, expectedBody: "startTime must be a valid RFC3339 timestamp"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, "/appointments", strings.NewReader(tt.body))

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

func TestCreateAppointmentStatusMapping(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		expectedStatus int
		expectedBody   string
	}{
		{name: "vehicle mismatch", err: scheduling.ErrVehicleCustomerMismatch, expectedStatus: http.StatusBadRequest, expectedBody: "vehicleId must belong to customerId"},
		{name: "unknown resource", err: scheduling.ErrNotFound, expectedStatus: http.StatusNotFound, expectedBody: "customer, vehicle, dealership, or service type not found"},
		{name: "resource unavailable", err: scheduling.ErrConflict, expectedStatus: http.StatusConflict, expectedBody: "RESOURCE_UNAVAILABLE"},
		{name: "internal", err: errors.New("database exploded"), expectedStatus: http.StatusInternalServerError, expectedBody: "failed to create appointment"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := appointmentTestRouter(fakeAppointmentService{createErr: tt.err})
			response := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, "/appointments", strings.NewReader(validCreateAppointmentJSON()))

			router.ServeHTTP(response, request)

			if response.Code != tt.expectedStatus {
				t.Fatalf("expected status %d, got %d", tt.expectedStatus, response.Code)
			}
			if !strings.Contains(response.Body.String(), tt.expectedBody) {
				t.Fatalf("expected body to contain %q, got %q", tt.expectedBody, response.Body.String())
			}
			if strings.Contains(response.Body.String(), "database exploded") {
				t.Fatalf("internal database error leaked: %q", response.Body.String())
			}
		})
	}
}

func TestAppointmentReadAndCancelHandlers(t *testing.T) {
	router := appointmentTestRouter(fakeAppointmentService{})

	tests := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
		expectedBody   string
	}{
		{name: "list", method: http.MethodGet, path: "/appointments", expectedStatus: http.StatusOK, expectedBody: "John Smith"},
		{name: "detail", method: http.MethodGet, path: "/appointments/99999999-9999-9999-9999-999999999999", expectedStatus: http.StatusOK, expectedBody: "CONFIRMED"},
		{name: "cancel", method: http.MethodPatch, path: "/appointments/99999999-9999-9999-9999-999999999999/cancel", expectedStatus: http.StatusOK, expectedBody: "CANCELLED"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			request := httptest.NewRequest(tt.method, tt.path, nil)

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

func TestAppointmentDetailAndCancelErrors(t *testing.T) {
	tests := []struct {
		name           string
		service        fakeAppointmentService
		method         string
		path           string
		expectedStatus int
		expectedBody   string
	}{
		{name: "detail not found", service: fakeAppointmentService{getErr: scheduling.ErrNotFound}, method: http.MethodGet, path: "/appointments/99999999-9999-9999-9999-999999999999", expectedStatus: http.StatusNotFound, expectedBody: "appointment not found"},
		{name: "completed cannot cancel", service: fakeAppointmentService{cancelErr: scheduling.ErrCompletedAppointmentCannotCancel}, method: http.MethodPatch, path: "/appointments/99999999-9999-9999-9999-999999999999/cancel", expectedStatus: http.StatusConflict, expectedBody: "completed appointments cannot be cancelled"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := appointmentTestRouter(tt.service)
			response := httptest.NewRecorder()
			request := httptest.NewRequest(tt.method, tt.path, nil)

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

func appointmentTestRouter(service fakeAppointmentService) http.Handler {
	return apphttp.NewRouter(nil, nil, service, testLogger())
}

func testAppointment(request scheduling.CreateAppointmentRequest) scheduling.Appointment {
	return scheduling.Appointment{
		ID:            "99999999-9999-9999-9999-999999999999",
		CustomerID:    request.CustomerID,
		VehicleID:     request.VehicleID,
		DealershipID:  request.DealershipID,
		ServiceTypeID: request.ServiceTypeID,
		TechnicianID:  "66666666-6666-6666-6666-666666666661",
		ServiceBayID:  "88888888-8888-8888-8888-888888888881",
		StartTime:     request.StartTime,
		EndTime:       request.StartTime.Add(30 * time.Minute),
		Status:        scheduling.StatusConfirmed,
	}
}

func validCreateAppointmentJSON() string {
	return `{"customerId":"11111111-1111-1111-1111-111111111111","vehicleId":"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa1","dealershipId":"44444444-4444-4444-4444-444444444444","serviceTypeId":"55555555-5555-5555-5555-555555555551","startTime":"2026-05-04T14:00:00Z"}`
}

func mustParseTime(value string) time.Time {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		panic(err)
	}
	return parsed
}
