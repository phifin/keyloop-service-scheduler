package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"keyloop-service-scheduler/internal/scheduling"
)

type AvailabilityChecker interface {
	Check(ctx context.Context, request scheduling.AvailabilityRequest) (scheduling.AvailabilityResult, error)
}

type AvailabilityHandler struct {
	service AvailabilityChecker
	logger  *slog.Logger
}

func NewAvailabilityHandler(service AvailabilityChecker, logger *slog.Logger) *AvailabilityHandler {
	if logger == nil {
		logger = slog.Default()
	}

	return &AvailabilityHandler{
		service: service,
		logger:  logger,
	}
}

func (h *AvailabilityHandler) Check(w http.ResponseWriter, r *http.Request) {
	if h.service == nil {
		h.logger.Error("availability_check_failed", "error", "availability service is not configured")
		writeError(w, http.StatusInternalServerError, "internal_error", "availability service is not configured")
		return
	}

	dealershipID, ok := requiredUUIDQuery(w, r, "dealershipId")
	if !ok {
		h.logger.Info("availability_check_failed", "reason", "invalid_dealership_id")
		return
	}

	serviceTypeID, ok := requiredUUIDQuery(w, r, "serviceTypeId")
	if !ok {
		h.logger.Info("availability_check_failed", "dealership_id", dealershipID, "reason", "invalid_service_type_id")
		return
	}

	startTime, ok := requiredRFC3339Query(w, r, "startTime")
	if !ok {
		h.logger.Info("availability_check_failed", "dealership_id", dealershipID, "service_type_id", serviceTypeID, "reason", "invalid_start_time")
		return
	}

	h.logger.Info(
		"availability_check_started",
		"dealership_id", dealershipID,
		"service_type_id", serviceTypeID,
		"start_time", startTime.Format(time.RFC3339),
	)

	result, err := h.service.Check(r.Context(), scheduling.AvailabilityRequest{
		DealershipID:  dealershipID,
		ServiceTypeID: serviceTypeID,
		StartTime:     startTime,
	})
	if errors.Is(err, scheduling.ErrNotFound) {
		h.logger.Info(
			"availability_check_failed",
			"dealership_id", dealershipID,
			"service_type_id", serviceTypeID,
			"reason", "not_found",
		)
		writeError(w, http.StatusNotFound, "not_found", "dealership or service type not found")
		return
	}
	if err != nil {
		h.logger.Error(
			"availability_check_failed",
			"dealership_id", dealershipID,
			"service_type_id", serviceTypeID,
			"error", err,
		)
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to check availability")
		return
	}

	h.logger.Info(
		"availability_check_completed",
		"dealership_id", dealershipID,
		"service_type_id", serviceTypeID,
		"available", result.Available,
		"available_technician_count", len(result.AvailableTechnicians),
		"available_service_bay_count", len(result.AvailableServiceBays),
	)

	writeJSON(w, http.StatusOK, result)
}

func requiredRFC3339Query(w http.ResponseWriter, r *http.Request, name string) (time.Time, bool) {
	value := r.URL.Query().Get(name)
	if value == "" {
		writeError(w, http.StatusBadRequest, "bad_request", name+" is required")
		return time.Time{}, false
	}

	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", name+" must be a valid RFC3339 timestamp")
		return time.Time{}, false
	}

	return parsed, true
}
