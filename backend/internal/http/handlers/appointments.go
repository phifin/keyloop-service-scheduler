package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"keyloop-service-scheduler/internal/scheduling"
)

type AppointmentService interface {
	Create(ctx context.Context, request scheduling.CreateAppointmentRequest) (scheduling.Appointment, error)
	List(ctx context.Context, filters scheduling.AppointmentFilters) ([]scheduling.AppointmentListItem, error)
	GetDetail(ctx context.Context, appointmentID string) (scheduling.AppointmentDetail, error)
	Cancel(ctx context.Context, appointmentID string) (scheduling.Appointment, error)
}

type AppointmentHandler struct {
	service AppointmentService
	logger  *slog.Logger
}

func NewAppointmentHandler(service AppointmentService, logger *slog.Logger) *AppointmentHandler {
	if logger == nil {
		logger = slog.Default()
	}

	return &AppointmentHandler{
		service: service,
		logger:  logger,
	}
}

type createAppointmentPayload struct {
	CustomerID    string `json:"customerId"`
	VehicleID     string `json:"vehicleId"`
	DealershipID  string `json:"dealershipId"`
	ServiceTypeID string `json:"serviceTypeId"`
	StartTime     string `json:"startTime"`
}

func (h *AppointmentHandler) Create(w http.ResponseWriter, r *http.Request) {
	if h.service == nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "appointment service is not configured")
		return
	}

	var payload createAppointmentPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "request body must be valid JSON")
		return
	}

	request, ok := parseCreateAppointmentRequest(w, payload)
	if !ok {
		return
	}

	h.logger.Info(
		"appointment_booking_started",
		"customer_id", request.CustomerID,
		"vehicle_id", request.VehicleID,
		"dealership_id", request.DealershipID,
		"service_type_id", request.ServiceTypeID,
		"start_time", request.StartTime.Format(time.RFC3339),
	)

	appointment, err := h.service.Create(r.Context(), request)
	if errors.Is(err, scheduling.ErrVehicleCustomerMismatch) {
		h.logger.Info("appointment_booking_failed", "reason", "vehicle_customer_mismatch")
		writeError(w, http.StatusBadRequest, "bad_request", "vehicleId must belong to customerId")
		return
	}
	if errors.Is(err, scheduling.ErrNotFound) {
		h.logger.Info("appointment_booking_failed", "reason", "not_found")
		writeError(w, http.StatusNotFound, "not_found", "customer, vehicle, dealership, or service type not found")
		return
	}
	if errors.Is(err, scheduling.ErrConflict) {
		h.logger.Info("appointment_booking_conflict", "dealership_id", request.DealershipID, "service_type_id", request.ServiceTypeID)
		writeResourceConflict(w)
		return
	}
	if err != nil {
		h.logger.Error("appointment_booking_failed", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to create appointment")
		return
	}

	h.logger.Info("appointment_created", "appointment_id", appointment.ID)
	writeJSON(w, http.StatusCreated, appointment)
}

func (h *AppointmentHandler) List(w http.ResponseWriter, r *http.Request) {
	if h.service == nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "appointment service is not configured")
		return
	}

	filters, ok := parseAppointmentFilters(w, r)
	if !ok {
		return
	}

	appointments, err := h.service.List(r.Context(), filters)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to load appointments")
		return
	}

	writeJSON(w, http.StatusOK, appointments)
}

func (h *AppointmentHandler) GetDetail(w http.ResponseWriter, r *http.Request) {
	if h.service == nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "appointment service is not configured")
		return
	}

	appointmentID := chi.URLParam(r, "appointmentId")
	if !validUUID(appointmentID) {
		writeError(w, http.StatusBadRequest, "bad_request", "appointmentId must be a valid UUID")
		return
	}

	appointment, err := h.service.GetDetail(r.Context(), appointmentID)
	if errors.Is(err, scheduling.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not_found", "appointment not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to load appointment")
		return
	}

	writeJSON(w, http.StatusOK, appointment)
}

func (h *AppointmentHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	if h.service == nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "appointment service is not configured")
		return
	}

	appointmentID := chi.URLParam(r, "appointmentId")
	if !validUUID(appointmentID) {
		writeError(w, http.StatusBadRequest, "bad_request", "appointmentId must be a valid UUID")
		return
	}

	appointment, err := h.service.Cancel(r.Context(), appointmentID)
	if errors.Is(err, scheduling.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not_found", "appointment not found")
		return
	}
	if errors.Is(err, scheduling.ErrCompletedAppointmentCannotCancel) {
		writeError(w, http.StatusConflict, "conflict", "completed appointments cannot be cancelled")
		return
	}
	if err != nil {
		h.logger.Error("appointment_booking_failed", "appointment_id", appointmentID, "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to cancel appointment")
		return
	}

	h.logger.Info("appointment_cancelled", "appointment_id", appointment.ID)
	writeJSON(w, http.StatusOK, appointment)
}

func parseCreateAppointmentRequest(w http.ResponseWriter, payload createAppointmentPayload) (scheduling.CreateAppointmentRequest, bool) {
	if payload.CustomerID == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "customerId is required")
		return scheduling.CreateAppointmentRequest{}, false
	}
	if !validUUID(payload.CustomerID) {
		writeError(w, http.StatusBadRequest, "bad_request", "customerId must be a valid UUID")
		return scheduling.CreateAppointmentRequest{}, false
	}

	if payload.VehicleID == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "vehicleId is required")
		return scheduling.CreateAppointmentRequest{}, false
	}
	if !validUUID(payload.VehicleID) {
		writeError(w, http.StatusBadRequest, "bad_request", "vehicleId must be a valid UUID")
		return scheduling.CreateAppointmentRequest{}, false
	}

	if payload.DealershipID == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "dealershipId is required")
		return scheduling.CreateAppointmentRequest{}, false
	}
	if !validUUID(payload.DealershipID) {
		writeError(w, http.StatusBadRequest, "bad_request", "dealershipId must be a valid UUID")
		return scheduling.CreateAppointmentRequest{}, false
	}

	if payload.ServiceTypeID == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "serviceTypeId is required")
		return scheduling.CreateAppointmentRequest{}, false
	}
	if !validUUID(payload.ServiceTypeID) {
		writeError(w, http.StatusBadRequest, "bad_request", "serviceTypeId must be a valid UUID")
		return scheduling.CreateAppointmentRequest{}, false
	}

	if payload.StartTime == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "startTime is required")
		return scheduling.CreateAppointmentRequest{}, false
	}
	startTime, err := time.Parse(time.RFC3339, payload.StartTime)
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "startTime must be a valid RFC3339 timestamp")
		return scheduling.CreateAppointmentRequest{}, false
	}

	return scheduling.CreateAppointmentRequest{
		CustomerID:    payload.CustomerID,
		VehicleID:     payload.VehicleID,
		DealershipID:  payload.DealershipID,
		ServiceTypeID: payload.ServiceTypeID,
		StartTime:     startTime,
	}, true
}

func parseAppointmentFilters(w http.ResponseWriter, r *http.Request) (scheduling.AppointmentFilters, bool) {
	var filters scheduling.AppointmentFilters

	if dealershipID := r.URL.Query().Get("dealershipId"); dealershipID != "" {
		if !validUUID(dealershipID) {
			writeError(w, http.StatusBadRequest, "bad_request", "dealershipId must be a valid UUID")
			return scheduling.AppointmentFilters{}, false
		}
		filters.DealershipID = &dealershipID
	}

	if status := r.URL.Query().Get("status"); status != "" {
		if !scheduling.IsValidAppointmentStatus(status) {
			writeError(w, http.StatusBadRequest, "bad_request", "status must be CONFIRMED, CANCELLED, or COMPLETED")
			return scheduling.AppointmentFilters{}, false
		}
		filters.Status = &status
	}

	return filters, true
}

func writeResourceConflict(w http.ResponseWriter) {
	writeErrorWithCode(
		w,
		http.StatusConflict,
		"conflict",
		"No qualified technician and service bay are available for the requested time range.",
		"RESOURCE_UNAVAILABLE",
	)
}

func validUUID(value string) bool {
	_, err := uuid.Parse(value)
	return err == nil
}
